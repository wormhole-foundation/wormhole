package vaa

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// CoreModule is the identifier of the Core module (which is used for governance messages)
var CoreModule = []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x43, 0x6f, 0x72, 0x65}

// WasmdModule is the identifier of the Wormchain Wasmd module (which is used for governance messages)
var WasmdModule = [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x57, 0x61, 0x73, 0x6D, 0x64, 0x4D, 0x6F, 0x64, 0x75, 0x6C, 0x65}
var WasmdModuleStr = string(WasmdModule[:])

// GatewayModule is the identifier of the Gateway module (which is used for general Gateway-related governance messages)
var GatewayModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x47, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65,
}
var GatewayModuleStr = string(GatewayModule[:])

// CircleIntegrationModule is the identifier of the Circle Integration module (which is used for governance messages).
// It is the hex representation of "CircleIntegration" left padded with zeroes.
var CircleIntegrationModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x43,
	0x69, 0x72, 0x63, 0x6c, 0x65, 0x49, 0x6e, 0x74, 0x65, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
}
var CircleIntegrationModuleStr = string(CircleIntegrationModule[:])

// IbcReceiverModule is the identifier of the Wormchain ibc_receiver contract module (which is used for governance messages)
// It is the hex representation of "IbcReceiver" left padded with zeroes.
var IbcReceiverModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x49, 0x62, 0x63, 0x52, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x72,
}
var IbcReceiverModuleStr = string(IbcReceiverModule[:])

// IbcTranslatorModule is the identifier of the Wormchain ibc_receiver contract module (which is used for governance messages)
// It is the hex representation of "IbcTranslator" left padded with zeroes.
var IbcTranslatorModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x62, 0x63, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x6c, 0x61, 0x74, 0x6f, 0x72,
}
var IbcTranslatorModuleStr = string(IbcTranslatorModule[:])

// WormholeRelayerModule is the identifier of the Wormhole Relayer module (which is used for governance messages).
// It is the hex representation of "WormholeRelayer" left padded with zeroes.
var WormholeRelayerModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x57, 0x6f, 0x72, 0x6d, 0x68, 0x6f, 0x6c, 0x65, 0x52, 0x65, 0x6c, 0x61, 0x79, 0x65, 0x72,
}
var WormholeRelayerModuleStr = string(WormholeRelayerModule[:])

type GovernanceAction uint8

var (
	// Wormhole core governance actions
	// See e.g. GovernanceStructs.sol for semantic meaning of these
	ActionContractUpgrade    GovernanceAction = 1
	ActionGuardianSetUpdate  GovernanceAction = 2
	ActionCoreSetMessageFee  GovernanceAction = 3
	ActionCoreTransferFees   GovernanceAction = 4
	ActionCoreRecoverChainId GovernanceAction = 5

	// Wormchain cosmwasm/middleware governance actions
	ActionStoreCode                      GovernanceAction = 1
	ActionInstantiateContract            GovernanceAction = 2
	ActionMigrateContract                GovernanceAction = 3
	ActionAddWasmInstantiateAllowlist    GovernanceAction = 4
	ActionDeleteWasmInstantiateAllowlist GovernanceAction = 5

	// Gateway governance actions
	ActionScheduleUpgrade               GovernanceAction = 1
	ActionCancelUpgrade                 GovernanceAction = 2
	ActionSetIbcComposabilityMwContract GovernanceAction = 3

	// Accountant goverance actions
	ActionModifyBalance GovernanceAction = 1

	// Wormhole tokenbridge governance actions
	ActionRegisterChain             GovernanceAction = 1
	ActionUpgradeTokenBridge        GovernanceAction = 2
	ActionTokenBridgeRecoverChainId GovernanceAction = 3

	// Circle Integration governance actions
	CircleIntegrationActionUpdateWormholeFinality        GovernanceAction = 1
	CircleIntegrationActionRegisterEmitterAndDomain      GovernanceAction = 2
	CircleIntegrationActionUpgradeContractImplementation GovernanceAction = 3

	// Ibc Receiver governance actions
	IbcReceiverActionUpdateChannelChain GovernanceAction = 1

	// Ibc Translator governance actions
	IbcTranslatorActionUpdateChannelChain GovernanceAction = 1

	// Wormhole relayer governance actions
	WormholeRelayerSetDefaultDeliveryProvider GovernanceAction = 3
)

type (
	// BodyContractUpgrade is a governance message to perform a contract upgrade of the core module
	BodyContractUpgrade struct {
		ChainID     ChainID
		NewContract Address
	}

	// BodyGuardianSetUpdate is a governance message to set a new guardian set
	BodyGuardianSetUpdate struct {
		Keys     []common.Address
		NewIndex uint32
	}

	// BodyTokenBridgeRegisterChain is a governance message to register a chain on the token bridge
	BodyTokenBridgeRegisterChain struct {
		Module         string
		ChainID        ChainID
		EmitterAddress Address
	}

	// BodyTokenBridgeUpgradeContract is a governance message to upgrade the token bridge.
	BodyTokenBridgeUpgradeContract struct {
		Module        string
		TargetChainID ChainID
		NewContract   Address
	}

	// BodyTokenBridgeModifyBalance is a governance message to modify accountant balances for the tokenbridge.
	BodyAccountantModifyBalance struct {
		Module        string
		TargetChainID ChainID
		Sequence      uint64
		ChainId       ChainID
		TokenChain    ChainID
		TokenAddress  Address
		Kind          uint8
		Amount        *uint256.Int
		Reason        string
	}

	// BodyWormchainStoreCode is a governance message to upload a new cosmwasm contract to wormchain
	BodyWormchainStoreCode struct {
		WasmHash [32]byte
	}

	// BodyWormchainInstantiateContract is a governance message to instantiate a cosmwasm contract on wormchain
	BodyWormchainInstantiateContract struct {
		InstantiationParamsHash [32]byte
	}

	// BodyWormchainInstantiateContract is a governance message to migrate a cosmwasm contract on wormchain
	BodyWormchainMigrateContract struct {
		MigrationParamsHash [32]byte
	}

	// BodyWormchainAllowlistInstantiateContract is a governance message to allowlist a specific contract address to instantiate a specific wasm code id.
	BodyWormchainWasmAllowlistInstantiate struct {
		ContractAddr [32]byte
		CodeId       uint64
	}

	// BodyGatewayScheduleUpgrade is a governance message to schedule an upgrade on Gateway
	BodyGatewayScheduleUpgrade struct {
		Name   string
		Height uint64
	}

	// BodyGatewayIbcComposabilityMwContract is a governance message to set a specific contract (i.e. IBC Translator) for the ibc composability middleware to use
	BodyGatewayIbcComposabilityMwContract struct {
		ContractAddr [32]byte
	}

	// BodyCircleIntegrationUpdateWormholeFinality is a governance message to update the wormhole finality for Circle Integration.
	BodyCircleIntegrationUpdateWormholeFinality struct {
		TargetChainID ChainID
		Finality      uint8
	}

	// BodyCircleIntegrationRegisterEmitterAndDomain is a governance message to register an emitter and domain for Circle Integration.
	BodyCircleIntegrationRegisterEmitterAndDomain struct {
		TargetChainID         ChainID
		ForeignEmitterChainId ChainID
		ForeignEmitterAddress [32]byte
		CircleDomain          uint32
	}

	// BodyCircleIntegrationUpgradeContractImplementation is a governance message to upgrade the contract implementation for Circle Integration.
	BodyCircleIntegrationUpgradeContractImplementation struct {
		TargetChainID            ChainID
		NewImplementationAddress [32]byte
	}

	// BodyIbcUpdateChannelChain is a governance message to update the ibc channel_id -> chain_id mapping in either of the ibc_receiver or ibc_translator contracts
	BodyIbcUpdateChannelChain struct {
		// The chain that this governance VAA should be redeemed on
		TargetChainId ChainID

		// This should follow the IBC channel identifier standard: https://github.com/cosmos/ibc/tree/main/spec/core/ics-024-host-requirements#paths-identifiers-separators
		// If the identifier string is shorter than 64 bytes, the correct number of 0x00 bytes should be prepended.
		ChannelId [64]byte
		ChainId   ChainID
	}

	// BodyWormholeRelayerSetDefaultDeliveryProvider is a governance message to set the default relay provider for the Wormhole Relayer.
	BodyWormholeRelayerSetDefaultDeliveryProvider struct {
		ChainID                           ChainID
		NewDefaultDeliveryProviderAddress Address
	}
)

func (b BodyContractUpgrade) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, ActionContractUpgrade)
	// ChainID
	MustWrite(buf, binary.BigEndian, uint16(b.ChainID))

	buf.Write(b.NewContract[:])

	return buf.Bytes()
}

func (b BodyGuardianSetUpdate) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, ActionGuardianSetUpdate)
	// ChainID - 0 for universal
	MustWrite(buf, binary.BigEndian, uint16(0))

	MustWrite(buf, binary.BigEndian, b.NewIndex)
	MustWrite(buf, binary.BigEndian, uint8(len(b.Keys)))
	for _, k := range b.Keys {
		buf.Write(k[:])
	}

	return buf.Bytes()
}

func (r BodyTokenBridgeRegisterChain) Serialize() []byte {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.ChainID)
	payload.Write(r.EmitterAddress[:])
	// target chain 0 = universal
	return serializeBridgeGovernanceVaa(r.Module, ActionRegisterChain, 0, payload.Bytes())
}

func (r BodyTokenBridgeUpgradeContract) Serialize() []byte {
	return serializeBridgeGovernanceVaa(r.Module, ActionUpgradeTokenBridge, r.TargetChainID, r.NewContract[:])
}

func (r BodyAccountantModifyBalance) Serialize() []byte {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.Sequence)
	MustWrite(payload, binary.BigEndian, r.ChainId)
	MustWrite(payload, binary.BigEndian, r.TokenChain)
	payload.Write(r.TokenAddress[:])
	payload.WriteByte(r.Kind)

	amount_bytes := r.Amount.Bytes32()
	payload.Write(amount_bytes[:])

	reason := make([]byte, 32)

	// truncate or pad "reason"
	count := copy(reason, r.Reason)
	for i := range reason[count:] {
		reason[i] = ' '
	}
	payload.Write(reason)

	return serializeBridgeGovernanceVaa(r.Module, ActionModifyBalance, r.TargetChainID, payload.Bytes())
}

func (r BodyWormchainStoreCode) Serialize() []byte {
	return serializeBridgeGovernanceVaa(WasmdModuleStr, ActionStoreCode, ChainIDWormchain, r.WasmHash[:])
}

func (r BodyWormchainInstantiateContract) Serialize() []byte {
	return serializeBridgeGovernanceVaa(WasmdModuleStr, ActionInstantiateContract, ChainIDWormchain, r.InstantiationParamsHash[:])
}

func (r BodyWormchainMigrateContract) Serialize() []byte {
	return serializeBridgeGovernanceVaa(WasmdModuleStr, ActionMigrateContract, ChainIDWormchain, r.MigrationParamsHash[:])
}

func (r BodyWormchainWasmAllowlistInstantiate) Serialize(action GovernanceAction) []byte {
	payload := &bytes.Buffer{}
	payload.Write(r.ContractAddr[:])
	MustWrite(payload, binary.BigEndian, r.CodeId)
	return serializeBridgeGovernanceVaa(WasmdModuleStr, action, ChainIDWormchain, payload.Bytes())
}

func (r *BodyWormchainWasmAllowlistInstantiate) Deserialize(bz []byte) {
	if len(bz) != 40 {
		panic("incorrect payload length")
	}

	var contractAddr [32]byte
	copy(contractAddr[:], bz[0:32])

	codeId := binary.BigEndian.Uint64(bz[32:40])

	r.ContractAddr = contractAddr
	r.CodeId = codeId
}

func (r BodyGatewayIbcComposabilityMwContract) Serialize() []byte {
	payload := &bytes.Buffer{}
	payload.Write(r.ContractAddr[:])
	return serializeBridgeGovernanceVaa(GatewayModuleStr, ActionSetIbcComposabilityMwContract, ChainIDWormchain, payload.Bytes())
}

func (r *BodyGatewayIbcComposabilityMwContract) Deserialize(bz []byte) {
	if len(bz) != 32 {
		panic("incorrect payload length")
	}

	var contractAddr [32]byte
	copy(contractAddr[:], bz[0:32])

	r.ContractAddr = contractAddr
}

func (r BodyGatewayScheduleUpgrade) Serialize() []byte {
	payload := &bytes.Buffer{}
	payload.Write([]byte(r.Name))
	MustWrite(payload, binary.BigEndian, r.Height)
	return serializeBridgeGovernanceVaa(GatewayModuleStr, ActionScheduleUpgrade, ChainIDWormchain, payload.Bytes())
}

func (r *BodyGatewayScheduleUpgrade) Deserialize(bz []byte) {
	r.Name = string(bz[0 : len(bz)-8])
	r.Height = binary.BigEndian.Uint64(bz[len(bz)-8:])
}

func (r BodyCircleIntegrationUpdateWormholeFinality) Serialize() []byte {
	return serializeBridgeGovernanceVaa(CircleIntegrationModuleStr, CircleIntegrationActionUpdateWormholeFinality, r.TargetChainID, []byte{r.Finality})
}

func (r BodyCircleIntegrationRegisterEmitterAndDomain) Serialize() []byte {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.ForeignEmitterChainId)
	payload.Write(r.ForeignEmitterAddress[:])
	MustWrite(payload, binary.BigEndian, r.CircleDomain)
	return serializeBridgeGovernanceVaa(CircleIntegrationModuleStr, CircleIntegrationActionRegisterEmitterAndDomain, r.TargetChainID, payload.Bytes())
}

func (r BodyCircleIntegrationUpgradeContractImplementation) Serialize() []byte {
	payload := &bytes.Buffer{}
	payload.Write(r.NewImplementationAddress[:])
	return serializeBridgeGovernanceVaa(CircleIntegrationModuleStr, CircleIntegrationActionUpgradeContractImplementation, r.TargetChainID, payload.Bytes())
}

func (r BodyIbcUpdateChannelChain) Serialize(module string) []byte {
	if module != IbcReceiverModuleStr && module != IbcTranslatorModuleStr {
		panic("module for BodyIbcUpdateChannelChain must be either IbcReceiver or IbcTranslator")
	}

	payload := &bytes.Buffer{}
	payload.Write(r.ChannelId[:])
	MustWrite(payload, binary.BigEndian, r.ChainId)
	return serializeBridgeGovernanceVaa(module, IbcReceiverActionUpdateChannelChain, r.TargetChainId, payload.Bytes())
}

func (r BodyWormholeRelayerSetDefaultDeliveryProvider) Serialize() []byte {
	payload := &bytes.Buffer{}
	payload.Write(r.NewDefaultDeliveryProviderAddress[:])
	return serializeBridgeGovernanceVaa(WormholeRelayerModuleStr, WormholeRelayerSetDefaultDeliveryProvider, r.ChainID, payload.Bytes())
}

func EmptyPayloadVaa(module string, actionId GovernanceAction, chainId ChainID) []byte {
	return serializeBridgeGovernanceVaa(module, actionId, chainId, []byte{})
}

func serializeBridgeGovernanceVaa(module string, actionId GovernanceAction, chainId ChainID, payload []byte) []byte {
	buf := LeftPadBytes(module, 32)
	// Write action ID
	MustWrite(buf, binary.BigEndian, actionId)
	// Write target chain
	MustWrite(buf, binary.BigEndian, chainId)
	// Write emitter address of chain to be registered
	buf.Write(payload[:])

	return buf.Bytes()
}

func LeftPadIbcChannelId(channelId string) [64]byte {
	channelIdBuf := LeftPadBytes(channelId, 64)
	var channelIdIdLeftPadded [64]byte
	copy(channelIdIdLeftPadded[:], channelIdBuf.Bytes())
	return channelIdIdLeftPadded
}

// Prepends 0x00 bytes to the payload buffer, up to a size of `length`
func LeftPadBytes(payload string, length int) *bytes.Buffer {
	if length < 0 {
		panic("cannot prepend bytes to a negative length buffer")
	}

	if len(payload) > length {
		panic(fmt.Sprintf("payload longer than %d bytes", length))
	}

	buf := &bytes.Buffer{}

	// Prepend correct number of 0x00 bytes to the payload slice
	for i := 0; i < (length - len(payload)); i++ {
		buf.WriteByte(0x00)
	}

	// add the payload slice
	buf.Write([]byte(payload))

	return buf
}
