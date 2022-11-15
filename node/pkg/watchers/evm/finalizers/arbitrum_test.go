package finalizers

import (
	"context"
	"math/big"
	"testing"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestArbitrumErrorReturnedIfBlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewArbitrumFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}

func TestArbitrumErrorReturnedIfL1BlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewArbitrumFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: nil,
	}

	_, err := finalizer.IsBlockFinalized(ctx, block)
	require.Error(t, err)
}

func TestArbitrumNotFinalizedIfNoFinalizedL1BlockYet(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := interfaces.MockL1Finalizer{}

	finalizer := NewArbitrumFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: big.NewInt(225),
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}

func TestArbitrumNotFinalizedWhenFinalizedL1IsLessThanTargetL1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 225}

	finalizer := NewArbitrumFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(127),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: big.NewInt(226),
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}

func TestArbitrumIsFinalizedWhenFinalizedL1IsEqualsTargetL1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 225}

	finalizer := NewArbitrumFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: big.NewInt(225),
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestArbitrumIsFinalizedWhenFinalizedL1IsGreaterThanTargetL1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 227}

	finalizer := NewArbitrumFinalizer(logger, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number:        big.NewInt(125),
		Hash:          ethCommon.Hash{},
		L1BlockNumber: big.NewInt(225),
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}
