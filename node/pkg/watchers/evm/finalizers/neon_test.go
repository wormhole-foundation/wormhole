package finalizers

import (
	"context"
	"math/big"
	"testing"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

// mockL1Finalizer implements the L1Finalizer interface for testing purposes.
type mockL1Finalizer struct {
	LatestFinalizedBlockNumber uint64
}

func (m *mockL1Finalizer) GetLatestFinalizedBlockNumber() uint64 {
	return m.LatestFinalizedBlockNumber
}

func TestNeonErrorReturnedIfBlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewNeonFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}

func TestNeonNotFinalizedIfNoFinalizedL1BlockYet(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{}

	finalizer := NewNeonFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: nil,
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}

func TestNeonNotFinalizedWhenL1IsLessThanL2(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewNeonFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(127),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: nil,
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}

func TestNeonIsFinalizedWhenL1EqualsL2(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewNeonFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: nil,
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestNeonIsFinalizedWhenL1GreaterThanL2(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 127}

	finalizer := NewNeonFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: nil,
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}
