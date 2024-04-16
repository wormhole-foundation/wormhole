package accountant

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const goodPayload = "9945ff10042942fafabe0000000000000000000000000000000000000000000000000000042942fababe00000000000000000000000000000000000000000000000000000091128434bafe23430000000000000000000000000000000000ce00aa00000000004667921341234300000000000000000000000000000000000000000000000000004f994e545407000000000012d687beefface00000000000000000000000000000000000000000000000000000000feebcafe0000000000000000000000000000000000000000000000000000000000110000"

func TestNttParsePayloadSuccess(t *testing.T) {
	payload, err := hex.DecodeString(goodPayload)
	require.NoError(t, err)
	assert.True(t, nttIsPayloadNTT(payload))
}

func TestNttParsePayloadTooShort(t *testing.T) {
	payload, err := hex.DecodeString("9945ff10042942fafabe00000000000000000000000000000000000000000000000000000079000000367999a1014667921341234300000000000000000000000000000000000000000000000000004f994e54")
	require.NoError(t, err)
	assert.False(t, nttIsPayloadNTT(payload))
}

func TestNttParsePayloadNoWhPrefix(t *testing.T) {
	payload, err := hex.DecodeString("9845ff10042942fafabe00000000000000000000000000000000000000000000000000000079000000367999a1014667921341234300000000000000000000000000000000000000000000000000004f994e545407000000000012d687beefface00000000000000000000000000000000000000000000000000000000feebcafe0000000000000000000000000000000000000000000000000000000000110000")
	require.NoError(t, err)
	assert.False(t, nttIsPayloadNTT(payload))
}

func TestNttParsePayloadNoTransferPrefix(t *testing.T) {
	payload, err := hex.DecodeString("9945ff10042942fafabe00000000000000000000000000000000000000000000000000000079000000367999a1014667921341234300000000000000000000000000000000000000000000000000004f994e545307000000000012d687beefface00000000000000000000000000000000000000000000000000000000feebcafe0000000000000000000000000000000000000000000000000000000000110000")
	require.NoError(t, err)
	assert.False(t, nttIsPayloadNTT(payload))
}

func TestNttParseMsgSuccess(t *testing.T) {
	emitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e74")
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodPayload)
	require.NoError(t, err)

	emitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDEthereum, emitterAddr: emitterAddr}: true,
	}

	msg := &common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(42),
		Sequence:         uint64(123456),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(0),
		Payload:          payload,
	}

	isNTT, enforceFlag := nttIsMsgDirectNTT(msg, emitters)
	assert.True(t, isNTT)
	assert.True(t, enforceFlag)
}

func TestNttParseMsgWrongEmitterChain(t *testing.T) {
	emitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e74")
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodPayload)
	require.NoError(t, err)

	emitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDEthereum, emitterAddr: emitterAddr}: true,
	}

	msg := &common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(42),
		Sequence:         uint64(123456),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: uint8(0),
		Payload:          payload,
	}

	isNTT, _ := nttIsMsgDirectNTT(msg, emitters)
	assert.False(t, isNTT)
}

func TestNttParseMsgWrongEmitterAddress(t *testing.T) {
	goodEmitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e74")
	require.NoError(t, err)

	badEmitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e75")
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodPayload)
	require.NoError(t, err)

	emitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDEthereum, emitterAddr: goodEmitterAddr}: true,
	}

	msg := &common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654543099), 0),
		Nonce:            uint32(42),
		Sequence:         uint64(123456),
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   badEmitterAddr,
		ConsistencyLevel: uint8(0),
		Payload:          payload,
	}

	isNTT, _ := nttIsMsgDirectNTT(msg, emitters)
	assert.False(t, isNTT)
}

const goodArPayload = "0127150000000000000000000000005a76440b725909000697e0f72646adf1a492df8b000000d99945ff1000000000000000000000000024c7e23e3a97cd2f04c9eb9f354bb7f3b31d2d1a000000000000000000000000605de5e0880cfd6ffc61af9585cbab3946594a3d009100000000000000000000000000000000000000000000000000000000000000040000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7004f994e5454080000000077359400000000000000000000000000169d91c797edf56100f1b765268145660503a4230000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7271500000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000493e0000000000000000000000000000000000000000000000000000000000983146f271500000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db810000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db81000000000000000000000000c5bf11ab6ae525ffca02e2af7f6704cdcecec2ea00"
const nttPayloadInAr = "9945ff1000000000000000000000000024c7e23e3a97cd2f04c9eb9f354bb7f3b31d2d1a000000000000000000000000605de5e0880cfd6ffc61af9585cbab3946594a3d009100000000000000000000000000000000000000000000000000000000000000040000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7004f994e5454080000000077359400000000000000000000000000169d91c797edf56100f1b765268145660503a4230000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad727150000"

func TestNttParseArPayloadSuccess(t *testing.T) {
	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000c5bf11ab6ae525ffca02e2af7f6704cdcecec2ea")
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodArPayload)
	require.NoError(t, err)

	success, senderAddress, payload := nttParseArPayload(payload)
	require.True(t, success)
	assert.True(t, bytes.Equal(nttEmitterAddr[:], senderAddress[:]))

	require.NoError(t, err)
	assert.Equal(t, nttPayloadInAr, hex.EncodeToString(payload[:]))
}

func TestNttParseArPayloadWrongDeliveryInstruction(t *testing.T) {
	badArPayload := "02271200000000000000000000000079689ce600d3fd3524ec2b4bedcc70131eda67b60000009f9945ff10000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914007900000000000000070000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7004f994e54540800000000000003e8000000000000000000000000a88085e6370a551cc046fb6b1e3fb9be23ac3a210000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7271200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a120000000000000000000000000000000000000000000000000000000046f5399e7271200000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db810000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db81000000000000000000000000e493cc4f069821404d272b994bb80b1ba163191400"
	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)

	success, _, _ := nttParseArPayload(payload)
	require.False(t, success)
}

func TestNttParseArPayloadTooShort(t *testing.T) {
	badArPayload := "01271200000000000000000000000079689ce600d3fd3524ec2b4bedcc70131eda67b60000009f9945ff10000000000000000000000000e4"
	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)

	success, _, _ := nttParseArPayload(payload)
	require.False(t, success)
}

func TestNttParseArPayloadReallyTooShort(t *testing.T) {
	badArPayload := "01"
	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)

	success, _, _ := nttParseArPayload(payload)
	require.False(t, success)
}

func TestNttParseArPayloadTooLong(t *testing.T) {
	badArPayload := goodArPayload + "00" // Tack one extra byte on the end.
	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)

	success, _, _ := nttParseArPayload(payload)
	require.False(t, success)
}

func TestNttParseArPayloadBadMessageKeyArray(t *testing.T) {
	// The standard good payload has no message keys, so the last byte is zero. Trim that off and add some message keys.
	badArPayload := goodArPayload[0:len(goodArPayload)-2] +
		"03" + // Three message keys
		"01" + "000000000000000000000000000000000000000000000000000000000000000000000000000000000000" + // Valid VAA_KEY_TYPE
		"03" + "00000002" + "abcd" + // Valid some other type
		"04" + "00000004" + "dead" // Some other type that is too short

	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)

	success, _, _ := nttParseArPayload(payload)
	require.False(t, success)
}

func TestNttParseArPayloadBadVaaKeyMessageKey(t *testing.T) {
	// The standard good payload has no message keys, so the last byte is zero. Trim that off and add some message keys.
	badArPayload := goodArPayload[0:len(goodArPayload)-2] +
		"01" + // Three message keys
		"01" + "0000000000000000000000000000000000000000000000000000000000000000000000000000000000" // Invalid VAA_KEY_TYPE that's one byte too short

	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)

	success, _, _ := nttParseArPayload(payload)
	require.False(t, success)
}

func TestNttParseArMsgSuccess(t *testing.T) {
	arEmitterAddr, err := vaa.StringToAddress("0000000000000000000000007b1bd7a6b4e61c2a123ac6bc2cbfc614437d0470")
	require.NoError(t, err)

	arEmitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: arEmitterAddr}: true,
	}

	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000c5bf11ab6ae525ffca02e2af7f6704cdcecec2ea")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: true,
	}

	payload, err := hex.DecodeString(goodArPayload)
	require.NoError(t, err)

	msg := &common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1708575745), 0),
		Nonce:            uint32(0),
		Sequence:         uint64(259),
		EmitterChain:     vaa.ChainIDArbitrumSepolia,
		EmitterAddress:   arEmitterAddr,
		ConsistencyLevel: uint8(15),
		Payload:          payload,
	}

	isNTT, enforceFlag := nttIsMsgArNTT(msg, arEmitters, nttEmitters)
	assert.True(t, isNTT)
	assert.True(t, enforceFlag)
}

func TestNttParseArMsgUnknownArEmitter(t *testing.T) {
	arEmitterAddr, err := vaa.StringToAddress("0000000000000000000000007b1bd7a6b4e61c2a123ac6bc2cbfc614437d0470")
	require.NoError(t, err)

	arEmitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: arEmitterAddr}: true,
	}

	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000c5bf11ab6ae525ffca02e2af7f6704cdcecec2ea")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]bool{
		{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: true,
	}

	differentEmitterAddr, err := vaa.StringToAddress("0000000000000000000000007b1bd7a6b4e61c2a123ac6bc2cbfc614437d0471") // This is different.
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodArPayload)
	require.NoError(t, err)

	msg := &common.MessagePublication{
		TxHash:           hashFromString("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1708575745), 0),
		Nonce:            uint32(0),
		Sequence:         uint64(259),
		EmitterChain:     vaa.ChainIDArbitrumSepolia,
		EmitterAddress:   differentEmitterAddr,
		ConsistencyLevel: uint8(15),
		Payload:          payload,
	}

	isNTT, _ := nttIsMsgArNTT(msg, arEmitters, nttEmitters)
	assert.False(t, isNTT)
}

func TestNttVerifyMainnetEmitters(t *testing.T) {
	directEmitters, arEmitters, err := nttGetEmitters(common.MainNet)
	require.NoError(t, err)
	assert.Equal(t, 5, len(directEmitters)) // TODO: Change this when we add a mainnet emitter!
	assert.NotEqual(t, 0, len(arEmitters))
}

func TestNttVerifyTestnetEmitters(t *testing.T) {
	directEmitters, arEmitters, err := nttGetEmitters(common.TestNet)
	require.NoError(t, err)
	assert.NotEqual(t, 0, len(directEmitters))
	assert.NotEqual(t, 0, len(arEmitters))
}

func TestNttVerifyDevnetEmitters(t *testing.T) {
	directEmitters, arEmitters, err := nttGetEmitters(common.UnsafeDevNet)
	require.NoError(t, err)
	assert.NotEqual(t, 0, len(directEmitters))
	assert.NotEqual(t, 0, len(arEmitters))
}
