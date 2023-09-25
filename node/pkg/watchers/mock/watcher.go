package mock

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	eth_common "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

func NewWatcherRunnable(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	setC chan<- *common.GuardianSet,
	c *WatcherConfig,
) supervisor.Runnable {
	return func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		if c.L1FinalizerRequired != "" && c.l1Finalizer == nil {
			logger.Fatal("Mock watcher: L1FinalizerRequired but not set.")
		}

		logger.Info("Mock Watcher running.")

		for {
			select {
			case <-ctx.Done():
				logger.Info("Mock Watcher shutting down.")
				return nil
			case observation := <-c.MockObservationC:
				logger.Info("message observed", observation.ZapFields(zap.String("digest", observation.CreateDigest()))...)
				msgC <- observation
			case gs := <-c.MockSetC:
				setC <- gs
			case o := <-obsvReqC:
				hash := eth_common.BytesToHash(o.TxHash)
				logger.Info("Received obsv request", zap.String("log_msg_type", "obsv_req_received"), zap.String("tx_hash", hash.Hex()))
				msg, ok := c.ObservationDb[hash]
				if ok {
					msg2 := *msg
					msg2.IsReobservation = true
					msgC <- &msg2
				}
			}
		}
	}
}

type MockL1Finalizer struct{}

func (f MockL1Finalizer) GetLatestFinalizedBlockNumber() uint64 {
	return 0
}
