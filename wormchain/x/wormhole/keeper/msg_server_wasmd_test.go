package keeper_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
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

type Testbench struct {
	guardians       []types.GuardianValidator
	set             types.GuardianSet
	privateKeys     []*ecdsa.PrivateKey
	signer          sdk.AccAddress
	context         context.Context
	msgServer       types.MsgServer
	codeId          uint64
	contractAddress string
}

func setupAccountantAndGuardianSet(t *testing.T, ctx sdk.Context, k *keeper.Keeper) *Testbench {
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	set := createNewGuardianSet(k, ctx, guardians)
	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// First we need to (1) upload some codes and (2) instantiate.
	// (1) upload
	payload := createWasmStoreCodePayload(keepertest.ACCOUNTANT_WASM_B64_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	assert.NoError(t, err)
	res, err := msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.ACCOUNTANT_WASM_B64_GZIP,
		Vaa:          vBz,
	})
	assert.NoError(t, err)
	code_id := res.CodeID

	// (2) instantiate
	payload = createWasmInstantiatePayload(code_id, "accountant", "{}")
	v = generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	instantiate, err := msgServer.InstantiateContract(context, &types.MsgInstantiateContract{
		Signer: signer.String(),
		CodeID: code_id,
		Label:  "accountant",
		Msg:    []byte("{}"),
		Vaa:    vBz,
	})
	require.NoError(t, err)
	return &Testbench{
		guardians:       guardians,
		set:             *set,
		privateKeys:     privateKeys,
		signer:          signer,
		context:         context,
		msgServer:       msgServer,
		codeId:          code_id,
		contractAddress: instantiate.Address,
	}
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
	payload := createWasmStoreCodePayload(keepertest.ACCOUNTANT_WASM_B64_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	assert.NoError(t, err)

	// store code should work
	res, err := msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.ACCOUNTANT_WASM_B64_GZIP,
		Vaa:          vBz,
	})
	_ = res
	assert.NoError(t, err)

	// replay attack does not work.
	_, err = msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.ACCOUNTANT_WASM_B64_GZIP,
		Vaa:          vBz,
	})
	assert.ErrorIs(t, err, types.ErrVAAAlreadyExecuted)

	// modified wasm byte code does not verify
	bad_wasm := make([]byte, len(keepertest.ACCOUNTANT_WASM_B64_GZIP))
	copy(bad_wasm, keepertest.ACCOUNTANT_WASM_B64_GZIP)
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
	payload_wrong_module := createWasmStoreCodePayload(keepertest.ACCOUNTANT_WASM_B64_GZIP)
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
	payload := createWasmStoreCodePayload(keepertest.ACCOUNTANT_WASM_B64_GZIP)
	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, err := v.Marshal()
	assert.NoError(t, err)
	res, err := msgServer.StoreCode(context, &types.MsgStoreCode{
		Signer:       signer.String(),
		WASMByteCode: keepertest.ACCOUNTANT_WASM_B64_GZIP,
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
	tb := setupAccountantAndGuardianSet(t, ctx, k)

	// First we need to (1) upload some codes and (2) instantiate.
	// (1) upload
	payload := createWasmStoreCodePayload(keepertest.ACCOUNTANT_WASM_B64_GZIP)
	code_ids := []uint64{}
	for i := 0; i < 5; i++ {
		v := generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
		vBz, err := v.Marshal()
		assert.NoError(t, err)
		res, err := tb.msgServer.StoreCode(tb.context, &types.MsgStoreCode{
			Signer:       tb.signer.String(),
			WASMByteCode: keepertest.ACCOUNTANT_WASM_B64_GZIP,
			Vaa:          vBz,
		})
		assert.NoError(t, err)

		code_ids = append(code_ids, res.CodeID)
	}

	// (2) instantiate
	payload = createWasmInstantiatePayload(code_ids[0], "btc", "{}")
	v := generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ := v.Marshal()
	instantiate, err := tb.msgServer.InstantiateContract(tb.context, &types.MsgInstantiateContract{
		Signer: tb.signer.String(),
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
		v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
		vBz, _ = v.Marshal()
		_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
			Signer:   tb.signer.String(),
			CodeID:   code_id,
			Contract: instantiate.Address,
			Msg:      []byte("{}"),
			Vaa:      vBz,
		})
		require.NoError(t, err)

		// replaying the same message should return ErrVAAAlreadyExecuted
		_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
			Signer:   tb.signer.String(),
			CodeID:   code_id,
			Contract: instantiate.Address,
			Msg:      []byte("{}"),
			Vaa:      vBz,
		})
		require.ErrorIs(t, err, types.ErrVAAAlreadyExecuted)
	}

	// Test failure using the wrong codeid
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, "{}")
	v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
		Signer: tb.signer.String(),
		// Switch codeid
		CodeID:   code_ids[1],
		Contract: instantiate.Address,
		Msg:      []byte("{}"),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Test failure using the wrong contract
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, "{}")
	v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
		Signer: tb.signer.String(),
		CodeID: code_ids[0],
		// swap contract address for the signer address
		Contract: tb.signer.String(),
		Msg:      []byte("{}"),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Test failure using the wrong msg
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, `{"hello": "world"}`)
	v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
		Signer:   tb.signer.String(),
		CodeID:   code_ids[0],
		Contract: instantiate.Address,
		// modify msg
		Msg: []byte(`{"hallo": "world"}`),
		Vaa: vBz,
	})
	assert.ErrorIs(t, err, types.ErrInvalidHash)

	// Test migrating with invalid json fails
	payload = createWasmMigratePayload(code_ids[0], instantiate.Address, `{"hello": }`)
	v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
		Signer:   tb.signer.String(),
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
	v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload_wrong_module)
	vBz, _ = v.Marshal()
	_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
		Signer:   tb.signer.String(),
		CodeID:   code_ids[0],
		Contract: instantiate.Address,
		Msg:      []byte(`{}`),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceModule)

	// test action byte is checked by sending a valid instantiate vaa
	payload = createWasmInstantiatePayload(code_ids[0], "btc", "{}")
	v = generateVaa(tb.set.Index, tb.privateKeys, vaa.ChainID(vaa.GovernanceChain), payload)
	vBz, _ = v.Marshal()
	_, err = tb.msgServer.MigrateContract(tb.context, &types.MsgMigrateContract{
		Signer:   tb.signer.String(),
		CodeID:   code_ids[0],
		Contract: "btc",
		Msg:      []byte("{}"),
		Vaa:      vBz,
	})
	assert.ErrorIs(t, err, types.ErrUnknownGovernanceAction)
}

// This specifically tests the modify vaa in accountant
// This also tests that the path to verify VAAs through the accountant contract to the wormhole querier interface is working.
func TestWasmdAccountantContractModify(t *testing.T) {
	k, wasmd, permissionedWasmd, ctx := keepertest.WormholeKeeperAndWasmd(t)
	_ = permissionedWasmd

	tb := setupAccountantAndGuardianSet(t, ctx, k)

	contract_addr, err := sdk.AccAddressFromBech32(tb.contractAddress)
	require.NoError(t, err)

	token_address := [32]byte{}
	for i := 0; i < len(token_address); i++ {
		token_address[i] = 0x7c
	}

	// construct the modify balance vaa
	modify_msg := vaa.BodyAccountantModifyBalance{
		Module:        "GlobalAccountant",
		TargetChainID: vaa.ChainIDWormchain,
		Sequence:      uint64(lastestSequence),
		ChainId:       vaa.ChainIDSolana,
		TokenChain:    vaa.ChainIDSolana,
		TokenAddress:  token_address,
		Kind:          1, // Add
		Amount:        uint256.NewInt(50),
		Reason:        "test modify",
	}
	ts := time.Date(2012, 12, 12, 12, 12, 12, 12, time.UTC)
	modify_buf, err := modify_msg.Serialize()
	require.NoError(t, err)
	modify_vaa := vaa.CreateGovernanceVAA(ts, 1, 1, tb.set.Index, modify_buf)
	*modify_vaa = signVaa(*modify_vaa, tb.privateKeys)
	vBz, err := modify_vaa.Marshal()
	require.NoError(t, err)

	// construct the `SubmitVAAs` payload
	vBzBase64 := base64.RawStdEncoding.EncodeToString(vBz)
	execute_msg := fmt.Sprintf(`{"submit_vaas": {"vaas": ["%s"]}}`, vBzBase64)
	fmt.Println("submit_vaas: ", execute_msg)

	// first query the balance and expect an error
	token_address_hex := hex.EncodeToString(token_address[:])
	query_msg := fmt.Sprintf(`{"balance" : { "chain_id": %d, "token_chain": %d, "token_address": "%s"}}`, vaa.ChainIDSolana, vaa.ChainIDSolana, token_address_hex)
	_, err = wasmd.QuerySmart(ctx, contract_addr, []byte(query_msg))
	require.Error(t, err)

	// Now we can test sending Modify VAA to accountant
	wasmResponse, err := permissionedWasmd.Execute(ctx, contract_addr, tb.signer, []byte(execute_msg), []sdk.Coin{})
	_ = wasmResponse
	require.NoError(t, err)

	// Query the balance and expect no error
	_, err = wasmd.QuerySmart(ctx, contract_addr, []byte(query_msg))
	require.NoError(t, err)
}

// This specifically tests the modify vaa in accountant
// This also tests that the path to verify VAAs through the accountant contract to the wormhole querier interface is working.
func TestWasmdAccountantContractSubmitObservation(t *testing.T) {
	k, _, permissionedWasmd, ctx := keepertest.WormholeKeeperAndWasmd(t)
	_ = permissionedWasmd

	tb := setupAccountantAndGuardianSet(t, ctx, k)

	contract_addr, err := sdk.AccAddressFromBech32(tb.contractAddress)
	require.NoError(t, err)

	token_address := [32]byte{}
	for i := 0; i < len(token_address); i++ {
		token_address[i] = 0x7c
	}

	// 1. Artisanally craft a serialized, signed observation message
	// See node/package/accountant/submit_obs.go
	var submitObservationPrefix = []byte("acct_sub_obsfig_000000000000000000|")
	observation := `[{
		"tx_hash": "1234",
		"timestamp": 1234,
		"nonce": 1,
		"sequence": 1,
		"consistency_level": 1,
		"emitter_chain": 1,
		"emitter_address": "2F80361EC4EC8DD9452F80A1CF5189442EEB09EC9F3B7C2FA0BCFCD350380AC6",
		"payload": ""
	}]`
	require.NoError(t, err)
	// the serialized observation
	observations_serialized := base64.RawStdEncoding.EncodeToString([]byte(observation))

	// the signature of observation
	digest, err := vaa.MessageSigningDigest(submitObservationPrefix, []byte(observation))
	require.NoError(t, err)
	sig, err := crypto.Sign(digest[:], tb.privateKeys[0])
	require.NoError(t, err)
	signature_serialized := strings.Join(strings.Fields(fmt.Sprintf("%d", sig)), ",")

	execute_msg := fmt.Sprintf(`{
		"submit_observations": {
			"guardian_set_index": %d,
			"observations": "%s",
			"signature": {
			  "index": %d,
			  "signature": %s
			}
		  }
		}`, tb.set.Index, observations_serialized, 0, signature_serialized)

	fmt.Println("submit_observations: ", execute_msg)

	// 2. Now we can test sending Modify VAA to accountant
	_, err = permissionedWasmd.Execute(ctx, contract_addr, tb.signer, []byte(execute_msg), []sdk.Coin{})
	require.NoError(t, err)

	// sending again is okay
	_, err = permissionedWasmd.Execute(ctx, contract_addr, tb.signer, []byte(execute_msg), []sdk.Coin{})
	require.NoError(t, err)

	// 3. Let's change one byte of the signature and verify it fails
	sig[5]++
	signature_serialized = strings.Join(strings.Fields(fmt.Sprintf("%d", sig)), ",")
	execute_msg = fmt.Sprintf(`{
		"submit_observations": {
			"guardian_set_index": %d,
			"observations": "%s",
			"signature": {
			  "index": %d,
			  "signature": %s
			}
		  }
		}`, tb.set.Index, observations_serialized, 0, signature_serialized)

	fmt.Println("submit_observations: ", execute_msg)

	_, err = permissionedWasmd.Execute(ctx, contract_addr, tb.signer, []byte(execute_msg), []sdk.Coin{})
	require.Error(t, err)
}
