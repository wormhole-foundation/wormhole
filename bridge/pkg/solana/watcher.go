package ethereum

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"

	agentv1 "github.com/certusone/wormhole/bridge/pkg/proto/agent/v1"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type (
	SolanaBridgeWatcher struct {
		url string

		lockChan chan *common.ChainLock
		vaaChan  chan *vaa.VAA
	}
)

func NewSolanaBridgeWatcher(url string, lockEvents chan *common.ChainLock, vaaQueue chan *vaa.VAA) *SolanaBridgeWatcher {
	return &SolanaBridgeWatcher{url: url, lockChan: lockEvents, vaaChan: vaaQueue}
}

// TODO: document/deduplicate
func padAddress(address eth_common.Address) vaa.Address {
	paddedAddress := eth_common.LeftPadBytes(address[:], 32)

	addr := vaa.Address{}
	copy(addr[:], paddedAddress)

	return addr
}

func (e *SolanaBridgeWatcher) Run(ctx context.Context) error {
	timeout, _ := context.WithTimeout(ctx, 15*time.Second)
	conn, err := grpc.DialContext(timeout, e.url, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to dial agent at %s: %w", e.url, err)
	}
	defer conn.Close()

	c := agentv1.NewAgentClient(conn)

	errC := make(chan error)
	logger := supervisor.Logger(ctx)

	// Subscribe to new token lockups
	tokensLockedSub, err := c.WatchLockups(ctx, &agentv1.WatchLockupsRequest{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to token lockup events: %w", err)
	}

	go func() {
		logger.Info("watching for on-chain events")

		for {
			ev, err := tokensLockedSub.Recv()
			if err != nil {
				errC <- err
				return
			}

			switch event := ev.Event.(type) {
			case *agentv1.LockupEvent_New:
				lock := &common.ChainLock{
					TxHash:        eth_common.HexToHash(ev.LockupAddress),
					Timestamp:     time.Time{}, // FIXME
					Nonce:         event.New.Nonce,
					SourceAddress: padAddress(eth_common.BytesToAddress(event.New.SourceAddress)),
					TargetAddress: padAddress(eth_common.BytesToAddress(event.New.TargetAddress)),
					SourceChain:   vaa.ChainIDSolana,
					TargetChain:   vaa.ChainID(event.New.TargetChain),
					TokenChain:    vaa.ChainID(event.New.TokenChain),
					TokenAddress:  padAddress(eth_common.BytesToAddress(event.New.TokenAddress)),
					Amount:        new(big.Int).SetBytes(event.New.Amount),
				}

				e.lockChan <- lock
				logger.Info("found new lockup transaction", zap.String("lockup_address", ev.LockupAddress))
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case v := <-e.vaaChan:
				vaaBytes, err := v.Marshal()
				if err != nil {
					panic(err)
				}

				// Calculate digest so we can log it (TODO: refactor to vaa method? we do this in different places)
				m, err := v.SigningMsg()
				if err != nil {
					panic(err)
				}
				h := hex.EncodeToString(m.Bytes())

				timeout, _ := context.WithTimeout(ctx, 15*time.Second)
				res, err := c.SubmitVAA(timeout, &agentv1.SubmitVAARequest{Vaa: vaaBytes})
				if err != nil {
					logger.Error("failed to submit VAA", zap.Error(err), zap.String("digest", h))
					break
				}

				logger.Info("submitted VAA",
					zap.String("tx_sig", res.Signature), zap.String("digest", h))
			}
		}
	}()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}
