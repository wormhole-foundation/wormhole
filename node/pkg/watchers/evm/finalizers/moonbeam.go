package finalizers

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	"go.uber.org/zap"
)

// MoonbeamFinalizer implements the finality check for Moonbeam.
// Moonbeam can publish blocks before they are marked final. This means we need to sit on the block until a special "is finalized"
// query returns true. The assumption is that every block number will eventually be published and finalized, it's just that the contents
// of the block (and therefore the hash) might change if there is a rollback.
type MoonbeamFinalizer struct {
	logger    *zap.Logger
	connector connectors.Connector
}

func NewMoonbeamFinalizer(logger *zap.Logger, connector connectors.Connector) *MoonbeamFinalizer {
	return &MoonbeamFinalizer{
		logger:    logger,
		connector: connector,
	}
}

func (m *MoonbeamFinalizer) IsBlockFinalized(ctx context.Context, block *connectors.NewBlock) (bool, error) {
	var finalized bool
	err := m.connector.RawCallContext(ctx, &finalized, "moon_isBlockFinalized", block.Hash.Hex())
	if err != nil {
		m.logger.Error("failed to check for finality", zap.String("eth_network", m.connector.NetworkName()), zap.Error(err))
		return false, err
	}

	return finalized, nil
}
