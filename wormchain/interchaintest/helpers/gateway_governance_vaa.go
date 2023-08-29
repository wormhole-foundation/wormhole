package helpers

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func GetMiddlewareContract(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) string {
	node := chain.GetFullNode()
	stdout, _, err := node.ExecQuery(ctx, "wormhole", "show-ibc-composability-mw-contract")
	require.NoError(t, err)
	return string(stdout)
}

func SetMiddlewareContract(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	cfg ibc.ChainConfig,
	contractBech32Addr string,
	guardians *guardians.ValSet,
) {
	node := chain.GetFullNode()

	contractAddr := [32]byte{}
	copy(contractAddr[:], MustAccAddressFromBech32(contractBech32Addr, cfg.Bech32Prefix).Bytes())
	payload := vaa.BodyGatewayIbcComposabilityMwContract{
		ContractAddr: contractAddr,
	}
	payloadBz := payload.Serialize()
	v := generateVaa(0, guardians, vaa.GovernanceChain, vaa.GovernanceEmitter, payloadBz)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "execute-gateway-governance-vaa", vHex, "--gas", "auto")
	require.NoError(t, err)
}

func ScheduleUpgrade(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	name string,
	height uint64,
	guardians *guardians.ValSet,
) {
	node := chain.GetFullNode()

	payload := vaa.BodyGatewayScheduleUpgrade{
		Name:   name,
		Height: height,
	}
	payloadBz := payload.Serialize()
	v := generateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payloadBz)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "execute-gateway-governance-vaa", vHex, "--gas", "auto")
	require.NoError(t, err)
}

func CancelUpgrade(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	guardians *guardians.ValSet,
) {
	node := chain.GetFullNode()

	payloadBz := vaa.EmptyPayloadVaa(vaa.GatewayModuleStr, vaa.ActionCancelUpgrade, vaa.ChainIDWormchain)
	v := generateVaa(0, guardians, vaa.GovernanceChain, vaa.GovernanceEmitter, payloadBz)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	_, err = node.ExecTx(ctx, keyName, "wormhole", "execute-gateway-governance-vaa", vHex, "--gas", "auto")
	require.NoError(t, err)
}
