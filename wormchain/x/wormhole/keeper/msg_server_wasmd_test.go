package keeper_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/crypto/sha3"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func createWasmStoreCodePayload(wasmBytes []byte) []byte {
	// governance message with sha3 of wasmBytes as the payload
	var hashWasm [32]byte
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(wasmBytes)
	keccak.Sum(hashWasm[:0])

	gov_msg := types.NewGovernanceMessage(vaa.WasmdModule, byte(vaa.ActionStoreCode), uint16(vaa.ChainIDWormchain), hashWasm[:])
	return gov_msg.MarshalBinary()
}

func createWasmInstantiatePayload(code_id uint64, label string, json_msg string) []byte {
	// governance message with sha3 of arguments to instantiate
	// - code_id (big endian)
	// - label
	// - json_msg
	expected_hash := vaa.CreateInstatiateCosmwasmContractHash(code_id, label, []byte(json_msg))

	var payload bytes.Buffer
	payload.Write(vaa.WasmdModule[:])
	payload.Write([]byte{byte(vaa.ActionInstantiateContract)})
	binary.Write(&payload, binary.BigEndian, uint16(vaa.ChainIDWormchain))
	// custom payload
	payload.Write(expected_hash[:])
	return payload.Bytes()
}

func createWasmMigratePayload(code_id uint64, contract string, json_msg string) []byte {
	// governance message with sha3 of arguments to instantiate
	// - code_id (big endian)
	// - label
	// - json_msg
	expected_hash := vaa.CreateMigrateCosmwasmContractHash(code_id, contract, []byte(json_msg))

	var payload bytes.Buffer
	payload.Write(vaa.WasmdModule[:])
	payload.Write([]byte{byte(vaa.ActionMigrateContract)})
	binary.Write(&payload, binary.BigEndian, uint16(vaa.ChainIDWormchain))
	// custom payload
	payload.Write(expected_hash[:])
	return payload.Bytes()
}

func TestWasmdStoreCode(t *testing.T) {
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

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// create governance to store code
	payload := createWasmStoreCodePayload(keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload_wrong_module)
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
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// First we need to upload code that we can instantiate.
	payload := createWasmStoreCodePayload(keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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

	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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

	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
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

	// Sending to wrong module is error (basically test that governance verification is in place)
	payload_wrong_module := createWasmInstantiatePayload(code_id, "btc", "{}")
	// tamper with the module id
	payload_wrong_module[16] = 0xff
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload_wrong_module)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc",
		Msg:    []byte("{\"arg\":\"bad\"}"),
		Vaa:    vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceModule)

	// test action byte is checked by sending a valid migrate vaa
	payload = createWasmMigratePayload(code_id, "btc", "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "btc",
		Msg:    []byte("{}"),
		Vaa:    vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceAction)
}

func TestWasmdMigrateContract(t *testing.T) {
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

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// First we need to (1) upload some codes and (2) instantiate.
	// (1) upload
	payload := createWasmStoreCodePayload(keepertest.EXAMPLE_WASM_CONTRACT_GZIP)
	code_ids := []uint64{}
	for i := 0; i < 5; i++ {
		v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
		vBz, err := v.Marshal()
		assert.NoError(t, err)
		res, err := msgServer.StoreCode(context, &types.MsgStoreCode{
			Signer:       signer.String(),
			WASMByteCode: keepertest.EXAMPLE_WASM_CONTRACT_GZIP,
			Vaa:          vBz,
		})
		assert.NoError(t, err)

		code_ids = append(code_ids, res.CodeID)
	}

	// (2) instantiate
	payload = createWasmInstantiatePayload(code_ids[0], "btc", "{}")
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ := v.Marshal()
	instantiate, err := msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_ids[0],
		Label:  "btc",
		Msg:    []byte("{}"),
		Vaa:    vBz,
	})
	require.NoError(t, err)

	// Now we can test migrating

	// Confirm migrate works
	for _, code_id := range code_ids {
		payload = createWasmMigratePayload(code_id, instantiate.Address, "{}")
		v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
		vBz, _ = v.Marshal()
		_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
			Signer:   signer.String(),
			CodeID:   code_id,
			Contract: instantiate.Address,
			Msg:      []byte("{}"),
			Vaa:      vBz,
		})
		require.NoError(t, err)
	}

	// Test failure using the wrong codeid
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
		Signer: signer.String(),
		// Switch codeid
		CodeID:   code_ids[1],
		Contract: instantiate.Address,
		Msg:      []byte("{}"),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Test failure using the wrong contract
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
		Signer: signer.String(),
		CodeID: code_ids[0],
		// modify address
		Contract: instantiate.Address + "a",
		Msg:      []byte("{}"),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Test failure using the wrong msg
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, `{"hello": "world"}`)
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
		Signer:   signer.String(),
		CodeID:   code_ids[0],
		Contract: instantiate.Address,
		// modify msg
		Msg: []byte(`{"hallo": "world"}`),
		Vaa: vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Test migrating with invalid json fails
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, `{"hello": }`)
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
		Signer:   signer.String(),
		CodeID:   code_ids[0],
		Contract: instantiate.Address,
		Msg:      []byte(`{"hello": }`),
		Vaa:      vBz,
	})
	assert.NotErrorIs(t, err, types.ErrInvalidHash)
	require.Error(t, err)

	// Sending to wrong module is error (basically test that governance verification is in place)
	payload_wrong_module := createWasmMigratePayload(code_ids[0], instantiate.Address, `{}`)
	// tamper with the module id
	payload_wrong_module[16] = 0xff
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload_wrong_module)
	vBz, _ = v.Marshal()
	_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
		Signer:   signer.String(),
		CodeID:   code_ids[0],
		Contract: instantiate.Address,
		Msg:      []byte(`{}`),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceModule)

	// test action byte is checked by sending a valid instantiate vaa
	payload = createWasmInstantiatePayload(code_ids[0], "btc", "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = msgServer.MigrateContract(context, &types.MsgMigrateContract{
		Signer:   signer.String(),
		CodeID:   code_ids[0],
		Contract: "btc",
		Msg:      []byte("{}"),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceAction)

}
