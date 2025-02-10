package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestExecuteGatewayGovernanceVaaUpgrades tests creating and cancelling upgrades.
func TestExecuteGatewayGovernanceVaaUpgrades(t *testing.T) {
	_, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	// Create upgrade payload
	payload, err := vaa.BodyGatewayScheduleUpgrade{
		Name:   "v5.0.0",
		Height: uint64(100),
	}.Serialize()
	require.NoError(t, err)

	// Generate VAA
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Submit upgrade governance VAA
	res, err := msgServer.ExecuteGatewayGovernanceVaa(ctx, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Create cancel upgrade payload
	payload, err = vaa.EmptyPayloadVaa(vaa.GatewayModuleStr, vaa.ActionCancelUpgrade, vaa.ChainIDWormchain)
	require.NoError(t, err)

	// Generate VAA
	v = generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err = v.Marshal()
	require.NoError(t, err)

	// Submit cancel upgrade governance VAA
	res, err = msgServer.ExecuteGatewayGovernanceVaa(ctx, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	require.NoError(t, err)
	require.NotNil(t, res)
}

// TestExecuteGatewayGovernanceVaaSetIbcComposabilityMwContract tests setting the IBC composability contract.
func TestExecuteGatewayGovernanceVaaSetIbcComposabilityMwContract(t *testing.T) {
	k, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	// Get contract bytes
	contractAddr := WormholeContractAddress1
	contractAddrBz, err := sdk.AccAddressFromBech32(contractAddr)
	require.NoError(t, err)

	// Create payload
	payload, err := vaa.BodyGatewayIbcComposabilityMwContract{
		ContractAddr: [32]byte(contractAddrBz),
	}.Serialize()
	require.NoError(t, err)

	// Generate VAA
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Submit governance VAA
	res, err := msgServer.ExecuteGatewayGovernanceVaa(ctx, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	// Validate the contract was set
	contract := k.GetIbcComposabilityMwContract(ctx)
	require.Equal(t, contractAddr, contract.ContractAddress)
}

// TestExecuteGatewayGovernanceVaaUnknownAction tests submitting an unknown action.
func TestExecuteGatewayGovernanceVaaUnknownAction(t *testing.T) {
	_, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	// Create payload
	payload, err := vaa.EmptyPayloadVaa(vaa.GatewayModuleStr, vaa.GovernanceAction(100), vaa.ChainIDWormchain)
	require.NoError(t, err)

	// Generate VAA
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Submit governance VAA
	_, err = msgServer.ExecuteGatewayGovernanceVaa(ctx, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	require.Error(t, err)
}

// TestExecuteGatewayGovernanceVaaInvalidVAA tests submitting an invalid VAA.
func TestExecuteGatewayGovernanceVaaInvalidVAA(t *testing.T) {
	_, ctx, msgServer, _, signer, guardianSet := setupWormholeMessageServer(t)

	// Create payload
	payload, err := vaa.EmptyPayloadVaa(vaa.GatewayModuleStr, vaa.ActionCancelUpgrade, vaa.ChainIDWormchain)
	require.NoError(t, err)

	// Generate VAA
	v := generateVaa(guardianSet.Index, nil, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Submit governance VAA
	_, err = msgServer.ExecuteGatewayGovernanceVaa(ctx, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	require.Error(t, err)
}
