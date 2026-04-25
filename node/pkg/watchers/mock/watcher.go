package mock

import (
	"context"
	"errors"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type Watcher struct {
	chainID  vaa.ChainID
	msgC     chan<- *common.MessagePublication
	obsvReqC <-chan *gossipv1.ObservationRequest
	setC     chan<- *common.GuardianSet
	config   *WatcherConfig
}

var _ watchers.Watcher = (*Watcher)(nil)

func (w *Watcher) ChainID() vaa.ChainID {
	return w.chainID
}

func (w *Watcher) Validate(req *gossipv1.ObservationRequest) (watchers.ValidObservation, error) {
	return watchers.ValidateObservationRequest(req, w.chainID)
}

func (w *Watcher) PublishMessage(msg *common.MessagePublication) error {
	if msg == nil {
		return errors.New("message publication is nil")
	}

	w.msgC <- msg //nolint:channelcheck // The channel to the processor is buffered and shared across chains, if it backs up we should stop processing new observations
	return nil
}

func (w *Watcher) PublishReobservation(observation watchers.ValidObservation, msg *common.MessagePublication) error {
	if err := watchers.ValidateReobservedMessage(observation, msg); err != nil {
		return err
	}

	msg.IsReobservation = true
	return w.PublishMessage(msg)
}

func (w *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)
	supervisor.Signal(ctx, supervisor.SignalHealthy)

	logger.Info("Mock Watcher running.")

	for {
		select {
		case <-ctx.Done():
			logger.Info("Mock Watcher shutting down.")
			return nil
		case observation := <-w.config.MockObservationC:
			logger.Info("message observed", observation.ZapFields(zap.String("digest", observation.CreateDigest()))...)
			if err := w.PublishMessage(observation); err != nil {
				logger.Error("failed to publish message", zap.Error(err))
			}
		case gs := <-w.config.MockSetC:
			w.setC <- gs //nolint:channelcheck // Will only block this mock watcher
		case o := <-w.obsvReqC:
			validatedObservation, err := w.Validate(o)
			if err != nil {
				watchers.LogInvalidObservationRequest(logger, o, err)
				continue
			}
			hash := eth_common.BytesToHash(validatedObservation.TxHash())
			logger.Info("received observation request", validatedObservation.ZapFields(zap.String("log_msg_type", "obsv_req_received"), zap.String("tx_hash", hash.Hex()))...)
			msg, ok := w.config.ObservationDb[hash]
			if ok {
				msg2 := *msg
				if err := w.PublishReobservation(validatedObservation, &msg2); err != nil {
					logger.Error("failed to publish reobservation", zap.Error(err))
				}
			}
		}
	}
}

func NewWatcherRunnable(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	setC chan<- *common.GuardianSet,
	c *WatcherConfig,
) supervisor.Runnable {
	w := &Watcher{chainID: c.ChainID, msgC: msgC, obsvReqC: obsvReqC, setC: setC, config: c}
	return w.Run
}
