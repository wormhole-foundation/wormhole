package vaa

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var addr = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
var dummyBytes = [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

func TestCoreModule(t *testing.T) {
	hexifiedCoreModule := "00000000000000000000000000000000000000000000000000000000436f7265"
	assert.Equal(t, hex.EncodeToString(CoreModule), hexifiedCoreModule)
}

func TestBodyContractUpgrade(t *testing.T) {
	test := BodyContractUpgrade{ChainID: 1, NewContract: addr}
	assert.Equal(t, test.ChainID, ChainID(1))
	assert.Equal(t, test.NewContract, addr)
}

func TestBodyGuardianSetUpdate(t *testing.T) {
	keys := []common.Address{
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
	}
	test := BodyGuardianSetUpdate{Keys: keys, NewIndex: uint32(1)}
	assert.Equal(t, test.Keys, keys)
	assert.Equal(t, test.NewIndex, uint32(1))
}

func TestBodyTokenBridgeRegisterChain(t *testing.T) {
	module := "test"
	test := BodyTokenBridgeRegisterChain{Module: module, ChainID: 1, EmitterAddress: addr}
	assert.Equal(t, test.Module, module)
	assert.Equal(t, test.ChainID, ChainID(1))
	assert.Equal(t, test.EmitterAddress, addr)
}

func TestBodyTokenBridgeUpgradeContract(t *testing.T) {
	module := "test"
	test := BodyTokenBridgeUpgradeContract{Module: module, TargetChainID: 1, NewContract: addr}
	assert.Equal(t, test.Module, module)
	assert.Equal(t, test.TargetChainID, ChainID(1))
	assert.Equal(t, test.NewContract, addr)
}

func TestBodyContractUpgradeSerialize(t *testing.T) {
	bodyContractUpgrade := BodyContractUpgrade{ChainID: 1, NewContract: addr}
	expected := "00000000000000000000000000000000000000000000000000000000436f72650100010000000000000000000000000000000000000000000000000000000000000004"
	serializedBodyContractUpgrade, err := bodyContractUpgrade.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyContractUpgrade))
}

func TestBodyGuardianSetUpdateSerialize(t *testing.T) {
	keys := []common.Address{
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
	}
	bodyGuardianSetUpdate := BodyGuardianSetUpdate{Keys: keys, NewIndex: uint32(1)}
	expected := "00000000000000000000000000000000000000000000000000000000436f726502000000000001025aaeb6053f3e94c9b9a09f33669435e7ef1beaed5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"
	serializedBodyGuardianSetUpdate, err := bodyGuardianSetUpdate.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyGuardianSetUpdate))
}

func TestBodyTokenBridgeRegisterChainSerialize(t *testing.T) {
	module := "test"
	tests := []struct {
		name     string
		expected string
		object   BodyTokenBridgeRegisterChain
		err      error
	}{
		{
			name:     "working_as_expected",
			err:      nil,
			object:   BodyTokenBridgeRegisterChain{Module: module, ChainID: 1, EmitterAddress: addr},
			expected: "000000000000000000000000000000000000000000000000000000007465737401000000010000000000000000000000000000000000000000000000000000000000000004",
		},
		{
			name:     "panic_at_the_disco!",
			err:      errors.New("payload longer than 32 bytes"),
			object:   BodyTokenBridgeRegisterChain{Module: "123456789012345678901234567890123", ChainID: 1, EmitterAddress: addr},
			expected: "payload longer than 32 bytes",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			buf, err := testCase.object.Serialize()
			if testCase.err != nil {
				require.ErrorContains(t, err, testCase.err.Error())
				assert.Nil(t, buf)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, hex.EncodeToString(buf))
			}
		})
	}
}

func TestBodyTokenBridgeUpgradeContractSerialize(t *testing.T) {
	module := "test"
	bodyTokenBridgeUpgradeContract := BodyTokenBridgeUpgradeContract{Module: module, TargetChainID: 1, NewContract: addr}
	expected := "00000000000000000000000000000000000000000000000000000000746573740200010000000000000000000000000000000000000000000000000000000000000004"
	serializedBodyTokenBridgeUpgradeContract, err := bodyTokenBridgeUpgradeContract.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyTokenBridgeUpgradeContract))
}

func TestBodyWormchainStoreCodeSerialize(t *testing.T) {
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65010c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	bodyWormchainStoreCode := BodyWormchainStoreCode{WasmHash: dummyBytes}
	buf, err := bodyWormchainStoreCode.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyWormchainInstantiateContractSerialize(t *testing.T) {
	actual := BodyWormchainInstantiateContract{InstantiationParamsHash: dummyBytes}
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65020c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	buf, err := actual.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyWormchainMigrateContractSerialize(t *testing.T) {
	actual := BodyWormchainMigrateContract{MigrationParamsHash: dummyBytes}
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65030c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	buf, err := actual.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyWormchainWasmAllowlistInstantiateSerialize(t *testing.T) {
	actual := BodyWormchainWasmAllowlistInstantiate{ContractAddr: dummyBytes, CodeId: uint64(42)}
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65040c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20000000000000002a"
	buf, err := actual.Serialize(ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

const BodyWormchainWasmAllowlistInstantiateBuf = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20000000000000002a"

func TestBodyWormchainWasmAllowlistInstantiateDeserialize(t *testing.T) {
	expected := BodyWormchainWasmAllowlistInstantiate{ContractAddr: dummyBytes, CodeId: uint64(42)}
	buf, err := hex.DecodeString(BodyWormchainWasmAllowlistInstantiateBuf)
	require.NoError(t, err)

	var actual BodyWormchainWasmAllowlistInstantiate
	err = actual.Deserialize(buf)
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(expected, actual))
}

func TestBodyWormchainWasmAllowlistInstantiateDeserializeFailureTooShort(t *testing.T) {
	buf, err := hex.DecodeString(BodyWormchainWasmAllowlistInstantiateBuf[0 : len(BodyWormchainWasmAllowlistInstantiateBuf)-2])
	require.NoError(t, err)

	var actual BodyWormchainWasmAllowlistInstantiate
	err = actual.Deserialize(buf)
	require.ErrorContains(t, err, "incorrect payload length, should be 40, is 39")
}

func TestBodyWormchainWasmAllowlistInstantiateDeserializeFailureTooLong(t *testing.T) {
	buf, err := hex.DecodeString(BodyWormchainWasmAllowlistInstantiateBuf + "00")
	require.NoError(t, err)

	var actual BodyWormchainWasmAllowlistInstantiate
	err = actual.Deserialize(buf)
	require.ErrorContains(t, err, "incorrect payload length, should be 40, is 41")
}

func TestBodyCircleIntegrationUpdateWormholeFinalitySerialize(t *testing.T) {
	expected := "000000000000000000000000000000436972636c65496e746567726174696f6e0100022a"
	bodyCircleIntegrationUpdateWormholeFinality := BodyCircleIntegrationUpdateWormholeFinality{TargetChainID: ChainIDEthereum, Finality: 42}
	buf, err := bodyCircleIntegrationUpdateWormholeFinality.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyCircleIntegrationRegisterEmitterAndDomainSerialize(t *testing.T) {
	expected := "000000000000000000000000000000436972636c65496e746567726174696f6e020002000600000000000000000000000000000000000000000000000000000000000000040000002a"
	bodyCircleIntegrationRegisterEmitterAndDomain := BodyCircleIntegrationRegisterEmitterAndDomain{
		TargetChainID:         ChainIDEthereum,
		ForeignEmitterChainId: ChainIDAvalanche,
		ForeignEmitterAddress: addr,
		CircleDomain:          42,
	}
	buf, err := bodyCircleIntegrationRegisterEmitterAndDomain.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyCircleIntegrationUpgradeContractImplementationSerialize(t *testing.T) {
	expected := "000000000000000000000000000000436972636c65496e746567726174696f6e0300020000000000000000000000000000000000000000000000000000000000000004"
	bodyCircleIntegrationUpgradeContractImplementation := BodyCircleIntegrationUpgradeContractImplementation{
		TargetChainID:            ChainIDEthereum,
		NewImplementationAddress: addr,
	}
	buf, err := bodyCircleIntegrationUpgradeContractImplementation.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyIbcReceiverUpdateChannelChain(t *testing.T) {
	expected := "0000000000000000000000000000000000000000004962635265636569766572010c20000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d300013"

	channelId, err := LeftPadIbcChannelId("channel-0")
	require.NoError(t, err)

	bodyIbcReceiverUpdateChannelChain := BodyIbcUpdateChannelChain{
		TargetChainId: ChainIDWormchain,
		ChannelId:     channelId,
		ChainId:       ChainIDInjective,
	}
	buf, err := bodyIbcReceiverUpdateChannelChain.Serialize(IbcReceiverModuleStr)
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyIbcReceiverUpdateChannelChainBadModuleName(t *testing.T) {
	channelId, err := LeftPadIbcChannelId("channel-0")
	require.NoError(t, err)

	bodyIbcReceiverUpdateChannelChain := BodyIbcUpdateChannelChain{
		TargetChainId: ChainIDWormchain,
		ChannelId:     channelId,
		ChainId:       ChainIDInjective,
	}
	buf, err := bodyIbcReceiverUpdateChannelChain.Serialize(IbcReceiverModuleStr + "ExtraJunk")
	require.ErrorContains(t, err, "module for BodyIbcUpdateChannelChain must be either IbcReceiver or IbcTranslator")
	assert.Nil(t, buf)
}

func TestLeftPadIbcChannelId(t *testing.T) {
	channelId, err := LeftPadIbcChannelId("channel-0")
	require.NoError(t, err)
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d30", hex.EncodeToString(channelId[:]))
}

func TestLeftPadIbcChannelIdFailureTooLong(t *testing.T) {
	channelId, err := LeftPadIbcChannelId("channel-ThatIsTooLong!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	require.ErrorContains(t, err, "failed to left pad module: payload longer than 64 bytes")
	expected := [64]byte{}
	assert.True(t, bytes.Equal(expected[:], channelId[:]))
}

func TestLeftPadBytes(t *testing.T) {
	payload := "AAAA"
	paddedPayload, err := LeftPadBytes(payload, int(8))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.Write([]byte(payload))

	assert.Equal(t, paddedPayload, buf)
}

func TestLeftPadBytesFailures(t *testing.T) {
	payload := "AAAA"

	paddedPayload, err := LeftPadBytes(payload, int(-2))
	require.ErrorContains(t, err, "cannot prepend bytes to a negative length buffer")
	assert.Nil(t, paddedPayload)

	paddedPayload, err = LeftPadBytes(payload, int(2))
	require.ErrorContains(t, err, "payload longer than 2 bytes")
	assert.Nil(t, paddedPayload)
}

func TestSerializeBridgeGovernanceVaaModuleTooLong(t *testing.T) {
	buf, err := serializeBridgeGovernanceVaa("ModuleNameIsMoreThanThirtyTwoCharacters", ActionRegisterChain, 1, []byte{0, 1, 2})
	require.ErrorContains(t, err, "failed to left pad module: payload longer than 32 bytes")
	assert.Nil(t, buf)
}

func FuzzLeftPadBytes(f *testing.F) {
	// Add examples to our fuzz corpus
	f.Add("FOO", 8)
	f.Add("123", 8)

	f.Fuzz(func(t *testing.T, payload string, length int) {
		// We know length could be negative, but we panic if it is in the implementation
		if length < 0 {
			t.Skip()
		}

		// We know we cannot left pad something shorter than the payload being provided, but we panic if it is
		if len(payload) > length {
			t.Skip()
		}

		paddedPayload, err := LeftPadBytes(payload, length)
		require.NoError(t, err)

		// paddedPayload must always be equal to length
		assert.Equal(t, paddedPayload.Len(), length)
	})
}

func TestBodyWormholeRelayerSetDefaultDeliveryProviderSerialize(t *testing.T) {
	expected := "0000000000000000000000000000000000576f726d686f6c6552656c617965720300040000000000000000000000000000000000000000000000000000000000000004"
	bodyWormholeRelayerSetDefaultDeliveryProvider := BodyWormholeRelayerSetDefaultDeliveryProvider{
		ChainID:                           4,
		NewDefaultDeliveryProviderAddress: addr,
	}
	buf, err := bodyWormholeRelayerSetDefaultDeliveryProvider.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyGatewayIbcComposabilityMwContractSerialize(t *testing.T) {
	expected := "00000000000000000000000000000000000000476174657761794d6f64756c65030c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	bodyGatewayIbcComposabilityMwContract := BodyGatewayIbcComposabilityMwContract{
		ContractAddr: dummyBytes,
	}
	buf, err := bodyGatewayIbcComposabilityMwContract.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

const BodyGatewayIbcComposabilityMwContractBuf = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func TestBodyGatewayIbcComposabilityMwContractDeserialize(t *testing.T) {
	expected := BodyGatewayIbcComposabilityMwContract{
		ContractAddr: dummyBytes,
	}
	var payloadBody BodyGatewayIbcComposabilityMwContract
	err := payloadBody.Deserialize(dummyBytes[:])
	require.NoError(t, err)
	assert.Equal(t, expected, payloadBody)
}

func TestBodyGatewayIbcComposabilityMwContractDeserializeFailureTooShort(t *testing.T) {
	buf, err := hex.DecodeString(BodyGatewayIbcComposabilityMwContractBuf[0 : len(BodyGatewayIbcComposabilityMwContractBuf)-2])
	require.NoError(t, err)

	var actual BodyGatewayIbcComposabilityMwContract
	err = actual.Deserialize(buf)
	require.ErrorContains(t, err, "incorrect payload length, should be 32, is 31")
}

func TestBodyGatewayIbcComposabilityMwContractDeserializeFailureTooLong(t *testing.T) {
	buf, err := hex.DecodeString(BodyGatewayIbcComposabilityMwContractBuf + "00")
	require.NoError(t, err)

	var actual BodyGatewayIbcComposabilityMwContract
	err = actual.Deserialize(buf)
	require.ErrorContains(t, err, "incorrect payload length, should be 32, is 33")
}

func TestBodyCoreRecoverChainIdSerialize(t *testing.T) {
	expected := "00000000000000000000000000000000000000000000000000000000436f72650500000000000000000000000000000000000000000000000000000000000000010fa0"
	BodyRecoverChainId := BodyRecoverChainId{
		Module:     "Core",
		EvmChainID: uint256.NewInt(1),
		NewChainID: 4000,
	}
	buf, err := BodyRecoverChainId.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyTokenBridgeRecoverChainIdSerialize(t *testing.T) {
	expected := "000000000000000000000000000000000000000000546f6b656e4272696467650300000000000000000000000000000000000000000000000000000000000000010fa0"
	BodyRecoverChainId := BodyRecoverChainId{
		Module:     "TokenBridge",
		EvmChainID: uint256.NewInt(1),
		NewChainID: 4000,
	}
	buf, err := BodyRecoverChainId.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestBodyRecoverChainIdModuleTooLong(t *testing.T) {
	BodyRecoverChainId := BodyRecoverChainId{
		Module:     "ModuleNameIsMoreThanThirtyTwoCharacters",
		EvmChainID: uint256.NewInt(1),
		NewChainID: 4000,
	}
	buf, err := BodyRecoverChainId.Serialize()
	require.ErrorContains(t, err, "failed to left pad module: payload longer than 32 bytes")
	assert.Nil(t, buf)
}
