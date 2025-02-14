package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
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

func TestExecuteSlashingParamsUpdate(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: set.Index})

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// create governance to update slashing params
	payloadBody := vaa.BodyGatewaySlashingParamsUpdate{
		SignedBlocksWindow:      uint64(100),
		MinSignedPerWindow:      sdk.NewDecWithPrec(5, 1).BigInt().Uint64(),
		DowntimeJailDuration:    uint64(600 * time.Second),
		SlashFractionDoubleSign: sdk.NewDecWithPrec(5, 2).BigInt().Uint64(),
		SlashFractionDowntime:   sdk.NewDecWithPrec(1, 2).BigInt().Uint64(),
	}
	payloadBz, err := payloadBody.Serialize()
	assert.NoError(t, err)

	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payloadBz)
	vBz, _ := v.Marshal()
	res, err := msgServer.ExecuteGatewayGovernanceVaa(context, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.NoError(t, err)
	assert.Equal(t, &types.EmptyResponse{}, res)
}

func TestExecuteUpdateClientVAA(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	_ = privateKeys
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: set.Index})

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// create governance to update ibc client
	subjectClientId := "07-tendermint-0"
	substituteClientId := "07-tendermint-1"

	subjectBz := [64]byte{}
	buf, err := vaa.LeftPadBytes(subjectClientId, 64)
	require.NoError(t, err)
	copy(subjectBz[:], buf.Bytes())

	substituteBz := [64]byte{}
	buf, err = vaa.LeftPadBytes(substituteClientId, 64)
	require.NoError(t, err)
	copy(substituteBz[:], buf.Bytes())

	payloadBody := vaa.BodyGatewayIBCClientUpdate{
		SubjectClientId:    subjectBz,
		SubstituteClientId: substituteBz,
	}

	payloadBz, err := payloadBody.Serialize()
	assert.NoError(t, err)

	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payloadBz)
	vBz, _ := v.Marshal()
	_, err = msgServer.ExecuteGatewayGovernanceVaa(context, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.Error(t, err)
	assert.ErrorContains(t, err, "light client not found")
}
