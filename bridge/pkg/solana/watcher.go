package ethereum

import (
	"context"
	"fmt"
	agentv1 "github.com/certusone/wormhole/bridge/pkg/proto/agent/v1"
	"google.golang.org/grpc"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type (
	SolanaBridgeWatcher struct {
		url string

		pendingLocks      map[string]*pendingLock
		pendingLocksGuard sync.Mutex

		lockChan chan *common.ChainLock
		setChan  chan *common.GuardianSet
		vaaChan  chan *vaa.VAA
	}

	pendingLock struct {
		lock *common.ChainLock
	}
)

func NewSolanaBridgeWatcher(url string, lockEvents chan *common.ChainLock, setEvents chan *common.GuardianSet, vaaQueue chan *vaa.VAA) *SolanaBridgeWatcher {
	return &SolanaBridgeWatcher{url: url, lockChan: lockEvents, setChan: setEvents, pendingLocks: map[string]*pendingLock{}, vaaChan: vaaQueue}
}

func (e *SolanaBridgeWatcher) Run(ctx context.Context) error {
	conn, err := grpc.Dial(e.url)
	if err != nil {
		return fmt.Errorf("failed to dial agent: %w", err)
	}
	c := agentv1.NewAgentClient(conn)

	errC := make(chan error)
	logger := supervisor.Logger(ctx)
	//// Subscribe to new token lockups
	//tokensLockedSub, err := c.WatchLockups(ctx, &agentv1.WatchLockupsRequest{})
	//if err != nil {
	//	return fmt.Errorf("failed to subscribe to token lockup events: %w", err)
	//}
	//
	//
	//go func() {
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
		for v := range e.vaaChan {
			vaaBytes, err := v.Marshal()
			if err != nil {
				logger.Error("failed to marshal VAA", zap.Any("vaa", v), zap.Error(err))
				continue
			}

			timeout, _ := context.WithTimeout(ctx, 15*time.Second)
			res, err := c.SubmitVAA(timeout, &agentv1.SubmitVAARequest{Vaa: vaaBytes})
			if err != nil {
				errC <- fmt.Errorf("failed to submit VAA: %w", err)
				return
			}

			logger.Debug("submitted VAA", zap.String("signature", res.Signature))
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
