package ethereum

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func (e *SolanaBridgeWatcher) Run(ctx context.Context) error {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
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
				errC <- fmt.Errorf("failed to receive message from agent: %w", err)
				return
			}

			switch event := ev.Event.(type) {
			case *agentv1.LockupEvent_New:
				logger.Debug("received lockup event",
					zap.Any("event", ev))

				lock := &common.ChainLock{
					TxHash:        eth_common.HexToHash(ev.LockupAddress),
					Timestamp:     time.Unix(int64(ev.Time), 0),
					Nonce:         event.New.Nonce,
					SourceChain:   vaa.ChainIDSolana,
					TargetChain:   vaa.ChainID(event.New.TargetChain),
					TokenChain:    vaa.ChainID(event.New.TokenChain),
					TokenDecimals: uint8(event.New.TokenDecimals),
					Amount:        new(big.Int).SetBytes(event.New.Amount),
				}
				copy(lock.TokenAddress[:], event.New.TokenAddress)
				copy(lock.SourceAddress[:], event.New.SourceAddress)
				copy(lock.TargetAddress[:], event.New.TargetAddress)

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

				timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
				res, err := c.SubmitVAA(timeout, &agentv1.SubmitVAARequest{Vaa: vaaBytes})
				cancel()
				if err != nil {
					st, ok := status.FromError(err)
					if !ok {
						panic("err not a status")
					}

					// For transient errors, we can put the VAA back into the queue such that it can
					// be retried after the runnable has been rescheduled.
					switch st.Code() {
					case
						// Our context was cancelled, likely because the watcher stream died.
						codes.Canceled,
						// The agent encountered a transient error, likely node unavailability.
						codes.Unavailable,
						codes.Aborted:

						logger.Error("transient error, requeuing VAA", zap.Error(err), zap.String("digest", h))

						// Tombstone goroutine
						go func(v *vaa.VAA) {
							time.Sleep(10 * time.Second)
							e.vaaChan <- v
						}(v)

					case codes.Internal:
						// This VAA has already been executed on chain, successfully or not.
						// TODO: dissect InstructionError in agent and convert this to the proper gRPC code
						if strings.Contains(st.Message(), "AlreadyExists") {
							logger.Info("VAA already submitted on-chain, ignoring", zap.Error(err), zap.String("digest", h))
							break
						}

						fallthrough
					default:
						logger.Error("error submitting VAA", zap.Error(err), zap.String("digest", h))
					}

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
