package keeper_test

import (
	"crypto/ecdsa"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	WormholeContractAddress1 = "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh"
	WormholeContractAddress2 = "wormhole1qg5ega6dykkxc307y25pecuufrjkxkaggkkxh7nad0vhyhtuhw3svg697z"
)

// setupWormholeMessageServer creates a keeper, context, msg server, private keys, signer, and guardian set for
// testing the wasm allowlist msg server.
func setupWormholeMessageServer(t *testing.T) (keeper.Keeper, sdk.Context, types.MsgServer, []*ecdsa.PrivateKey, sdk.AccAddress, *types.GuardianSet) {
	k, ctx := keepertest.WormholeKeeper(t)
	msgServer := keeper.NewMsgServerImpl(*k)

	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	guardianSet := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: guardianSet.Index})

	return *k, ctx, msgServer, privateKeys, signer, guardianSet
}

// TestWasmAllowlistMsgServer tests the endpoints of the wasm allowlist msg server (happy path).
func TestWasmAllowlistMsgServer(t *testing.T) {
	k, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	bech32ContractAddr := WormholeContractAddress1

	codeId := uint64(1)
	contractAddr, err := sdk.AccAddressFromBech32(bech32ContractAddr)
	require.NoError(t, err)

	// copy bytes to 32 byte array
	contractAddrBytes := [32]byte{}
	copy(contractAddrBytes[:], contractAddr.Bytes())

	// Create payload for the wasm instantiate allow list
	payload, err := vaa.BodyWormchainWasmAllowlistInstantiate{
		CodeId:       codeId,
		ContractAddr: contractAddrBytes,
	}.Serialize(vaa.ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Send msg to add wasm instantiate allow list
	_, err = msgServer.AddWasmInstantiateAllowlist(ctx, &types.MsgAddWasmInstantiateAllowlist{
		Signer:  signer.String(),
		Address: bech32ContractAddr,
		CodeId:  codeId,
		Vaa:     vBz,
	})
	require.NoError(t, err)

	// Query the allowlist
	res := k.GetAllWasmInstiateAllowedAddresses(ctx)
	require.Len(t, res, 1)
	require.Equal(t, bech32ContractAddr, res[0].ContractAddress)
	require.Equal(t, codeId, res[0].CodeId)

	// Re-generate vaa for delete wasm instantiate allow list
	payload, err = vaa.BodyWormchainWasmAllowlistInstantiate{
		CodeId:       codeId,
		ContractAddr: contractAddrBytes,
	}.Serialize(vaa.ActionDeleteWasmInstantiateAllowlist)
	require.NoError(t, err)
	v = generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err = v.Marshal()
	require.NoError(t, err)

	// Send msg to delete wasm instantiate allow list
	_, err = msgServer.DeleteWasmInstantiateAllowlist(ctx, &types.MsgDeleteWasmInstantiateAllowlist{
		Signer:  signer.String(),
		Address: bech32ContractAddr,
		CodeId:  codeId,
		Vaa:     vBz,
	})
	require.NoError(t, err)

	// Query the allowlist
	res = k.GetAllWasmInstiateAllowedAddresses(ctx)
	require.Len(t, res, 0)
}

// TestWasmAllowlistMsgServerMismatchedCodeId tests the endpoints of the wasm allowlist msg server
// with mismatched code id.
func TestWasmAllowlistMsgServerMismatchedCodeId(t *testing.T) {
	_, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	bech32ContractAddr := WormholeContractAddress1
	codeId := uint64(1)

	contractAddr, err := sdk.AccAddressFromBech32(bech32ContractAddr)
	require.NoError(t, err)

	// copy bytes to 32 byte array
	contractAddrBytes := [32]byte{}
	copy(contractAddrBytes[:], contractAddr.Bytes())

	// Create payload with mismatched code id
	payload, err := vaa.BodyWormchainWasmAllowlistInstantiate{
		CodeId:       uint64(2),
		ContractAddr: contractAddrBytes,
	}.Serialize(vaa.ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Send msg to add wasm instantiate allow list
	_, err = msgServer.AddWasmInstantiateAllowlist(ctx, &types.MsgAddWasmInstantiateAllowlist{
		Signer:  signer.String(),
		Address: bech32ContractAddr,
		CodeId:  codeId,
		Vaa:     vBz,
	})
	require.Error(t, err)
}

// TestWasmAllowlistMsgServerMismatchedContractAddr tests the endpoints of the wasm allowlist msg server
// with mismatched contract addresses.
func TestWasmAllowlistMsgServerMismatchedContractAddr(t *testing.T) {
	_, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	bech32ContractAddr := WormholeContractAddress1
	codeId := uint64(1)

	contractAddr2, err := sdk.AccAddressFromBech32(WormholeContractAddress2)
	require.NoError(t, err)

	// Create payload with mismatched contract address
	payload, err := vaa.BodyWormchainWasmAllowlistInstantiate{
		CodeId:       codeId,
		ContractAddr: [32]byte(contractAddr2),
	}.Serialize(vaa.ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Send msg to add wasm instantiate allow list
	_, err = msgServer.AddWasmInstantiateAllowlist(ctx, &types.MsgAddWasmInstantiateAllowlist{
		Signer:  signer.String(),
		Address: bech32ContractAddr,
		CodeId:  codeId,
		Vaa:     vBz,
	})
	require.Error(t, err)
}

// TestWasmAllowlistMsgServerMismatchedVaaAction tests the endpoints of the wasm allowlist msg server
// with mismatched vaa action.
func TestWasmAllowlistMsgServerMismatchedVaaAction(t *testing.T) {
	_, ctx, msgServer, privateKeys, signer, guardianSet := setupWormholeMessageServer(t)

	bech32ContractAddr := WormholeContractAddress1
	codeId := uint64(1)

	contractAddr, err := sdk.AccAddressFromBech32(bech32ContractAddr)
	require.NoError(t, err)

	// Create payload with mismatched contract address
	payload, err := vaa.BodyWormchainWasmAllowlistInstantiate{
		CodeId:       codeId,
		ContractAddr: [32]byte(contractAddr),
	}.Serialize(vaa.ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	v := generateVaa(guardianSet.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Send mismatched action
	_, err = msgServer.DeleteWasmInstantiateAllowlist(ctx, &types.MsgDeleteWasmInstantiateAllowlist{
		Signer:  signer.String(),
		Address: bech32ContractAddr,
		CodeId:  codeId,
		Vaa:     vBz,
	})
	require.Error(t, err)
}

// TestWasmAllowlistMsgServerInvalidVAA tests the endpoints of the wasm allowlist msg server
// with invalid vaa.
func TestWasmAllowlistMsgServerInvalidVAA(t *testing.T) {
	_, ctx, msgServer, _, signer, guardianSet := setupWormholeMessageServer(t)

	bech32ContractAddr := WormholeContractAddress1
	codeId := uint64(1)

	contractAddr, err := sdk.AccAddressFromBech32(bech32ContractAddr)
	require.NoError(t, err)

	// Create payload with mismatched contract address
	payload, err := vaa.BodyWormchainWasmAllowlistInstantiate{
		CodeId:       codeId,
		ContractAddr: [32]byte(contractAddr),
	}.Serialize(vaa.ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	v := generateVaa(guardianSet.Index, nil, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	// Send mismatched action
	_, err = msgServer.DeleteWasmInstantiateAllowlist(ctx, &types.MsgDeleteWasmInstantiateAllowlist{
		Signer:  signer.String(),
		Address: bech32ContractAddr,
		CodeId:  codeId,
		Vaa:     vBz,
	})
	require.Error(t, err)
}
