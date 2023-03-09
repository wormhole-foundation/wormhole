package vaa

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
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
	serializedBodyContractUpgrade := bodyContractUpgrade.Serialize()
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyContractUpgrade))
}

func TestBodyGuardianSetUpdateSerialize(t *testing.T) {
	keys := []common.Address{
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
	}
	bodyGuardianSetUpdate := BodyGuardianSetUpdate{Keys: keys, NewIndex: uint32(1)}
	expected := "00000000000000000000000000000000000000000000000000000000436f726502000000000001025aaeb6053f3e94c9b9a09f33669435e7ef1beaed5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"
	serializedBodyGuardianSetUpdate := bodyGuardianSetUpdate.Serialize()
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyGuardianSetUpdate))
}

func TestBodyTokenBridgeRegisterChainSerialize(t *testing.T) {
	module := "test"
	tests := []struct {
		name     string
		expected string
		object   BodyTokenBridgeRegisterChain
		panic    bool
	}{
		{
			name:     "working_as_expected",
			panic:    false,
			object:   BodyTokenBridgeRegisterChain{Module: module, ChainID: 1, EmitterAddress: addr},
			expected: "000000000000000000000000000000000000000000000000000000007465737401000000010000000000000000000000000000000000000000000000000000000000000004",
		},
		{
			name:     "panic_at_the_disco!",
			panic:    true,
			object:   BodyTokenBridgeRegisterChain{Module: "123456789012345678901234567890123", ChainID: 1, EmitterAddress: addr},
			expected: "module longer than 32 byte",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.panic {
				assert.PanicsWithValue(t, testCase.expected, func() { testCase.object.Serialize() })
			} else {
				assert.Equal(t, testCase.expected, hex.EncodeToString(testCase.object.Serialize()))
			}
		})
	}
}

func TestBodyTokenBridgeUpgradeContractSerialize(t *testing.T) {
	module := "test"
	bodyTokenBridgeUpgradeContract := BodyTokenBridgeUpgradeContract{Module: module, TargetChainID: 1, NewContract: addr}
	expected := "00000000000000000000000000000000000000000000000000000000746573740200010000000000000000000000000000000000000000000000000000000000000004"
	serializedBodyTokenBridgeUpgradeContract := bodyTokenBridgeUpgradeContract.Serialize()
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyTokenBridgeUpgradeContract))
}

func TestBodyWormchainStoreCodeSerialize(t *testing.T) {
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65010c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	bodyWormchainStoreCode := BodyWormchainStoreCode{WasmHash: dummyBytes}
	assert.Equal(t, expected, hex.EncodeToString(bodyWormchainStoreCode.Serialize()))
}

func TestBodyWormchainInstantiateContractSerialize(t *testing.T) {
	actual := BodyWormchainInstantiateContract{InstantiationParamsHash: dummyBytes}
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65020c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	assert.Equal(t, expected, hex.EncodeToString(actual.Serialize()))
}

func TestBodyWormchainMigrateContractSerialize(t *testing.T) {
	actual := BodyWormchainMigrateContract{MigrationParamsHash: dummyBytes}
	expected := "0000000000000000000000000000000000000000005761736d644d6f64756c65030c200102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	assert.Equal(t, expected, hex.EncodeToString(actual.Serialize()))
}

func TestBodyCircleIntegrationUpdateWormholeFinalitySerialize(t *testing.T) {
	expected := "000000000000000000000000000000436972636c65496e746567726174696f6e0100022a"
	bodyCircleIntegrationUpdateWormholeFinality := BodyCircleIntegrationUpdateWormholeFinality{TargetChainID: ChainIDEthereum, Finality: 42}
	assert.Equal(t, expected, hex.EncodeToString(bodyCircleIntegrationUpdateWormholeFinality.Serialize()))
}

func TestBodyCircleIntegrationRegisterEmitterAndDomainSerialize(t *testing.T) {
	expected := "000000000000000000000000000000436972636c65496e746567726174696f6e020002000600000000000000000000000000000000000000000000000000000000000000040000002a"
	bodyCircleIntegrationRegisterEmitterAndDomain := BodyCircleIntegrationRegisterEmitterAndDomain{
		TargetChainID:         ChainIDEthereum,
		ForeignEmitterChainId: ChainIDAvalanche,
		ForeignEmitterAddress: addr,
		CircleDomain:          42,
	}
	assert.Equal(t, expected, hex.EncodeToString(bodyCircleIntegrationRegisterEmitterAndDomain.Serialize()))
}

func TestBodyCircleIntegrationUpgradeContractImplementationSerialize(t *testing.T) {
	expected := "000000000000000000000000000000436972636c65496e746567726174696f6e0300020000000000000000000000000000000000000000000000000000000000000004"
	bodyCircleIntegrationUpgradeContractImplementation := BodyCircleIntegrationUpgradeContractImplementation{
		TargetChainID:            ChainIDEthereum,
		NewImplementationAddress: addr,
	}
	assert.Equal(t, expected, hex.EncodeToString(bodyCircleIntegrationUpgradeContractImplementation.Serialize()))
	
func TestBodyWormholeRelayerSetDefaultRelayProviderSerialize(t *testing.T) {
	expected := "000000000000000000000000000000000000000000436f726552656c617965720300040000000000000000000000000000000000000000000000000000000000000004"
	bodyWormholeRelayerSetDefaultRelayProvider := BodyWormholeRelayerSetDefaultRelayProvider{
		ChainID: 4,
		NewDefaultRelayProviderAddress: addr,
	}
	assert.Equal(t, expected, hex.EncodeToString(bodyWormholeRelayerSetDefaultRelayProvider.Serialize()))
}
