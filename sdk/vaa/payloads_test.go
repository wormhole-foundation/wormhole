package vaa

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"

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

func TestBodySlashingParamsUpdateSerialize(t *testing.T) {
	signedBlocksWindow := uint64(100)
	minSignedPerWindow := uint64(500000000000000000)
	downtimeJailDuration := uint64(600 * time.Second)
	slashFractionDoubleSign := uint64(50000000000000000)
	slashFractionDowntime := uint64(10000000000000000)

	bodySlashingParamsUpdate := BodyGatewaySlashingParamsUpdate{
		SignedBlocksWindow:      signedBlocksWindow,
		MinSignedPerWindow:      minSignedPerWindow,
		DowntimeJailDuration:    downtimeJailDuration,
		SlashFractionDoubleSign: slashFractionDoubleSign,
		SlashFractionDowntime:   slashFractionDowntime,
	}
	serializedBody, err := bodySlashingParamsUpdate.Serialize()
	require.NoError(t, err)

	expected := "00000000000000000000000000000000000000476174657761794d6f64756c65040c20000000000000006406f05b59d3b200000000008bb2c9700000b1a2bc2ec50000002386f26fc10000"
	assert.Equal(t, expected, hex.EncodeToString(serializedBody))
}

const BodySlashingParamsUpdateBuf = "000000000000006406f05b59d3b200000000008bb2c9700000b1a2bc2ec50000002386f26fc10000"

func TestBodySlashingParamsUpdateDeserialize(t *testing.T) {
	expected := BodyGatewaySlashingParamsUpdate{
		SignedBlocksWindow:      100,
		MinSignedPerWindow:      500000000000000000,
		DowntimeJailDuration:    uint64(600 * time.Second),
		SlashFractionDoubleSign: 50000000000000000,
		SlashFractionDowntime:   10000000000000000,
	}
	var payloadBody BodyGatewaySlashingParamsUpdate
	bz, err := hex.DecodeString(BodySlashingParamsUpdateBuf)
	require.NoError(t, err)
	err = payloadBody.Deserialize(bz)
	require.NoError(t, err)
	assert.Equal(t, expected, payloadBody)
}

func TestBodySlashingParamsUpdateDeserializeFailureTooShort(t *testing.T) {
	buf, err := hex.DecodeString(BodySlashingParamsUpdateBuf[0 : len(BodySlashingParamsUpdateBuf)-2])
	require.NoError(t, err)

	var actual BodyGatewaySlashingParamsUpdate
	err = actual.Deserialize(buf)
	require.ErrorContains(t, err, "incorrect payload length, should be 40, is 39")
}

func TestBodySlashingParamsUpdateDeserializeFailureTooLong(t *testing.T) {
	buf, err := hex.DecodeString(BodySlashingParamsUpdateBuf + "00")
	require.NoError(t, err)

	var actual BodyGatewaySlashingParamsUpdate
	err = actual.Deserialize(buf)
	require.ErrorContains(t, err, "incorrect payload length, should be 40, is 41")
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

func TestBodyCoreBridgeSetMessageFeeSerialize(t *testing.T) {
	expected := "00000000000000000000000000000000000000000000000000000000436f72650304560000000000000000000000000000000000000000000000000000000000000123"
	bodyCoreBridgeSetMessageFee := BodyCoreBridgeSetMessageFee{
		ChainID:    0x456,
		MessageFee: uint256.NewInt(0x123),
	}
	buf, err := bodyCoreBridgeSetMessageFee.Serialize()
	require.NoError(t, err)
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestDelegatedManagerModule(t *testing.T) {
	expected := "0000000000000000000000000000000044656c6567617465644d616e61676572"
	assert.Equal(t, expected, hex.EncodeToString(DelegatedManagerModule[:]))
}

func TestBodyManagerSetUpdateSerialize(t *testing.T) {
	// Create a simple manager set with 2 public keys
	managerSet := Secp256k1MultisigManagerSet{
		M: 2,
		N: 2,
		PublicKeys: [][CompressedSecp256k1PublicKeyLength]byte{
			{0x02, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
			{0x03, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
		},
	}
	managerSetBytes, err := managerSet.Serialize()
	require.NoError(t, err)

	bodyManagerSetUpdate := BodyManagerSetUpdate{
		ManagerChainID:     ChainIDDogecoin,
		NewManagerSetIndex: 1,
		NewManagerSet:      managerSetBytes,
	}

	buf, err := bodyManagerSetUpdate.Serialize()
	require.NoError(t, err)

	// Verify the serialized output structure:
	// - Module: "DelegatedManager" (32 bytes)
	// - Action: 1 (1 byte)
	// - Chain: 0 (2 bytes, universal)
	// - ManagerChainID: ChainIDDogecoin (2 bytes)
	// - NewManagerSetIndex: 1 (4 bytes)
	// - NewManagerSet: managerSetBytes (variable)
	expected := "0000000000000000000000000000000044656c6567617465644d616e61676572" + // Module
		"01" + // Action
		"0000" + // Chain (universal)
		"0041" + // ManagerChainID (ChainIDDogecoin = 65)
		"00000001" + // NewManagerSetIndex
		hex.EncodeToString(managerSetBytes) // NewManagerSet
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

const BodyManagerSetUpdatePayloadBuf = "0041" + // ManagerChainID (ChainIDDogecoin = 65)
	"00000001" + // NewManagerSetIndex
	"01" + // Type (Secp256k1Multisig)
	"02" + // M
	"02" + // N
	"020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + // PublicKey 1 (33 bytes)
	"032122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40" // PublicKey 2 (33 bytes)

func TestBodyManagerSetUpdateDeserialize(t *testing.T) {
	buf, err := hex.DecodeString(BodyManagerSetUpdatePayloadBuf)
	require.NoError(t, err)

	var actual BodyManagerSetUpdate
	err = actual.Deserialize(buf)
	require.NoError(t, err)

	assert.Equal(t, ChainIDDogecoin, actual.ManagerChainID)
	assert.Equal(t, uint32(1), actual.NewManagerSetIndex)
	assert.NotEmpty(t, actual.NewManagerSet)

	// Further deserialize the manager set
	var managerSet Secp256k1MultisigManagerSet
	err = managerSet.Deserialize(actual.NewManagerSet)
	require.NoError(t, err)
	assert.Equal(t, uint8(2), managerSet.M)
	assert.Equal(t, uint8(2), managerSet.N)
	assert.Len(t, managerSet.PublicKeys, 2)
}

func TestBodyManagerSetUpdateDeserializeFailureTooShort(t *testing.T) {
	buf, err := hex.DecodeString("001e00000001") // Only 6 bytes, missing manager set
	require.NoError(t, err)

	var actual BodyManagerSetUpdate
	err = actual.Deserialize(buf[:5]) // Only 5 bytes
	require.ErrorContains(t, err, "incorrect payload length, should be at least 6 bytes, is 5")
}

func TestSecp256k1MultisigManagerSetSerialize(t *testing.T) {
	managerSet := Secp256k1MultisigManagerSet{
		M: 2,
		N: 3,
		PublicKeys: [][CompressedSecp256k1PublicKeyLength]byte{
			{0x02, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
			{0x03, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
			{0x02, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60},
		},
	}

	buf, err := managerSet.Serialize()
	require.NoError(t, err)

	// Expected format: Type (1) + M (1) + N (1) + Keys (3 * 33)
	expected := "01" + // Type
		"02" + // M
		"03" + // N
		"020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + // Key 1
		"032122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40" + // Key 2
		"024142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60" // Key 3

	// Check the length and structure
	assert.Equal(t, 1+1+1+3*CompressedSecp256k1PublicKeyLength, len(buf))
	assert.Equal(t, byte(ManagerSetTypeSecp256k1Multisig), buf[0]) // Type
	assert.Equal(t, byte(2), buf[1])                               // M
	assert.Equal(t, byte(3), buf[2])                               // N

	// Compare public keys
	assert.Equal(t, expected, hex.EncodeToString(buf))
}

func TestSecp256k1MultisigManagerSetSerializeValidation(t *testing.T) {
	tests := []struct {
		name        string
		managerSet  Secp256k1MultisigManagerSet
		expectedErr string
	}{
		{
			name: "N does not match keys",
			managerSet: Secp256k1MultisigManagerSet{
				M:          2,
				N:          3,
				PublicKeys: [][CompressedSecp256k1PublicKeyLength]byte{{}, {}}, // Only 2 keys but N=3
			},
			expectedErr: "n (3) does not match number of public keys (2)",
		},
		{
			name: "M greater than N",
			managerSet: Secp256k1MultisigManagerSet{
				M:          3,
				N:          2,
				PublicKeys: [][CompressedSecp256k1PublicKeyLength]byte{{}, {}},
			},
			expectedErr: "m (3) cannot be greater than n (2)",
		},
		{
			name: "M is zero",
			managerSet: Secp256k1MultisigManagerSet{
				M:          0,
				N:          2,
				PublicKeys: [][CompressedSecp256k1PublicKeyLength]byte{{}, {}},
			},
			expectedErr: "m must be at least 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.managerSet.Serialize()
			require.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

const Secp256k1MultisigManagerSetBuf = "01" + // Type
	"02" + // M
	"03" + // N
	"020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + // Key 1
	"032122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40" + // Key 2
	"024142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60" // Key 3

func TestSecp256k1MultisigManagerSetDeserialize(t *testing.T) {
	buf, err := hex.DecodeString(Secp256k1MultisigManagerSetBuf)
	require.NoError(t, err)

	var actual Secp256k1MultisigManagerSet
	err = actual.Deserialize(buf)
	require.NoError(t, err)

	assert.Equal(t, uint8(2), actual.M)
	assert.Equal(t, uint8(3), actual.N)
	assert.Len(t, actual.PublicKeys, 3)

	// Verify first byte of each key matches expected prefix
	assert.Equal(t, byte(0x02), actual.PublicKeys[0][0])
	assert.Equal(t, byte(0x03), actual.PublicKeys[1][0])
	assert.Equal(t, byte(0x02), actual.PublicKeys[2][0])
}

func TestSecp256k1MultisigManagerSetDeserializeFailures(t *testing.T) {
	tests := []struct {
		name        string
		hexInput    string
		expectedErr string
	}{
		{
			name:        "too short",
			hexInput:    "0102", // Only 2 bytes, missing N
			expectedErr: "payload too short, expected at least 3 bytes, got 2",
		},
		{
			name:        "wrong type",
			hexInput:    "020203" + "020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + "032122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40" + "024142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60",
			expectedErr: "unexpected manager set type 2, expected 1",
		},
		{
			name:        "M greater than N",
			hexInput:    "010302" + "020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + "032122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40", // M=3, N=2
			expectedErr: "m (3) cannot be greater than n (2)",
		},
		{
			name:        "M is zero",
			hexInput:    "010002" + "020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20" + "032122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40", // M=0
			expectedErr: "m must be at least 1",
		},
		{
			name:        "payload length mismatch",
			hexInput:    "010202" + "020102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20", // Only 1 key but N=2
			expectedErr: "payload length mismatch",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := hex.DecodeString(tc.hexInput)
			require.NoError(t, err)

			var actual Secp256k1MultisigManagerSet
			err = actual.Deserialize(buf)
			require.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func TestSecp256k1MultisigManagerSetRoundTrip(t *testing.T) {
	original := Secp256k1MultisigManagerSet{
		M: 5,
		N: 7,
		PublicKeys: [][CompressedSecp256k1PublicKeyLength]byte{
			{0x02, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20},
			{0x03, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40},
			{0x02, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f, 0x60},
			{0x03, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f, 0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79, 0x7a, 0x7b, 0x7c, 0x7d, 0x7e, 0x7f, 0x80},
			{0x02, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f, 0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99, 0x9a, 0x9b, 0x9c, 0x9d, 0x9e, 0x9f, 0xa0},
			{0x03, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf, 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, 0xb8, 0xb9, 0xba, 0xbb, 0xbc, 0xbd, 0xbe, 0xbf, 0xc0},
			{0x02, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7, 0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf, 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7, 0xd8, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde, 0xdf, 0xe0},
		},
	}

	// Serialize
	buf, err := original.Serialize()
	require.NoError(t, err)

	// Deserialize
	var deserialized Secp256k1MultisigManagerSet
	err = deserialized.Deserialize(buf)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.M, deserialized.M)
	assert.Equal(t, original.N, deserialized.N)
	assert.Equal(t, original.PublicKeys, deserialized.PublicKeys)
}

// Test data for UTXO payload tests
var (
	testRecipientAddress = [32]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
		0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
	}
	testTransactionID = [32]byte{
		0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28,
		0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, 0x30,
		0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38,
		0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, 0x40,
	}
	testP2PKHAddress = []byte{
		0x55, 0xae, 0x51, 0x68, 0x4c, 0x43, 0x43, 0x5d,
		0xa7, 0x51, 0xac, 0x8d, 0x21, 0x73, 0xb2, 0x65,
		0x2e, 0xb6, 0x41, 0x05,
	}
	testP2SHAddress = []byte{
		0x74, 0x82, 0x84, 0x39, 0x0f, 0x9e, 0x26, 0x3a,
		0x4b, 0x76, 0x6a, 0x75, 0xd0, 0x63, 0x3c, 0x50,
		0x42, 0x6e, 0xb8, 0x75,
	}
)

func TestUTXOPayloadPrefix(t *testing.T) {
	expected := [4]byte{'U', 'T', 'X', '0'}
	assert.Equal(t, expected, UTXOPayloadPrefix)
	assert.Equal(t, "UTX0", string(UTXOPayloadPrefix[:]))
}

func TestUTXOAddressTypeLength(t *testing.T) {
	tests := []struct {
		name        string
		addrType    UTXOAddressType
		expectedLen int
		expectError bool
	}{
		{
			name:        "P2PKH",
			addrType:    UTXOAddressTypeP2PKH,
			expectedLen: 20,
			expectError: false,
		},
		{
			name:        "P2SH",
			addrType:    UTXOAddressTypeP2SH,
			expectedLen: 20,
			expectError: false,
		},
		{
			name:        "Unknown type",
			addrType:    UTXOAddressType(99),
			expectedLen: 0,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			length, err := tc.addrType.AddressLength()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedLen, length)
			}
		})
	}
}

func TestUTXOInputSerialize(t *testing.T) {
	input := UTXOInput{
		OriginalRecipientAddress: testRecipientAddress,
		TransactionID:            testTransactionID,
		Vout:                     42,
	}

	buf := input.Serialize()
	assert.Len(t, buf, 68)

	// Verify structure
	assert.Equal(t, testRecipientAddress[:], buf[0:32])
	assert.Equal(t, testTransactionID[:], buf[32:64])
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x2a}, buf[64:68]) // vout = 42
}

func TestDeserializeUTXOInput(t *testing.T) {
	// Create expected input
	expected := &UTXOInput{
		OriginalRecipientAddress: testRecipientAddress,
		TransactionID:            testTransactionID,
		Vout:                     42,
	}

	// Serialize
	buf := expected.Serialize()

	// Deserialize
	actual, err := DeserializeUTXOInput(buf)
	require.NoError(t, err)

	assert.Equal(t, expected.OriginalRecipientAddress, actual.OriginalRecipientAddress)
	assert.Equal(t, expected.TransactionID, actual.TransactionID)
	assert.Equal(t, expected.Vout, actual.Vout)
}

func TestDeserializeUTXOInputTooShort(t *testing.T) {
	buf := make([]byte, 67) // One byte too short
	_, err := DeserializeUTXOInput(buf)
	require.ErrorContains(t, err, "UTXO input too short")
}

func TestUTXOOutputSerialize(t *testing.T) {
	tests := []struct {
		name        string
		output      UTXOOutput
		expectError bool
	}{
		{
			name: "P2PKH output",
			output: UTXOOutput{
				Amount:      1000000,
				AddressType: UTXOAddressTypeP2PKH,
				Address:     testP2PKHAddress,
			},
			expectError: false,
		},
		{
			name: "P2SH output",
			output: UTXOOutput{
				Amount:      2000000,
				AddressType: UTXOAddressTypeP2SH,
				Address:     testP2SHAddress,
			},
			expectError: false,
		},
		{
			name: "Wrong address length",
			output: UTXOOutput{
				Amount:      1000000,
				AddressType: UTXOAddressTypeP2PKH,
				Address:     []byte{0x01, 0x02}, // Too short
			},
			expectError: true,
		},
		{
			name: "Unknown address type",
			output: UTXOOutput{
				Amount:      1000000,
				AddressType: UTXOAddressType(99),
				Address:     testP2PKHAddress,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := tc.output.Serialize()
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, buf, 12+len(tc.output.Address))
			}
		})
	}
}

func TestDeserializeUTXOOutput(t *testing.T) {
	expected := &UTXOOutput{
		Amount:      1000000,
		AddressType: UTXOAddressTypeP2PKH,
		Address:     testP2PKHAddress,
	}

	// Serialize
	buf, err := expected.Serialize()
	require.NoError(t, err)

	// Deserialize
	actual, size, err := DeserializeUTXOOutput(buf)
	require.NoError(t, err)
	assert.Equal(t, len(buf), size)
	assert.Equal(t, expected.Amount, actual.Amount)
	assert.Equal(t, expected.AddressType, actual.AddressType)
	assert.Equal(t, expected.Address, actual.Address)
}

func TestDeserializeUTXOOutputTooShort(t *testing.T) {
	buf := make([]byte, 11) // One byte too short for header
	_, _, err := DeserializeUTXOOutput(buf)
	require.ErrorContains(t, err, "UTXO output too short")
}

func TestDeserializeUTXOOutputUnknownAddressType(t *testing.T) {
	buf := make([]byte, 32)
	// Set amount (8 bytes)
	buf[7] = 0x01
	// Set unknown address type (4 bytes)
	buf[11] = 0x63 // Unknown type 99

	_, _, err := DeserializeUTXOOutput(buf)
	require.ErrorContains(t, err, "unknown UTXO address type")
}

func TestUTXOUnlockPayloadSerialize(t *testing.T) {
	payload := UTXOUnlockPayload{
		DestinationChain:         ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []UTXOInput{
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     0,
			},
		},
		Outputs: []UTXOOutput{
			{
				Amount:      900000,
				AddressType: UTXOAddressTypeP2PKH,
				Address:     testP2PKHAddress,
			},
			{
				Amount:      50000,
				AddressType: UTXOAddressTypeP2SH,
				Address:     testP2SHAddress,
			},
		},
	}

	buf, err := payload.Serialize()
	require.NoError(t, err)

	// Calculate expected length:
	// prefix (4) + dest_chain (2) + manager_set (4) + len_input (4) + input (68) + len_output (4) + output1 (32) + output2 (32)
	// = 4 + 2 + 4 + 4 + 68 + 4 + 32 + 32 = 150
	expectedLen := 4 + 2 + 4 + 4 + 68 + 4 + 32 + 32
	assert.Len(t, buf, expectedLen)

	// Verify prefix
	assert.Equal(t, UTXOPayloadPrefix[:], buf[0:4])
}

func TestDeserializeUTXOUnlockPayload(t *testing.T) {
	original := &UTXOUnlockPayload{
		DestinationChain:         ChainIDDogecoin,
		DelegatedManagerSetIndex: 5,
		Inputs: []UTXOInput{
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     0,
			},
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     1,
			},
		},
		Outputs: []UTXOOutput{
			{
				Amount:      900000,
				AddressType: UTXOAddressTypeP2PKH,
				Address:     testP2PKHAddress,
			},
		},
	}

	// Serialize
	buf, err := original.Serialize()
	require.NoError(t, err)

	// Deserialize
	actual, err := DeserializeUTXOUnlockPayload(buf)
	require.NoError(t, err)

	assert.Equal(t, original.DestinationChain, actual.DestinationChain)
	assert.Equal(t, original.DelegatedManagerSetIndex, actual.DelegatedManagerSetIndex)
	assert.Len(t, actual.Inputs, 2)
	assert.Len(t, actual.Outputs, 1)

	// Verify first input
	assert.Equal(t, original.Inputs[0].OriginalRecipientAddress, actual.Inputs[0].OriginalRecipientAddress)
	assert.Equal(t, original.Inputs[0].TransactionID, actual.Inputs[0].TransactionID)
	assert.Equal(t, original.Inputs[0].Vout, actual.Inputs[0].Vout)

	// Verify output
	assert.Equal(t, original.Outputs[0].Amount, actual.Outputs[0].Amount)
	assert.Equal(t, original.Outputs[0].AddressType, actual.Outputs[0].AddressType)
	assert.Equal(t, original.Outputs[0].Address, actual.Outputs[0].Address)
}

func TestDeserializeUTXOUnlockPayloadErrors(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		expectedErr string
	}{
		{
			name:        "too short",
			payload:     make([]byte, 17), // Less than minimum 18 bytes
			expectedErr: "UTXO unlock payload too short",
		},
		{
			name:        "invalid prefix",
			payload:     append([]byte("XXXX"), make([]byte, 14)...), // Invalid prefix
			expectedErr: "invalid UTXO payload prefix",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DeserializeUTXOUnlockPayload(tc.payload)
			require.ErrorContains(t, err, tc.expectedErr)
		})
	}
}

func TestDeserializeUTXOUnlockPayloadTrailingBytes(t *testing.T) {
	original := &UTXOUnlockPayload{
		DestinationChain:         ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs:                   []UTXOInput{},
		Outputs:                  []UTXOOutput{},
	}

	buf, err := original.Serialize()
	require.NoError(t, err)

	// Add trailing bytes
	buf = append(buf, 0x00, 0x01)

	_, err = DeserializeUTXOUnlockPayload(buf)
	require.ErrorContains(t, err, "trailing bytes")
}

func TestUTXOUnlockPayloadRoundTrip(t *testing.T) {
	original := &UTXOUnlockPayload{
		DestinationChain:         ChainIDDogecoin,
		DelegatedManagerSetIndex: 7,
		Inputs: []UTXOInput{
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     0,
			},
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     1,
			},
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     2,
			},
		},
		Outputs: []UTXOOutput{
			{
				Amount:      1000000,
				AddressType: UTXOAddressTypeP2PKH,
				Address:     testP2PKHAddress,
			},
			{
				Amount:      500000,
				AddressType: UTXOAddressTypeP2SH,
				Address:     testP2SHAddress,
			},
		},
	}

	// Serialize
	buf, err := original.Serialize()
	require.NoError(t, err)

	// Deserialize
	deserialized, err := DeserializeUTXOUnlockPayload(buf)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.DestinationChain, deserialized.DestinationChain)
	assert.Equal(t, original.DelegatedManagerSetIndex, deserialized.DelegatedManagerSetIndex)
	assert.Len(t, deserialized.Inputs, len(original.Inputs))
	assert.Len(t, deserialized.Outputs, len(original.Outputs))

	for i, input := range original.Inputs {
		assert.Equal(t, input.OriginalRecipientAddress, deserialized.Inputs[i].OriginalRecipientAddress)
		assert.Equal(t, input.TransactionID, deserialized.Inputs[i].TransactionID)
		assert.Equal(t, input.Vout, deserialized.Inputs[i].Vout)
	}

	for i, output := range original.Outputs {
		assert.Equal(t, output.Amount, deserialized.Outputs[i].Amount)
		assert.Equal(t, output.AddressType, deserialized.Outputs[i].AddressType)
		assert.Equal(t, output.Address, deserialized.Outputs[i].Address)
	}
}

// Test hex encoding for documentation/debugging purposes
func TestUTXOUnlockPayloadHexEncoding(t *testing.T) {
	payload := UTXOUnlockPayload{
		DestinationChain:         ChainIDDogecoin, // 65 = 0x0041
		DelegatedManagerSetIndex: 1,
		Inputs: []UTXOInput{
			{
				OriginalRecipientAddress: testRecipientAddress,
				TransactionID:            testTransactionID,
				Vout:                     0,
			},
		},
		Outputs: []UTXOOutput{
			{
				Amount:      1000000, // 0x000000000F4240
				AddressType: UTXOAddressTypeP2PKH,
				Address:     testP2PKHAddress,
			},
		},
	}

	buf, err := payload.Serialize()
	require.NoError(t, err)

	// Expected structure:
	// "UTX0" prefix (4 bytes): 55545830
	// destination_chain (2 bytes): 0041 (ChainIDDogecoin = 65)
	// delegated_manager_set (4 bytes): 00000001
	// len_input (4 bytes): 00000001
	// input[0]:
	//   original_recipient_address (32 bytes)
	//   transaction_id (32 bytes)
	//   vout (4 bytes): 00000000
	// len_output (4 bytes): 00000001
	// output[0]:
	//   amount (8 bytes): 00000000000f4240
	//   address_type (4 bytes): 00000000 (P2PKH)
	//   address (20 bytes)

	hexStr := hex.EncodeToString(buf)

	// Verify prefix
	assert.True(t, bytes.HasPrefix(buf, []byte("UTX0")))
	assert.Equal(t, "55545830", hexStr[0:8])

	// Verify destination chain (Dogecoin = 65 = 0x0041)
	assert.Equal(t, "0041", hexStr[8:12])

	// Verify manager set index (1)
	assert.Equal(t, "00000001", hexStr[12:20])

	// Verify input count (1)
	assert.Equal(t, "00000001", hexStr[20:28])
}

func TestUTXOUnlockPayloadEmptyInputsOutputs(t *testing.T) {
	payload := UTXOUnlockPayload{
		DestinationChain:         ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs:                   []UTXOInput{},
		Outputs:                  []UTXOOutput{},
	}

	buf, err := payload.Serialize()
	require.NoError(t, err)

	// Expected length: prefix (4) + dest_chain (2) + manager_set (4) + len_input (4) + len_output (4) = 18
	assert.Len(t, buf, 18)

	// Deserialize
	deserialized, err := DeserializeUTXOUnlockPayload(buf)
	require.NoError(t, err)
	assert.Empty(t, deserialized.Inputs)
	assert.Empty(t, deserialized.Outputs)
}
