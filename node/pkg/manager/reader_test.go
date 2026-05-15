package manager

import (
	"context"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func TestGetManagerSet_Dogecoin_Index1(t *testing.T) {
	rpcURL := "https://ethereum-sepolia-rpc.publicnode.com"

	ctx := context.Background()
	logger := zap.NewNop()

	reader, err := NewManagerSetReader(logger, common.TestNet, rpcURL)
	require.NoError(t, err)

	set, err := reader.GetManagerSet(ctx, vaa.ChainIDDogecoin, 1, nil)
	require.NoError(t, err)

	// Verify basic properties
	assert.Equal(t, uint32(1), set.Index)
	assert.Equal(t, uint8(5), set.M, "expected M=5 for 5-of-7 multisig")
	assert.Equal(t, uint8(7), set.N, "expected N=7 for 5-of-7 multisig")
	assert.Len(t, set.PublicKeys, 7)
	for i, pk := range set.PublicKeys {
		assert.Len(t, pk, 33, "expected compressed secp256k1 public key at index %d", i)
	}
	assert.False(t, set.IsSigner, "expected IsSigner=false when no signer provided")

	// Verify caching works
	set2, err := reader.GetManagerSet(ctx, vaa.ChainIDDogecoin, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, set, set2, "expected cached result")
}
