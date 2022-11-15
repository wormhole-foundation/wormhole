package finalizers

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestOptimismErrorReturnedIfBlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}

func TestOptimismNotFinalizedIfNoFinalizedL1BlockYet(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}

func TestOptimismNotFinalizedWhenFinalizedL1IsLessThanTargetL1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 7954401}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	baseConnector.SetResults([]string{
		`{"mode":"verifier","syncing":false,"ethContext":{"blockNumber":7954402,"timestamp":1668466522},"rollupContext":{"index":2699324,"queueIndex":20022,"verifiedIndex":127}}`,
	})

	block := &connectors.NewBlock{
		Number: big.NewInt(127),
		Hash:   ethCommon.Hash{},
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}

func TestOptimismNotFinalizedWhenFinalizedL1IsEqualsTargetL1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 7954402}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	baseConnector.SetResults([]string{
		`{"mode":"verifier","syncing":false,"ethContext":{"blockNumber":7954402,"timestamp":1668466522},"rollupContext":{"index":2699324,"queueIndex":20022,"verifiedIndex":125}}`,
	})

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestOptimismIsFinalizedWhenFinalizedL1IsGreaterThanTargetL1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 7954403}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	baseConnector.SetResults([]string{
		`{"mode":"verifier","syncing":false,"ethContext":{"blockNumber":7954402,"timestamp":1668466522},"rollupContext":{"index":2699324,"queueIndex":20022,"verifiedIndex":125}}`,
	})

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestOptimismVerifierIndexMustBeNonZero(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 7954403}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	baseConnector.SetResults([]string{
		`{"mode":"verifier","syncing":false,"ethContext":{"blockNumber":7954402,"timestamp":1668466522},"rollupContext":{"index":2699324,"queueIndex":20022,"verifiedIndex":0}}`,
	})

	_, err := finalizer.IsBlockFinalized(ctx, block)
	require.Error(t, err)
}

func TestOptimismRpcError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}
	l1Finalizer := interfaces.MockL1Finalizer{LatestFinalizedBlockNumber: 125}

	finalizer := NewOptimismFinalizer(ctx, logger, &baseConnector, &l1Finalizer)
	assert.NotNil(t, finalizer)

	baseConnector.SetError(fmt.Errorf("RPC failed"))

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	_, err := finalizer.IsBlockFinalized(ctx, block)
	require.Error(t, err)
}
