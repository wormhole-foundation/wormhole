package aztec

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// WatcherFactory creates and initializes a new Aztec watcher
type WatcherFactory struct {
	// Configuration values passed in from the main application
	NetworkID string
	ChainID   vaa.ChainID
}

// NewWatcherRunnable creates a new Aztec watcher runnable
func NewWatcherRunnable(
	chainID vaa.ChainID,
	networkID string,
	rpcURL string,
	contractAddress string,
	msgC chan<- *common.MessagePublication,
	_ <-chan *gossipv1.ObservationRequest,
) supervisor.Runnable {
	// Create a runnable
	runnable := supervisor.Runnable(func(ctx context.Context) error {
		logger := supervisor.Logger(ctx)
		logger.Info("Starting Aztec watcher",
			zap.String("rpc", rpcURL),
			zap.String("contract", contractAddress))

		// Create the readiness component
		readinessSync := common.MustConvertChainIdToReadinessSyncing(chainID)

		// Create default config
		config := DefaultConfig(chainID, networkID, rpcURL, contractAddress)

		// Create the block fetcher
		blockFetcher, err := NewAztecBlockFetcher(ctx, rpcURL, logger)
		if err != nil {
			return fmt.Errorf("failed to create block fetcher: %v", err)
		}

		// Create L1Verifier - this is used internally by the watcher
		l1Verifier, err := NewAztecFinalityVerifier(ctx, rpcURL, logger.Named("aztec_finality"))
		if err != nil {
			return fmt.Errorf("failed to create L1Verifier: %v", err)
		}

		// Create the observation manager
		observationManager := NewObservationManager(networkID, logger)

		// Create the watcher
		watcher := NewWatcher(
			config,
			blockFetcher,
			l1Verifier,
			observationManager,
			msgC,
			logger,
		)

		// Signal initialization complete
		readiness.SetReady(readinessSync)
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		// Run the watcher
		return watcher.Run(ctx)
	})

	return runnable
}

// Create a factory instance that can be used in the main application
func NewAztecWatcherFactory(networkID string, chainID vaa.ChainID) *WatcherFactory {
	return &WatcherFactory{
		NetworkID: networkID,
		ChainID:   chainID,
	}
}
