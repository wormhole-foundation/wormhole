package finalizers

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/interfaces"

	ethClient "github.com/ethereum/go-ethereum/ethclient"

	"go.uber.org/zap"
)

// ArbitrumFinalizer implements the finality check for Arbitrum.
// Arbitrum blocks should not be considered finalized until they are finalized on Ethereum.

type ArbitrumFinalizer struct {
	logger      *zap.Logger
	connector   connectors.Connector
	l1Finalizer interfaces.L1Finalizer
}

func NewArbitrumFinalizer(logger *zap.Logger, connector connectors.Connector, client *ethClient.Client, l1Finalizer interfaces.L1Finalizer) *ArbitrumFinalizer {
	return &ArbitrumFinalizer{
		logger:      logger,
		connector:   connector,
		l1Finalizer: l1Finalizer,
	}
}

// IsBlockFinalized compares the number of the L1 block containing the Arbitrum block with the latest finalized block on Ethereum.
func (a *ArbitrumFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
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
