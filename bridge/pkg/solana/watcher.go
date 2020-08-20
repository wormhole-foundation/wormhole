package ethereum

import (
	"context"
	"fmt"
	"time"

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

	//// Subscribe to new token lockups
	//tokensLockedSub, err := c.WatchLockups(ctx, &agentv1.WatchLockupsRequest{})
	//if err != nil {
	//	return fmt.Errorf("failed to subscribe to token lockup events: %w", err)
	//}
	//
	//go func() {
	//	// TODO: does this properly terminate on ctx cancellation?
	//	ev, err := tokensLockedSub.Recv()
	//	for ; err == nil; ev, err = tokensLockedSub.Recv() {
	//		switch event := ev.Event.(type) {
	//		case *agentv1.LockupEvent_New:
	//			lock := &common.ChainLock{
	//				TxHash:        eth_common.HexToHash(ev.TxHash),
	//				SourceAddress: event.New.SourceAddress,
	//				TargetAddress: event.New.TargetAddress,
	//				SourceChain:   vaa.ChainIDSolana,
	//				TargetChain:   vaa.ChainID(event.New.TargetChain),
	//				TokenChain:    vaa.ChainID(event.New.TokenChain),
	//				TokenAddress:  event.New.TokenAddress,
	//				Amount:        new(big.Int).SetBytes(event.New.Amount),
	//			}
	//
	//			logger.Info("found new lockup transaction", zap.String("tx", ev.TxHash))
	//			e.pendingLocksGuard.Lock()
	//			e.pendingLocks[ev.BlockHash] = &pendingLock{
	//				lock: lock,
	//			}
	//			e.pendingLocksGuard.Unlock()
	//		}
	//	}
	//
	//	if err != io.EOF {
	//		errC <- err
	//	}
	//}()

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

				timeout, _ := context.WithTimeout(ctx, 15*time.Second)
				res, err := c.SubmitVAA(timeout, &agentv1.SubmitVAARequest{Vaa: vaaBytes})
				if err != nil {
					logger.Error("failed to submit VAA", zap.Error(err))
					break
				}

				logger.Info("submitted VAA",
					zap.String("signature", res.Signature))
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
