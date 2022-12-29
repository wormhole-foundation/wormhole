package finalizers

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	"go.uber.org/zap"
)

// ArbitrumFinalizer implements the finality check for Arbitrum.
// Arbitrum blocks should not be considered finalized until they are finalized on Ethereum.

type ArbitrumFinalizer struct {
	logger      *zap.Logger
	l1Finalizer interfaces.L1Finalizer
}

func NewArbitrumFinalizer(logger *zap.Logger, l1Finalizer interfaces.L1Finalizer) *ArbitrumFinalizer {
	return &ArbitrumFinalizer{
		logger:      logger,
		l1Finalizer: l1Finalizer,
	}
}

// IsBlockFinalized compares the number of the L1 block containing the Arbitrum block with the latest finalized block on Ethereum.
func (a *ArbitrumFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
	if block == nil {
		return false, fmt.Errorf("block is nil")
	}

	if block.L1BlockNumber == nil {
		return false, fmt.Errorf("l1 block number is nil")
	}

	latestL1Block := a.l1Finalizer.GetLatestFinalizedBlockNumber()
	if latestL1Block == 0 {
		// This happens on start up.
		return false, nil
	}

	isFinalized := block.L1BlockNumber.Uint64() <= latestL1Block
	return isFinalized, nil
}
