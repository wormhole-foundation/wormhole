package accountant

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const goodPayload = "9945ff10042942fafabe00000000000000000000000000000000000000000000000000000079000000367999a1014667921341234300000000000000000000000000000000000000000000000000004f994e545407000000000012d687beefface00000000000000000000000000000000000000000000000000000000feebcafe0000000000000000000000000000000000000000000000000000000000110000"

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

	emitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDEthereum, emitterAddr: emitterAddr}: {},
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

	assert.True(t, nttIsMsgDirectNTT(msg, emitters))
}

func TestNttParseMsgWrongEmitterChain(t *testing.T) {
	emitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e74")
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodPayload)
	require.NoError(t, err)

	emitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDEthereum, emitterAddr: emitterAddr}: {},
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

	assert.False(t, nttIsMsgDirectNTT(msg, emitters))
}

func TestNttParseMsgWrongEmitterAddress(t *testing.T) {
	goodEmitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e74")
	require.NoError(t, err)

	badEmitterAddr, err := vaa.StringToAddress("000000000000000000000000000000000000000000000000656e64706f696e75")
	require.NoError(t, err)

	payload, err := hex.DecodeString(goodPayload)
	require.NoError(t, err)

	emitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDEthereum, emitterAddr: goodEmitterAddr}: {},
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

	assert.False(t, nttIsMsgDirectNTT(msg, emitters))
}

const arPayload = "01271200000000000000000000000079689ce600d3fd3524ec2b4bedcc70131eda67b60000009f9945ff10000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914007900000000000000070000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7004f994e54540800000000000003e8000000000000000000000000a88085e6370a551cc046fb6b1e3fb9be23ac3a210000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7271200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a120000000000000000000000000000000000000000000000000000000046f5399e7271200000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db810000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db81000000000000000000000000e493cc4f069821404d272b994bb80b1ba163191400"

func TestNttParseArPayloadSuccess(t *testing.T) {
	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	payload, err := hex.DecodeString(arPayload)
	require.NoError(t, err)
	assert.True(t, nttIsArPayloadNTT(vaa.ChainIDArbitrumSepolia, payload, nttEmitters))
}

func TestNttParseArPayloadWrongDeliveryInstruction(t *testing.T) {
	badArPayload := "02271200000000000000000000000079689ce600d3fd3524ec2b4bedcc70131eda67b60000009f9945ff10000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914007900000000000000070000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7004f994e54540800000000000003e8000000000000000000000000a88085e6370a551cc046fb6b1e3fb9be23ac3a210000000000000000000000008f26a0025dccc6cfc07a7d38756280a10e295ad7271200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a120000000000000000000000000000000000000000000000000000000046f5399e7271200000000000000000000000000000000000000000000000000000000000000000000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db810000000000000000000000007a0a53847776f7e94cc35742971acb2217b0db81000000000000000000000000e493cc4f069821404d272b994bb80b1ba163191400"
	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)
	assert.False(t, nttIsArPayloadNTT(vaa.ChainIDArbitrumSepolia, payload, nttEmitters))
}

func TestNttParseArPayloadTooShort(t *testing.T) {
	badArPayload := "01271200000000000000000000000079689ce600d3fd3524ec2b4bedcc70131eda67b60000009f9945ff10000000000000000000000000e4"
	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)
	assert.False(t, nttIsArPayloadNTT(vaa.ChainIDArbitrumSepolia, payload, nttEmitters))
}

func TestNttParseArPayloadReallyTooShort(t *testing.T) {
	badArPayload := "01"
	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)
	assert.False(t, nttIsArPayloadNTT(vaa.ChainIDArbitrumSepolia, payload, nttEmitters))
}

func TestNttParseArPayloadUnknownNttEmitter(t *testing.T) {
	badArPayload := arPayload
	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631915") // This is different.
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	payload, err := hex.DecodeString(badArPayload)
	require.NoError(t, err)
	assert.False(t, nttIsArPayloadNTT(vaa.ChainIDArbitrumSepolia, payload, nttEmitters))
}

func TestNttParseArMsgSuccess(t *testing.T) {
	arEmitterAddr, err := vaa.StringToAddress("0000000000000000000000007b1bd7a6b4e61c2a123ac6bc2cbfc614437d0470")
	require.NoError(t, err)

	arEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: arEmitterAddr}: {},
	}

	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	payload, err := hex.DecodeString(arPayload)
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

	assert.True(t, nttIsMsgArNTT(msg, arEmitters, nttEmitters))
}

func TestNttParseArMsgUnknownArEmitter(t *testing.T) {
	arEmitterAddr, err := vaa.StringToAddress("0000000000000000000000007b1bd7a6b4e61c2a123ac6bc2cbfc614437d0470")
	require.NoError(t, err)

	arEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: arEmitterAddr}: {},
	}

	nttEmitterAddr, err := vaa.StringToAddress("000000000000000000000000e493cc4f069821404d272b994bb80b1ba1631914")
	require.NoError(t, err)

	nttEmitters := map[emitterKey]struct{}{
		emitterKey{emitterChainId: vaa.ChainIDArbitrumSepolia, emitterAddr: nttEmitterAddr}: {},
	}

	differentEmitterAddr, err := vaa.StringToAddress("0000000000000000000000007b1bd7a6b4e61c2a123ac6bc2cbfc614437d0471") // This is different.
	require.NoError(t, err)

	payload, err := hex.DecodeString(arPayload)
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

	assert.False(t, nttIsMsgArNTT(msg, arEmitters, nttEmitters))
}
