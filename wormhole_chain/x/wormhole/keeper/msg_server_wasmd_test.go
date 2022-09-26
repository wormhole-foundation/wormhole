package keeper_test

import (
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/crypto/sha3"
	keepertest "github.com/wormhole-foundation/wormhole-chain/testutil/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var GOVERNANCE_EMITTER = [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 04}
var GOVERNANCE_CHAIN = 1
var WH_CHAIN_ID = 3104

func createWasmStoreCodePayload(wasmBytes []byte) []byte {
	// governance message with sha3 of wasmBytes as the payload
	var hashWasm [32]byte
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(wasmBytes)
	keccak.Sum(hashWasm[:0])

	gov_msg := NewGovernanceMessage(keeper.WasmdModule, byte(keeper.ActionStoreCode), uint16(WH_CHAIN_ID), hashWasm[:])
	return gov_msg.MarshalBinary()
}

func createWasmInstantiatePayload(code_id uint64, label string, json_msg string) []byte {
	// governance message with sha3 of arguments to instantiate
	// - code_id (big endian)
	// - label
	// - json_msg
	hash_base := make([]byte, 8)
	binary.BigEndian.PutUint64(hash_base, code_id)
	hash_base = append(hash_base, []byte(label)...)
	hash_base = append(hash_base, []byte(json_msg)...)
	var expected_hash [32]byte
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(hash_base)
	keccak.Sum(expected_hash[:0])

	payload := []byte{}
	payload = append(payload, keeper.WasmdModule[:]...)
	payload = append(payload, byte(keeper.ActionInstantiateContract))
	chain_bz := [2]byte{}
	binary.BigEndian.PutUint16(chain_bz[:], uint16(WH_CHAIN_ID))
	payload = append(payload, chain_bz[:]...)
	// custom payload
	payload = append(payload, expected_hash[:]...)
	return payload
}

func TestWasmdStoreCode(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	_ = privateKeys
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     GOVERNANCE_EMITTER[:],
		GovernanceChain:       uint32(GOVERNANCE_CHAIN),
		ChainId:               uint32(WH_CHAIN_ID),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// create governance to store code
	payload := createWasmStoreCodePayload(keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, err := v.Marshal()
	assert.NoError(t, err)

	// store code should work
	res, err := msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.EXAMPLE_WASM_CONTRACT_GZIP,
		Vaa:          vBz,
	})
	_ = res
	assert.NoError(t, err)

	// replay attack does not work.
	_, err = msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.EXAMPLE_WASM_CONTRACT_GZIP,
		Vaa:          vBz,
	})
	assert.ErrorIs(t, err, types.ErrVAAAlreadyExecuted)

	// modified wasm byte code does not verify
	bad_wasm := make([]byte, len(keepertest.EXAMPLE_WASM_CONTRACT_GZIP))
	copy(bad_wasm, keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	bad_wasm[100] = bad_wasm[100] ^ 0x40
	// create vaa with the hash of the "valid" wasm
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: bad_wasm,
		Vaa:          vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Sending to wrong module is error
	payload_wrong_module := createWasmStoreCodePayload(keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	// tamper with the module id
	payload_wrong_module[16] = 0xff
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload_wrong_module)
	vBz, _ = v.Marshal()
	_, err = msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: bad_wasm,
		Vaa:          vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceModule)
}

func TestWasmdInstantiateContract(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	_ = privateKeys
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     GOVERNANCE_EMITTER[:],
		GovernanceChain:       uint32(GOVERNANCE_CHAIN),
		ChainId:               uint32(WH_CHAIN_ID),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// First we need to upload code that we can instantiate.
	payload := createWasmStoreCodePayload(keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, err := v.Marshal()
	assert.NoError(t, err)
	res, err := msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.EXAMPLE_WASM_CONTRACT_GZIP,
		Vaa:          vBz,
	})
	assert.NoError(t, err)

	code_id := res.CodeID

	// Now that we have a code_id, we can test instantiating it.
	payload = createWasmInstantiatePayload(code_id, "btc", "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc",
		Msg:    []byte("{}"),
		Vaa:    vBz,
	})
	require.NoError(t, err)

	// Test instantiating with invalid json fails
	payload = createWasmInstantiatePayload(code_id, "btc", "{")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc",
		Msg:    []byte("{"),
		Vaa:    vBz,
	})
	require.Error(t, err)

	// Test that tampering with either code_id, label, or msg fails vaa check
	payload = createWasmInstantiatePayload(code_id, "btc", "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id + 1,
		Label:  "btc",
		Msg:    []byte("{}"),
		Vaa:    vBz,
	})
	// Bad code_id
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc_bad",
		Msg:    []byte("{}"),
		Vaa:    vBz,
	})
	// Bad label
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc",
		Msg:    []byte("{\"arg\":\"bad\"}"),
		Vaa:    vBz,
	})
	// Bad msg
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Sending to wrong module is error
	payload_wrong_module := createWasmInstantiatePayload(code_id, "btc", "{}")
	// tamper with the module id
	payload_wrong_module[16] = 0xff
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(GOVERNANCE_CHAIN), payload_wrong_module)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc",
		Msg:    []byte("{\"arg\":\"bad\"}"),
		Vaa:    vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceModule)
}
