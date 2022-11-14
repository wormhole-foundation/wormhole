package finalizers

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	ethClient "github.com/ethereum/go-ethereum/ethclient"

	"go.uber.org/zap"
)

// NeonFinalizer implements the finality check for Neon. The Neon block number is actually the Solana slot number.
// Blocks on Neon should not be considered finalized until that slot is finalized on Solana. Confirmed this with the
// Neon team on 11/12/2022. Also confirmed that they do not have a websocket interface so we need to poll for log events.
type NeonFinalizer struct {
	logger      *zap.Logger
	connector   connectors.Connector
	l1Finalizer interfaces.L1Finalizer
}

func NewNeonFinalizer(logger *zap.Logger, connector connectors.Connector, client *ethClient.Client, l1Finalizer interfaces.L1Finalizer) *NeonFinalizer {
	return &NeonFinalizer{
		logger:      logger,
		connector:   connector,
		l1Finalizer: l1Finalizer,
	}
}

// IsBlockFinalized compares the number of the Neon block with the latest finalized block on Solana.
func (f *NeonFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
	latestL1Block := f.l1Finalizer.GetLatestFinalizedBlockNumber()
	if latestL1Block == 0 {
		// This happens on start up.
		return false, nil
	}

	isFinalized := block.Number.Uint64() <= latestL1Block
	return isFinalized, nil
}
