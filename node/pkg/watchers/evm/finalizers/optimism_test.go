package finalizers

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"

	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

type (
	mockCtcCaller struct {
		mutex               sync.Mutex
		totalElements       []*big.Int
		lastBlockNumbers    []*big.Int
		totalElementsErr    error
		lastBlockNumbersErr error
	}
)

// SetTotalElements takes an array of big int pointers that represent the L2 block numbers to be returned by GetTotalElements()
func (m *mockCtcCaller) SetTotalElements(totalElements []*big.Int) {
	m.mutex.Lock()
	m.totalElements = totalElements
	m.mutex.Unlock()
}

// SetLastBlockNumber takes an array of big int pointers that represent the L1 block numbers to be returned by GetLastBlockNumber()
func (m *mockCtcCaller) SetLastBlockNumbers(lastBlockNumbers []*big.Int) {
	m.mutex.Lock()
	m.lastBlockNumbers = lastBlockNumbers
	m.mutex.Unlock()
}

// SetTotalElementsError takes an error (or nil) which will be returned on the next call to GetTotalElements. The error will persist until cleared.
func (m *mockCtcCaller) SetTotalElementsError(err error) {
	m.mutex.Lock()
	m.totalElementsErr = err
	m.mutex.Unlock()
}

// SetLastBlockNumber takes an error (or nil) which will be returned on the next call to GetLastBlockNumber. The error will persist until cleared.
func (m *mockCtcCaller) SetLastBlockNumberError(err error) {
	m.mutex.Lock()
	m.lastBlockNumbersErr = err
	m.mutex.Unlock()
}

func (m *mockCtcCaller) GetTotalElements(opts *ethBind.CallOpts) (result *big.Int, err error) {
	m.totalElements, result, err = m.getResult(m.totalElements, m.totalElementsErr)
	return
}

func (m *mockCtcCaller) GetLastBlockNumber(opts *ethBind.CallOpts) (result *big.Int, err error) {
	m.lastBlockNumbers, result, err = m.getResult(m.lastBlockNumbers, m.lastBlockNumbersErr)
	return
}

func (m *mockCtcCaller) getResult(resultsIn []*big.Int, errIn error) (resultsOut []*big.Int, result *big.Int, err error) {
	for {
		m.mutex.Lock()
		// If they set the error, return that immediately.
		if errIn != nil {
			err = errIn
			break
		}

		// If there are pending results, return the first one.
		if len(resultsIn) != 0 {
			result = resultsIn[0]
			resultsOut = resultsIn[1:]
			break
		}

		// If we don't have any results, sleep and try again.
		m.mutex.Unlock()
		time.Sleep(1 * time.Millisecond)
	}

	m.mutex.Unlock()
	return
}

func NewOptimismFinalizerForTest(
	ctx context.Context,
	logger *zap.Logger,
	l1Finalizer interfaces.L1Finalizer,
	ctcCaller ctcCallerIntf,
) *OptimismFinalizer {
	finalizer := &OptimismFinalizer{
		logger:                 logger,
		l1Finalizer:            l1Finalizer,
		latestFinalizedL2Block: big.NewInt(0),
		finalizerMapping:       make([]RollupInfo, 0),
		ctcCaller:              ctcCaller,
	}

	return finalizer
}

func TestOptimismErrorReturnedIfBlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 125}
	ctcCaller := &mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, ctcCaller)
	require.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	assert.Error(t, err)
}

func TestOptimismNotFinalizedIfNoFinalizedL1BlockYet(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

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
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 7954401}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

	ctcCaller.SetLastBlockNumbers([]*big.Int{big.NewInt(7954402)})
	ctcCaller.SetTotalElements([]*big.Int{big.NewInt(127)})

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
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 7954402}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

	ctcCaller.SetLastBlockNumbers([]*big.Int{big.NewInt(7954402)})
	ctcCaller.SetTotalElements([]*big.Int{big.NewInt(125)})

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
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 7954403}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	ctcCaller.SetLastBlockNumbers([]*big.Int{big.NewInt(7954402)})
	ctcCaller.SetTotalElements([]*big.Int{big.NewInt(125)})

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestOptimismL2BlockNumberMustNotGoBackwards(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 7954400}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	ctcCaller.SetLastBlockNumbers([]*big.Int{big.NewInt(7954402), big.NewInt(7954403)})
	ctcCaller.SetTotalElements([]*big.Int{big.NewInt(124), big.NewInt(123)})

	isFinalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	require.Equal(t, false, isFinalized)

	_, err = finalizer.IsBlockFinalized(ctx, block)
	require.Error(t, err)
}

func TestOptimismGetTotalElementsRpcError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 125}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

	ctcCaller.SetTotalElementsError(fmt.Errorf("RPC failed"))
	ctcCaller.SetLastBlockNumbers([]*big.Int{big.NewInt(7954402)})

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	_, err := finalizer.IsBlockFinalized(ctx, block)
	require.Error(t, err)
}

func TestOptimismGetLastBlockNumberRpcError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	l1Finalizer := mockL1Finalizer{LatestFinalizedBlockNumber: 125}
	ctcCaller := mockCtcCaller{}

	finalizer := NewOptimismFinalizerForTest(ctx, logger, &l1Finalizer, &ctcCaller)
	require.NotNil(t, finalizer)

	ctcCaller.SetLastBlockNumberError(fmt.Errorf("RPC failed"))
	ctcCaller.SetTotalElements([]*big.Int{big.NewInt(125)})

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.Hash{},
	}

	_, err := finalizer.IsBlockFinalized(ctx, block)
	require.Error(t, err)
}
