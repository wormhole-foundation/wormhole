package algorand

import (
	"context"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Algorand blockchain and reporting new transactions to the contract
	Watcher struct {
		urlRPC   string
		urlToken string
		contract string

		msgChan chan *common.MessagePublication
		setChan chan *common.GuardianSet
	}
)

// NewWatcher creates a new Algorand contract watcher
func NewWatcher(urlRPC string, urlToken string, contract string, lockEvents chan *common.MessagePublication, setEvents chan *common.GuardianSet) *Watcher {
	return &Watcher{urlRPC: urlRPC, urlToken: urlToken, contract: contract, msgChan: lockEvents, setChan: setEvents}
}

func (e *Watcher) Run(ctx context.Context) error {
	logger := supervisor.Logger(ctx)
	logger.Info("connecting to algorand RPC", zap.String("url", e.urlRPC))

	readiness.SetReady(common.ReadinessAlgorandSyncing)

	select {}
}
