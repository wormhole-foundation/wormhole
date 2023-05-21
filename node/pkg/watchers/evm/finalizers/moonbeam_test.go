package finalizers

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethEvent "github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

type moonbeamMockConnector struct {
	isFinalized string
	err         error
}

func (e *moonbeamMockConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) (err error) {
	if method != "moon_isBlockFinalized" {
		panic("method not implemented by moonbeamMockConnector")
	}

	err = json.Unmarshal([]byte(e.isFinalized), &result)
	return
}

func (e *moonbeamMockConnector) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	panic("method not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) NetworkName() string {
	return "moonbeamMockConnector"
}

func (e *moonbeamMockConnector) ContractAddress() ethCommon.Address {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	panic("not implemented by moonbeamMockConnector")
}

func (e *moonbeamMockConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *connectors.NewBlock) (ethereum.Subscription, error) {
	panic("not implemented by moonbeamMockConnector")
}

func TestMoonbeamErrorReturnedIfBlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := moonbeamMockConnector{isFinalized: "true", err: nil}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}

func TestMoonbeamBlockNotFinalized(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := moonbeamMockConnector{isFinalized: "false", err: nil}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.HexToHash("0x1076cd8c207f31e1638b37bb358c458f216f5451f06e2ccb4eb9db66ad669f30"),
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}
func TestMoonbeamBlockIsFinalized(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := moonbeamMockConnector{isFinalized: "true", err: nil}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.HexToHash("0x1076cd8c207f31e1638b37bb358c458f216f5451f06e2ccb4eb9db66ad669f30"),
	}

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestMoonbeamRpcError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := moonbeamMockConnector{isFinalized: "true", err: fmt.Errorf("RPC failed")}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}
