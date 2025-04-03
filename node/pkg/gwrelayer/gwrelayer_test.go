package gwrelayer

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"
)

func Test_convertBech32AddressToWormhole(t *testing.T) {
	expectedAddress, err := hex.DecodeString("ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b")
	require.NoError(t, err)

	// Basic success case.
	targetAddress, err := convertBech32AddressToWormhole("wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465")
	require.NoError(t, err)
	assert.Equal(t, true, bytes.Equal(expectedAddress, targetAddress.Bytes()))

	// Garbage in should generate an error.
	_, err = convertBech32AddressToWormhole("hello world!")
	assert.Error(t, err)

	// Empty input should generate an error.
	_, err = convertBech32AddressToWormhole("")
	assert.Error(t, err)
}

func Test_shouldPublishToIbcTranslator(t *testing.T) {
	type Test struct {
		label   string
		payload []byte
		result  bool
		err     bool
	}

	tests := []Test{
		{label: "empty payload", payload: []byte{}, result: false, err: false},
		{label: "non-transfer", payload: []byte{0x0}, result: false, err: false},
		{label: "payload type 1", payload: []byte{0x1}, result: false, err: false},
		{label: "payload too short", payload: []byte{0x3, 0x00, 0x00}, result: false, err: true},
		{label: "wrong target chain", payload: decodeBytes("0300000000000000000000000000000000000000000000000000000000000000640000000000000000000000005425890298aed601595a70ab815c96711a31bc6500066d9ae6b2d333c1d65301a59da3eed388ca5dc60cb12496584b75cbe6b15fdbed0020000000000000000000000000e6990c7e206d418d62b9e50c8e61f59dc360183b7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d57786c656d3179636d31336348687865575679626e6c344d33706a595768735a4756715958686e4f485a364f484e774d32526f227d7d"), result: false, err: false},
		{label: "wrong target address", payload: decodeBytes("0300000000000000000000000000000000000000000000000000000000000000640000000000000000000000005425890298aed601595a70ab815c96711a31bc6500066d9ae6b2d333c1d65301a59da3eed388ca5dc60cb12496584b75cbe6b15fdbed0C20000000000000000000000000e6990c7e206d418d62b9e50c8e61f59dc360183b7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d57786c656d3179636d31336348687865575679626e6c344d33706a595768735a4756715958686e4f485a364f484e774d32526f227d7d"), result: false, err: false},
		{label: "should publish", payload: decodeBytes("0300000000000000000000000000000000000000000000000000000000000000640000000000000000000000005425890298aed601595a70ab815c96711a31bc650006ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b0C20000000000000000000000000e6990c7e206d418d62b9e50c8e61f59dc360183b7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d57786c656d3179636d31336348687865575679626e6c344d33706a595768735a4756715958686e4f485a364f484e774d32526f227d7d"), result: true, err: false},
	}

	logger := zap.NewNop()

	solanaEmitter, err := vaa.StringToAddress("ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5")
	require.NoError(t, err)

	tokenBridges, _, err := buildTokenBridgeMap(logger, common.MainNet)
	require.NoError(t, err)

	targetAddress, err := convertBech32AddressToWormhole("wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465")
	require.NoError(t, err)

	for seqNum, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			v := &vaa.VAA{
				Version:          uint8(1),
				GuardianSetIndex: uint32(1),
				Signatures:       []*vaa.Signature{},
				Timestamp:        time.Unix(0, 0),
				Nonce:            uint32(1),
				Sequence:         uint64(seqNum), // #nosec G115 -- We're iterating over a fixed length array defined above
				ConsistencyLevel: uint8(32),
				EmitterChain:     vaa.ChainIDSolana,
				EmitterAddress:   solanaEmitter,
				Payload:          tc.payload,
			}
			result, err := shouldPublishToIbcTranslator(tokenBridges, v, vaa.ChainIDWormchain, targetAddress)
			assert.Equal(t, tc.err, err != nil)
			assert.Equal(t, tc.result, result)
		})
	}
}

func Test_shouldPublishToTokenBridge(t *testing.T) {
	type Test struct {
		label   string
		chain   vaa.ChainID
		address vaa.Address
		payload []byte
		result  bool
	}

	logger := zap.NewNop()

	tokenBridges, tokenBridgeAddress, err := buildTokenBridgeMap(logger, common.MainNet)
	require.NoError(t, err)
	require.NotNil(t, tokenBridges)
	require.Equal(t, tokenBridgeAddress, "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh")

	tests := []Test{
		{label: "unknown chain", chain: vaa.ChainIDUnset, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{}, result: false},
		{label: "unknown emitter", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000000000000000000000000000000000000000000000"), payload: []byte{}, result: false},
		{label: "wormchain", chain: vaa.ChainIDWormchain, address: addr("aeb534c45c3049d380b9d9b966f9895f53abd4301bfaff407fa09dea8ae7a924"), payload: []byte{}, result: false},
		{label: "empty payload", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{}, result: false},
		{label: "not an attest", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{0x1}, result: false},
		{label: "should publish", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{0x2}, result: true},
	}

	for _, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			v := &vaa.VAA{
				Version:          uint8(1),
				GuardianSetIndex: uint32(1),
				Signatures:       []*vaa.Signature{},
				Timestamp:        time.Unix(0, 0),
				Nonce:            uint32(1),
				Sequence:         uint64(1),
				ConsistencyLevel: uint8(32),
				EmitterChain:     tc.chain,
				EmitterAddress:   tc.address,
				Payload:          tc.payload,
			}

			result := shouldPublishToTokenBridge(tokenBridges, v)
			assert.Equal(t, tc.result, result)
		})
	}

	_, err = sdktypes.Bech32ifyAddressBytes("wormhole", decodeBytes("aeb534c45c3049d380b9d9b966f9895f53abd4301bfaff407fa09dea8ae7a924"))
	require.NoError(t, err)
}

func decodeBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func addr(str string) vaa.Address {
	a, err := vaa.StringToAddress(str)
	if err != nil {
		panic("failed to convert address")
	}
	return a
}

func Test_verifyDevnetTokenBridgeAddress(t *testing.T) {
	tokenBridgeAddressInTilt := "wormhole1eyfccmjm6732k7wp4p6gdjwhxjwsvje44j0hfx8nkgrm8fs7vqfssvpdkx" //nolint:gosec
	targetAddress, err := convertBech32AddressToWormhole(tokenBridgeAddressInTilt)
	require.NoError(t, err)

	expectedAddress, exists := sdk.KnownDevnetTokenbridgeEmitters[vaa.ChainIDWormchain]
	require.True(t, exists)
	assert.True(t, bytes.Equal(expectedAddress[:], targetAddress[:]))
}

func Test_shouldPublishToIbcTranslatorShouldIgnoreUnknownEmitter(t *testing.T) {
	vaaBytes, err := hex.DecodeString("01000000040d008f68689ec1cef3f53d7a20e83b6f42346ddcdcaff1c8f9435905a674949621b73a37e803e65f213564d2b019964ad3b66586c2f8255c5308dde4444759e702ef01024d6b7e7b74d6d56d11ee0fd59468142a7566cf019cc57dd63c010dbdfd32da155d8d4fc3bf6404ff26bfd754b96f7c2426e680acb99a224687ba819e289d92d401040feda4402fdf099b7aa50553aa9bdd991724d25309ae325fbd56ec13136f83bc0fc4129ecee55d08c38257d6bc7465cf21568212e9199ab1e4e28001baeaf2b0000634097288b9f9570af929f1cc52bb966430d1606f13c916180f86ff2e0467b49e2e05d299fa24150f7f478319b39b03815b8a23f52e8e6c994d6bdd6a544607bf0007bb7b6393e60523997c2d2f999a4ce2d57a2958dca25284a966c4abb0c097284324d1329d9eb75b4d1d24601a3ed5b94571ee57b9fb4c873ce5bcc398505f5694000844d5fe1672a004612c436c9fffd2c2c8bbe09e827754cef20b81dd65f9094e9a757888203b0f84747f2a1259e1699a3605b7320202011875b8b497584c4b253500092badc4d98dd77ef6b55d189d74c8191f32a374c450cfaea380951cf266683bd7097f5ada2e3fe5747baa99cc4db933e12f9b3270739230b834fa19c8da962612000a6d08e0b4a6bf5e59ae96c63169f2566a839aa7eee125ff31536f797fa1a4174a158be54f31c583453025fba6b768e2aad6966871ea367d11958f1aa683ec913a000d5a69c9968066e471726fe59633df72ecad6737b931754314ebde14fcf4b723b97ab58774aad432e3aa75896ec4620a07b3041ca531a5b67a1eb40805cd05f222010eb82dc0abf7c87bceb03ded9d83186eb3cd3082d042eaed3b1c05f69f13aeeffa7e93a224100acc12f1425feb63e235b63b38ca6e3507847f099ec41b8aea0453010f55af1b26a5c2b704aa064dad6db5105fa5c75816732da3ecdad1dc285eda6e043aaa37f5f5e1d613c4df5a15a82c347e6badde62382d16f9b1be58e93fb157a30011d63eac9a9c84d01a32fb03f12ae113fd28201682eec627d56db30496a837ed086560b674c2e305235eba76e20260afd566ee4c2125a1e8366aa1ca46415f504401129ed9924a060e5e83ac17076fd01204b5b35c6a0d754dd5ab27f8a4d7d3451b8710708fa090525ecbfeba2e72bdaff0ffecd0e71ab02f10c5bf74bea76e54efeb0166856d4400000001000175233cdd3dab6c9f3134860c2b2443e80072b0880a3405b2bd87234f024711bc0000000000002a1f200301000000000000a62c00000005000000000000000000000000aa446e32b8c1f1bb82e6a548c6baf54654ebe9110000000000000000000000000000348b")
	require.NoError(t, err)
	v, err := vaa.Unmarshal(vaaBytes)
	require.NoError(t, err)

	logger := zap.NewNop()

	tokenBridges, tokenBridgeAddress, err := buildTokenBridgeMap(logger, common.MainNet)
	require.NoError(t, err)
	require.NotNil(t, tokenBridges)
	require.Equal(t, tokenBridgeAddress, "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh")

	targetAddress, err := convertBech32AddressToWormhole("wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465")
	require.NoError(t, err)

	shouldPub, err := shouldPublishToIbcTranslator(tokenBridges, v, vaa.ChainIDWormchain, targetAddress)
	assert.NoError(t, err)
	assert.False(t, shouldPub)
}

func Test_canIgnoreFailure(t *testing.T) {
	assert.True(t, canIgnoreFailure("failed to execute message; message index: 0: Generic error: VaaAlreadyExecuted: execute wasm contract failed"))
	assert.True(t, canIgnoreFailure("failed to execute message; message index: 0: Generic error: this asset has already been attested: execute wasm contract failed"))
	assert.False(t, canIgnoreFailure("failed to execute message; message index: 0: Generic error: some other failure: execute wasm contract failed"))
}
