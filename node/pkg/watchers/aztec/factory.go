package aztec

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// WatcherFactory creates and initializes a new Aztec watcher
type WatcherFactory struct {
	// Configuration values passed in from the main application
	NetworkID string
	ChainID   vaa.ChainID
}

// NewWatcherFromConfig creates a new Aztec watcher from config values
func NewWatcherFromConfig(
	chainID vaa.ChainID,
	networkID string,
	rpcURL string,
	contractAddress string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) (interfaces.L1Finalizer, supervisor.Runnable) {
	// Create a logger
	logger := zap.L().Named("aztec")

	// Create a shared L1Verifier instance
	l1Verifier, err := NewAztecFinalityVerifier(rpcURL, logger.Named("finality"))
	if err != nil {
		// Log error but continue - we'll retry in the runnable
		logger.Error("Failed to create L1Verifier at startup, will retry in runnable", zap.Error(err))
	}

	// Create a runnable that uses the L1Verifier
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

		// If L1Verifier creation failed earlier, create it now
		if l1Verifier == nil {
			var initErr error
			l1Verifier, initErr = NewAztecFinalityVerifier(rpcURL, logger.Named("aztec_finality"))
			if initErr != nil {
				return fmt.Errorf("failed to create L1Verifier: %v", initErr)
			}
		} else if l1v, ok := l1Verifier.(*aztecFinalityVerifier); ok {
			// Update the logger in the existing L1Verifier
			l1v.logger = logger.Named("aztec_finality")
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

	return l1Verifier, runnable
}

// Create a factory instance that can be used in the main application
func NewAztecWatcherFactory(networkID string, chainID vaa.ChainID) *WatcherFactory {
	return &WatcherFactory{
		NetworkID: networkID,
		ChainID:   chainID,
	}
}
