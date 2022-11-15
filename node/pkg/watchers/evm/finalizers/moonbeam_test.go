package finalizers

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func TestMoonbeamErrorReturnedIfBlockIsNil(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}

func TestMoonbeamBlockNotFinalized(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.HexToHash("0x1076cd8c207f31e1638b37bb358c458f216f5451f06e2ccb4eb9db66ad669f30"),
	}

	baseConnector.SetResults([]string{"false"})

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, false, finalized)
}
func TestMoonbeamBlockIsFinalized(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	block := &connectors.NewBlock{
		Number: big.NewInt(125),
		Hash:   ethCommon.HexToHash("0x1076cd8c207f31e1638b37bb358c458f216f5451f06e2ccb4eb9db66ad669f30"),
	}

	baseConnector.SetResults([]string{"true"})

	finalized, err := finalizer.IsBlockFinalized(ctx, block)
	require.NoError(t, err)
	assert.Equal(t, true, finalized)
}

func TestMoonbeamRpcError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := connectors.MockConnector{}

	finalizer := NewMoonbeamFinalizer(logger, &baseConnector)
	assert.NotNil(t, finalizer)

	baseConnector.SetError(fmt.Errorf("RPC failed"))

	_, err := finalizer.IsBlockFinalized(ctx, nil)
	require.Error(t, err)
}
