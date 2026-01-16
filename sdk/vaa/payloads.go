package vaa

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// CoreModule is the identifier of the Core module (which is used for governance messages)
var CoreModule = []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x43, 0x6f, 0x72, 0x65}
var CoreModuleStr = string(CoreModule[:])

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

var GeneralPurposeGovernanceModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x47, 0x65, 0x6E, 0x65, 0x72, 0x61, 0x6C,
	0x50, 0x75, 0x72, 0x70, 0x6F, 0x73, 0x65, 0x47, 0x6F, 0x76, 0x65, 0x72, 0x6E, 0x61, 0x6E,
	0x63, 0x65,
}
var GeneralPurposeGovernanceModuleStr = string(GeneralPurposeGovernanceModule[:])

// DelegatedManagerModule is the identifier of the Delegated Manager module (which is used for governance messages).
// It is the hex representation of "DelegatedManager" left padded with zeroes.
var DelegatedManagerModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x44, 0x65, 0x6C, 0x65, 0x67, 0x61, 0x74, 0x65, 0x64, 0x4D, 0x61, 0x6E, 0x61, 0x67, 0x65, 0x72,
}
var DelegatedManagerModuleStr = string(DelegatedManagerModule[:])

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
	ActionSlashingParamsUpdate          GovernanceAction = 4

	// Accountant governance actions
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

	// General purpose governance
	GeneralPurposeGovernanceEvmAction    GovernanceAction = 1
	GeneralPurposeGovernanceSolanaAction GovernanceAction = 2

	// Delegated manager governance actions
	ActionManagerSetUpdate GovernanceAction = 1
)

type (
	// BodyContractUpgrade is a governance message to perform a contract upgrade of the core module
	BodyContractUpgrade struct {
		ChainID     ChainID
		NewContract Address
	}

	// BodyGuardianSetUpdate is a governance message to set a new guardian set
	BodyGuardianSetUpdate struct {
		Keys     []ethcommon.Address
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

	// BodyRecoverChainId is a governance message to recover a chain id.
	BodyRecoverChainId struct {
		Module     string
		EvmChainID *uint256.Int
		NewChainID ChainID
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

	// BodyGatewaySlashingParamsUpdate is a governance message to update the slashing parameters on Wormchain.
	//
	// It is important to note that the slashing keeper only accepts `int64` values as input, so we need to convert
	// the `uint64` values to `int64` before passing them to the keeper. This conversion can introduce overflow
	// issues if the `uint64` values are too large. To combat this, the Wormchain CLI and the slashing keeper run
	// validation checks on the new parameter values.
	//
	// Below documents the entire process of updating the slashing parameters:
	// 1. The CLI command receives the new slashing parameters from the user as `uint64` values for `SignedBlocksWindow` and `DowntimeJailDuration` and as `string` values
	// for `MinSignedPerWindow`, `SlashFractionDoubleSign`, and `SlashFractionDowntime`. The command accepts `string` values for ease of use when providing decimal values.
	// 2. The CLI command converts the `string` values into `sdk.Dec` values and then into `uint64` values.
	// 3. The CLI command validates that the `uint64` values are within the acceptable range for the slashing parameters.
	// 4. The CLI command serializes the new slashing parameters into a governance VAA.
	// 5. The governance VAA is signed & broadcasted to the Wormchain.
	// 6. Wormchain deserializes the governance VAA and extracts every new slashing parameter as a uint64 value.
	// 7. Wormchain converts the uint64 values to int64 values and passes them to the slashing keeper.
	// 8. The slashing keeper runs validation checks on the new slashing parameters and throws an error if they are invalid.
	// 9. If the new slashing parameters pass the validation checks, the slashing keeper updates its parameters.
	BodyGatewaySlashingParamsUpdate struct {
		SignedBlocksWindow      uint64
		MinSignedPerWindow      uint64
		DowntimeJailDuration    uint64
		SlashFractionDoubleSign uint64
		SlashFractionDowntime   uint64
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

	// BodyGeneralPurposeGovernanceEvm is a general purpose governance message for EVM chains
	BodyGeneralPurposeGovernanceEvm struct {
		ChainID            ChainID
		GovernanceContract ethcommon.Address
		TargetContract     ethcommon.Address
		Payload            []byte
	}

	// BodyGeneralPurposeGovernanceSolana is a general purpose governance message for Solana chains
	BodyGeneralPurposeGovernanceSolana struct {
		ChainID            ChainID
		GovernanceContract Address
		// NOTE: unlike in EVM, no target contract in the schema here, the
		// instruction encodes the target contract address (unlike in EVM, where
		// an abi encoded calldata doesn't include the target contract address)
		Instruction []byte
	}

	// BodyCoreBridgeSetMessageFee is a governance message to set the message fee for the core bridge.
	BodyCoreBridgeSetMessageFee struct {
		ChainID    ChainID
		MessageFee *uint256.Int
	}

	// BodyManagerSetUpdate is a governance message to update the delegated manager set for a chain.
	// This is used to delegate manager signing authority to a set of public keys for bridging to chains
	// without smart contract support (e.g., Dogecoin, Bitcoin).
	BodyManagerSetUpdate struct {
		// ManagerChainID is the Wormhole Chain ID for the manager chain (e.g., Dogecoin)
		ManagerChainID ChainID
		// NewManagerSetIndex is the index of the new manager set
		NewManagerSetIndex uint32
		// NewManagerSet is the raw bytes of the new manager set (format is chain-specific)
		NewManagerSet []byte
	}
)

// ManagerSetType represents the type of manager set.
type ManagerSetType uint8

const (
	// ManagerSetTypeSecp256k1Multisig represents an equal-weight compressed secp256k1 public key multisig.
	ManagerSetTypeSecp256k1Multisig ManagerSetType = 1

	// CompressedSecp256k1PublicKeyLength is the length in bytes of a compressed secp256k1 public key.
	CompressedSecp256k1PublicKeyLength = 33
)

// Secp256k1MultisigManagerSet represents an equal-weight compressed secp256k1 public key multisig manager set.
// This is used for UTXO-based chains like Dogecoin and Bitcoin.
type Secp256k1MultisigManagerSet struct {
	// M is the number of valid signatures required (threshold)
	M uint8
	// N is the number of public keys in the set
	N uint8
	// PublicKeys is the list of compressed secp256k1 public keys
	PublicKeys [][CompressedSecp256k1PublicKeyLength]byte
}

//nolint:unparam // TODO: The error is always nil here. This function should not return an error.
func (b BodyContractUpgrade) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, ActionContractUpgrade)
	// ChainID
	MustWrite(buf, binary.BigEndian, uint16(b.ChainID))

	buf.Write(b.NewContract[:])

	return buf.Bytes(), nil
}

//nolint:unparam // TODO: The error is always nil here. This function should not return an error.
func (b BodyGuardianSetUpdate) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, ActionGuardianSetUpdate)
	// ChainID - 0 for universal
	MustWrite(buf, binary.BigEndian, uint16(0))

	MustWrite(buf, binary.BigEndian, b.NewIndex)
	MustWrite(buf, binary.BigEndian, uint8(len(b.Keys))) // #nosec G115 -- There will never be 256 guardians
	for _, k := range b.Keys {
		buf.Write(k[:])
	}

	return buf.Bytes(), nil
}

func (r BodyTokenBridgeRegisterChain) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.ChainID)
	payload.Write(r.EmitterAddress[:])
	// target chain 0 = universal
	return serializeBridgeGovernanceVaa(r.Module, ActionRegisterChain, 0, payload.Bytes())
}

func (r BodyTokenBridgeUpgradeContract) Serialize() ([]byte, error) {
	return serializeBridgeGovernanceVaa(r.Module, ActionUpgradeTokenBridge, r.TargetChainID, r.NewContract[:])
}

func (r BodyRecoverChainId) Serialize() ([]byte, error) {
	// Module
	buf, err := LeftPadBytes(r.Module, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to left pad module: %w", err)
	}
	// Action
	var action GovernanceAction
	if r.Module == "Core" {
		action = ActionCoreRecoverChainId
	} else {
		action = ActionTokenBridgeRecoverChainId
	}
	MustWrite(buf, binary.BigEndian, action)
	// EvmChainID
	MustWrite(buf, binary.BigEndian, r.EvmChainID.Bytes32())
	// NewChainID
	MustWrite(buf, binary.BigEndian, r.NewChainID)
	return buf.Bytes(), nil
}

const AccountantModifyBalanceReasonLength = 32

func (r BodyAccountantModifyBalance) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.Sequence)
	MustWrite(payload, binary.BigEndian, r.ChainId)
	MustWrite(payload, binary.BigEndian, r.TokenChain)
	payload.Write(r.TokenAddress[:])
	payload.WriteByte(r.Kind)

	amount_bytes := r.Amount.Bytes32()
	payload.Write(amount_bytes[:])

	reason := make([]byte, AccountantModifyBalanceReasonLength)

	// truncate or pad "reason"
	count := copy(reason, r.Reason)
	for i := range reason[count:] {
		reason[i] = ' '
	}
	payload.Write(reason)

	return serializeBridgeGovernanceVaa(r.Module, ActionModifyBalance, r.TargetChainID, payload.Bytes())
}

func (r BodyWormchainStoreCode) Serialize() ([]byte, error) {
	return serializeBridgeGovernanceVaa(WasmdModuleStr, ActionStoreCode, ChainIDWormchain, r.WasmHash[:])
}

func (r BodyWormchainInstantiateContract) Serialize() ([]byte, error) {
	return serializeBridgeGovernanceVaa(WasmdModuleStr, ActionInstantiateContract, ChainIDWormchain, r.InstantiationParamsHash[:])
}

func (r BodyWormchainMigrateContract) Serialize() ([]byte, error) {
	return serializeBridgeGovernanceVaa(WasmdModuleStr, ActionMigrateContract, ChainIDWormchain, r.MigrationParamsHash[:])
}

func (r BodyWormchainWasmAllowlistInstantiate) Serialize(action GovernanceAction) ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write(r.ContractAddr[:])
	MustWrite(payload, binary.BigEndian, r.CodeId)
	return serializeBridgeGovernanceVaa(WasmdModuleStr, action, ChainIDWormchain, payload.Bytes())
}

func (r *BodyWormchainWasmAllowlistInstantiate) Deserialize(bz []byte) error {
	if len(bz) != 40 {
		return fmt.Errorf("incorrect payload length, should be 40, is %d", len(bz))
	}

	var contractAddr [32]byte
	copy(contractAddr[:], bz[0:32])

	codeId := binary.BigEndian.Uint64(bz[32:40])

	r.ContractAddr = contractAddr
	r.CodeId = codeId
	return nil
}

func (r BodyGatewayIbcComposabilityMwContract) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write(r.ContractAddr[:])
	return serializeBridgeGovernanceVaa(GatewayModuleStr, ActionSetIbcComposabilityMwContract, ChainIDWormchain, payload.Bytes())
}

func (r *BodyGatewayIbcComposabilityMwContract) Deserialize(bz []byte) error {
	if len(bz) != 32 {
		return fmt.Errorf("incorrect payload length, should be 32, is %d", len(bz))
	}

	var contractAddr [32]byte
	copy(contractAddr[:], bz[0:32])

	r.ContractAddr = contractAddr
	return nil
}

func (b BodyGatewaySlashingParamsUpdate) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	MustWrite(payload, binary.BigEndian, b.SignedBlocksWindow)
	MustWrite(payload, binary.BigEndian, b.MinSignedPerWindow)
	MustWrite(payload, binary.BigEndian, b.DowntimeJailDuration)
	MustWrite(payload, binary.BigEndian, b.SlashFractionDoubleSign)
	MustWrite(payload, binary.BigEndian, b.SlashFractionDowntime)
	return serializeBridgeGovernanceVaa(GatewayModuleStr, ActionSlashingParamsUpdate, ChainIDWormchain, payload.Bytes())
}

func (r *BodyGatewaySlashingParamsUpdate) Deserialize(bz []byte) error {
	if len(bz) != 40 {
		return fmt.Errorf("incorrect payload length, should be 40, is %d", len(bz))
	}

	r.SignedBlocksWindow = binary.BigEndian.Uint64(bz[0:8])
	r.MinSignedPerWindow = binary.BigEndian.Uint64(bz[8:16])
	r.DowntimeJailDuration = binary.BigEndian.Uint64(bz[16:24])
	r.SlashFractionDoubleSign = binary.BigEndian.Uint64(bz[24:32])
	r.SlashFractionDowntime = binary.BigEndian.Uint64(bz[32:40])
	return nil
}

func (r BodyGatewayScheduleUpgrade) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write([]byte(r.Name))
	MustWrite(payload, binary.BigEndian, r.Height)
	return serializeBridgeGovernanceVaa(GatewayModuleStr, ActionScheduleUpgrade, ChainIDWormchain, payload.Bytes())
}

//nolint:unparam // TODO: The error is always nil here. This function should not return an error.
func (r *BodyGatewayScheduleUpgrade) Deserialize(bz []byte) error {
	r.Name = string(bz[0 : len(bz)-8])
	r.Height = binary.BigEndian.Uint64(bz[len(bz)-8:])
	return nil
}

func (r BodyCircleIntegrationUpdateWormholeFinality) Serialize() ([]byte, error) {
	return serializeBridgeGovernanceVaa(CircleIntegrationModuleStr, CircleIntegrationActionUpdateWormholeFinality, r.TargetChainID, []byte{r.Finality})
}

func (r BodyCircleIntegrationRegisterEmitterAndDomain) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.ForeignEmitterChainId)
	payload.Write(r.ForeignEmitterAddress[:])
	MustWrite(payload, binary.BigEndian, r.CircleDomain)
	return serializeBridgeGovernanceVaa(CircleIntegrationModuleStr, CircleIntegrationActionRegisterEmitterAndDomain, r.TargetChainID, payload.Bytes())
}

func (r BodyCircleIntegrationUpgradeContractImplementation) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write(r.NewImplementationAddress[:])
	return serializeBridgeGovernanceVaa(CircleIntegrationModuleStr, CircleIntegrationActionUpgradeContractImplementation, r.TargetChainID, payload.Bytes())
}

func (r BodyIbcUpdateChannelChain) Serialize(module string) ([]byte, error) {
	if module != IbcReceiverModuleStr && module != IbcTranslatorModuleStr {
		return nil, errors.New("module for BodyIbcUpdateChannelChain must be either IbcReceiver or IbcTranslator")
	}

	payload := &bytes.Buffer{}
	payload.Write(r.ChannelId[:])
	MustWrite(payload, binary.BigEndian, r.ChainId)
	return serializeBridgeGovernanceVaa(module, IbcReceiverActionUpdateChannelChain, r.TargetChainId, payload.Bytes())
}

func (r BodyWormholeRelayerSetDefaultDeliveryProvider) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write(r.NewDefaultDeliveryProviderAddress[:])
	return serializeBridgeGovernanceVaa(WormholeRelayerModuleStr, WormholeRelayerSetDefaultDeliveryProvider, r.ChainID, payload.Bytes())
}

func (r BodyCoreBridgeSetMessageFee) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	feeBytes := r.MessageFee.Bytes()
	if len(feeBytes) > 32 {
		return nil, fmt.Errorf("message fee too large")
	}
	padded := make([]byte, 32)
	copy(padded[32-len(feeBytes):], feeBytes)
	payload.Write(padded)
	return serializeBridgeGovernanceVaa(CoreModuleStr, ActionCoreSetMessageFee, r.ChainID, payload.Bytes())
}

func (r BodyGeneralPurposeGovernanceEvm) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write(r.GovernanceContract[:])
	payload.Write(r.TargetContract[:])

	// write payload len as uint16
	if len(r.Payload) > math.MaxUint16 {
		return nil, fmt.Errorf("payload too long; expected at most %d bytes", math.MaxUint16)
	}
	MustWrite(payload, binary.BigEndian, uint16(len(r.Payload))) // #nosec G115 -- This is checked above
	payload.Write(r.Payload)
	return serializeBridgeGovernanceVaa(GeneralPurposeGovernanceModuleStr, GeneralPurposeGovernanceEvmAction, r.ChainID, payload.Bytes())
}

func (r BodyGeneralPurposeGovernanceSolana) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	payload.Write(r.GovernanceContract[:])
	// NOTE: unlike in EVM, we don't write the payload length here, because we're using
	// a custom instruction encoding (there is no standard encoding like evm ABI
	// encoding), generated by an external tool. That tool length-prefixes all
	// the relevant dynamic fields.
	payload.Write(r.Instruction)
	return serializeBridgeGovernanceVaa(GeneralPurposeGovernanceModuleStr, GeneralPurposeGovernanceSolanaAction, r.ChainID, payload.Bytes())
}

func (r BodyManagerSetUpdate) Serialize() ([]byte, error) {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.ManagerChainID)
	MustWrite(payload, binary.BigEndian, r.NewManagerSetIndex)
	payload.Write(r.NewManagerSet)
	// ChainID - 0 for universal
	return serializeBridgeGovernanceVaa(DelegatedManagerModuleStr, ActionManagerSetUpdate, 0, payload.Bytes())
}

func (r *BodyManagerSetUpdate) Deserialize(bz []byte) error {
	// Minimum length: 2 (ManagerChainID) + 4 (NewManagerSetIndex) = 6 bytes
	if len(bz) < 6 {
		return fmt.Errorf("incorrect payload length, should be at least 6 bytes, is %d", len(bz))
	}

	r.ManagerChainID = ChainID(binary.BigEndian.Uint16(bz[0:2]))
	r.NewManagerSetIndex = binary.BigEndian.Uint32(bz[2:6])
	r.NewManagerSet = bz[6:]
	return nil
}

// Serialize serializes the Secp256k1MultisigManagerSet into bytes.
// Format: Type (1 byte) + M (1 byte) + N (1 byte) + PublicKeys (N * CompressedSecp256k1PublicKeyLength bytes)
func (s Secp256k1MultisigManagerSet) Serialize() ([]byte, error) {
	numKeys := len(s.PublicKeys)
	if numKeys > 255 {
		return nil, fmt.Errorf("too many public keys: %d (max 255)", numKeys)
	}
	if int(s.N) != numKeys {
		return nil, fmt.Errorf("n (%d) does not match number of public keys (%d)", s.N, numKeys)
	}
	if s.M > s.N {
		return nil, fmt.Errorf("m (%d) cannot be greater than n (%d)", s.M, s.N)
	}
	if s.M == 0 {
		return nil, errors.New("m must be at least 1")
	}

	buf := new(bytes.Buffer)
	// Type
	buf.WriteByte(byte(ManagerSetTypeSecp256k1Multisig))
	// M
	buf.WriteByte(s.M)
	// N
	buf.WriteByte(s.N)
	// PublicKeys
	for _, pk := range s.PublicKeys {
		buf.Write(pk[:])
	}

	return buf.Bytes(), nil
}

// Deserialize deserializes bytes into a Secp256k1MultisigManagerSet.
func (s *Secp256k1MultisigManagerSet) Deserialize(bz []byte) error {
	// Minimum length: 1 (Type) + 1 (M) + 1 (N) = 3 bytes
	if len(bz) < 3 {
		return fmt.Errorf("payload too short, expected at least 3 bytes, got %d", len(bz))
	}

	managerSetType := ManagerSetType(bz[0])
	if managerSetType != ManagerSetTypeSecp256k1Multisig {
		return fmt.Errorf("unexpected manager set type %d, expected %d", managerSetType, ManagerSetTypeSecp256k1Multisig)
	}

	m := bz[1]
	n := bz[2]

	expectedLen := 3 + int(n)*CompressedSecp256k1PublicKeyLength
	if len(bz) != expectedLen {
		return fmt.Errorf("payload length mismatch, expected %d bytes for %d keys, got %d", expectedLen, n, len(bz))
	}

	if m > n {
		return fmt.Errorf("m (%d) cannot be greater than n (%d)", m, n)
	}
	if m == 0 {
		return errors.New("m must be at least 1")
	}

	publicKeys := make([][CompressedSecp256k1PublicKeyLength]byte, n)
	for i := uint8(0); i < n; i++ {
		offset := 3 + int(i)*CompressedSecp256k1PublicKeyLength
		copy(publicKeys[i][:], bz[offset:offset+CompressedSecp256k1PublicKeyLength])
	}

	s.M = m
	s.N = n
	s.PublicKeys = publicKeys
	return nil
}

func EmptyPayloadVaa(module string, actionId GovernanceAction, chainId ChainID) ([]byte, error) {
	return serializeBridgeGovernanceVaa(module, actionId, chainId, []byte{})
}

func serializeBridgeGovernanceVaa(module string, actionId GovernanceAction, chainId ChainID, payload []byte) ([]byte, error) {
	buf, err := LeftPadBytes(module, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to left pad module: %w", err)
	}
	// Write action ID
	MustWrite(buf, binary.BigEndian, actionId)
	// Write target chain
	MustWrite(buf, binary.BigEndian, chainId)
	// Write emitter address of chain to be registered
	buf.Write(payload[:])

	return buf.Bytes(), nil
}

func LeftPadIbcChannelId(channelId string) ([64]byte, error) {
	channelIdBuf, err := LeftPadBytes(channelId, 64)
	if err != nil {
		return [64]byte{}, fmt.Errorf("failed to left pad module: %w", err)
	}
	var channelIdIdLeftPadded [64]byte
	copy(channelIdIdLeftPadded[:], channelIdBuf.Bytes())
	return channelIdIdLeftPadded, nil
}

// Prepends 0x00 bytes to the payload buffer, up to a size of `length`
func LeftPadBytes(payload string, length int) (*bytes.Buffer, error) {
	if length < 0 {
		return nil, errors.New("cannot prepend bytes to a negative length buffer")
	}

	if len(payload) > length {
		return nil, fmt.Errorf("payload longer than %d bytes", length)
	}

	buf := &bytes.Buffer{}

	// Prepend correct number of 0x00 bytes to the payload slice
	for i := 0; i < (length - len(payload)); i++ {
		buf.WriteByte(0x00)
	}

	// add the payload slice
	buf.Write([]byte(payload))

	return buf, nil
}

// UTXOPayloadPrefix is the 4-byte prefix for UTXO unlock payloads
var UTXOPayloadPrefix = [4]byte{'U', 'T', 'X', '0'}

// UTXOAddressType represents the type of address in a UTXO output
type UTXOAddressType uint32

const (
	// UTXOAddressTypeP2PKH represents a Pay-to-Public-Key-Hash address (20 bytes)
	UTXOAddressTypeP2PKH UTXOAddressType = 0
	// UTXOAddressTypeP2SH represents a Pay-to-Script-Hash address (20 bytes)
	UTXOAddressTypeP2SH UTXOAddressType = 1
)

// AddressLength returns the length of the address for the given address type
func (t UTXOAddressType) AddressLength() (int, error) {
	switch t {
	case UTXOAddressTypeP2PKH, UTXOAddressTypeP2SH:
		return 20, nil
	default:
		return 0, fmt.Errorf("unknown UTXO address type: %d", t)
	}
}

// UTXOInput represents an input in a UTXO unlock payload.
// This corresponds to a UTXO that will be spent.
type UTXOInput struct {
	// OriginalRecipientAddress is the recipient address in the redeem script (32 bytes)
	OriginalRecipientAddress [32]byte
	// TransactionID is the deposit transaction ID (32 bytes, big-endian)
	TransactionID [32]byte
	// Vout is the vout index of the deposit transaction
	Vout uint32
}

// UTXOOutput represents an output in a UTXO unlock payload.
// This corresponds to a destination for unlocked funds.
type UTXOOutput struct {
	// Amount is the amount in the smallest unit (e.g., satoshis)
	Amount uint64
	// AddressType indicates the type of address (P2PKH, P2SH, etc.)
	AddressType UTXOAddressType
	// Address is the destination address (length depends on AddressType)
	Address []byte
}

// UTXOUnlockPayload represents a VAA payload for unlocking funds on a UTXO chain.
// This payload is emitted by a registered emitter to trigger Manager Service signing.
// All uint values are encoded big-endian.
type UTXOUnlockPayload struct {
	// DestinationChain is the Wormhole Chain ID for the destination chain
	DestinationChain ChainID
	// DelegatedManagerSetIndex is the index of the delegated manager set to use
	DelegatedManagerSetIndex uint32
	// Inputs is the list of UTXOs to spend
	Inputs []UTXOInput
	// Outputs is the list of destinations for the unlocked funds
	Outputs []UTXOOutput
}

// DeserializeUTXOInput deserializes a single UTXOInput from bytes.
// Expected format: OriginalRecipientAddress (32 bytes) + TransactionID (32 bytes) + Vout (4 bytes)
func DeserializeUTXOInput(bz []byte) (*UTXOInput, error) {
	const inputSize = 32 + 32 + 4 // OriginalRecipientAddress + TransactionID + Vout
	if len(bz) < inputSize {
		return nil, fmt.Errorf("UTXO input too short: expected at least %d bytes, got %d", inputSize, len(bz))
	}

	input := &UTXOInput{}
	copy(input.OriginalRecipientAddress[:], bz[0:32])
	copy(input.TransactionID[:], bz[32:64])
	input.Vout = binary.BigEndian.Uint32(bz[64:68])

	return input, nil
}

// Serialize serializes a UTXOInput to bytes.
func (i *UTXOInput) Serialize() []byte {
	buf := make([]byte, 68)
	copy(buf[0:32], i.OriginalRecipientAddress[:])
	copy(buf[32:64], i.TransactionID[:])
	binary.BigEndian.PutUint32(buf[64:68], i.Vout)
	return buf
}

// DeserializeUTXOOutput deserializes a single UTXOOutput from bytes.
// Expected format: Amount (8 bytes) + AddressType (4 bytes) + Address (variable length based on AddressType)
func DeserializeUTXOOutput(bz []byte) (*UTXOOutput, int, error) {
	// Minimum size: Amount (8) + AddressType (4)
	if len(bz) < 12 {
		return nil, 0, fmt.Errorf("UTXO output too short: expected at least 12 bytes, got %d", len(bz))
	}

	output := &UTXOOutput{}
	output.Amount = binary.BigEndian.Uint64(bz[0:8])
	output.AddressType = UTXOAddressType(binary.BigEndian.Uint32(bz[8:12]))

	addrLen, err := output.AddressType.AddressLength()
	if err != nil {
		return nil, 0, err
	}

	totalSize := 12 + addrLen
	if len(bz) < totalSize {
		return nil, 0, fmt.Errorf("UTXO output too short for address: expected %d bytes, got %d", totalSize, len(bz))
	}

	output.Address = make([]byte, addrLen)
	copy(output.Address, bz[12:12+addrLen])

	return output, totalSize, nil
}

// Serialize serializes a UTXOOutput to bytes.
func (o *UTXOOutput) Serialize() ([]byte, error) {
	addrLen, err := o.AddressType.AddressLength()
	if err != nil {
		return nil, err
	}
	if len(o.Address) != addrLen {
		return nil, fmt.Errorf("address length mismatch: expected %d bytes for address type %d, got %d", addrLen, o.AddressType, len(o.Address))
	}

	buf := make([]byte, 12+addrLen)
	binary.BigEndian.PutUint64(buf[0:8], o.Amount)
	binary.BigEndian.PutUint32(buf[8:12], uint32(o.AddressType))
	copy(buf[12:], o.Address)
	return buf, nil
}

// DeserializeUTXOUnlockPayload deserializes a UTXOUnlockPayload from bytes.
// This is the payload format for triggering Manager Service signing on UTXO chains.
func DeserializeUTXOUnlockPayload(bz []byte) (*UTXOUnlockPayload, error) {
	// Minimum size: prefix (4) + destination_chain (2) + manager_set (4) + len_input (4) + len_output (4)
	const minSize = 4 + 2 + 4 + 4 + 4
	if len(bz) < minSize {
		return nil, fmt.Errorf("UTXO unlock payload too short: expected at least %d bytes, got %d", minSize, len(bz))
	}

	// Check prefix
	if !bytes.Equal(bz[0:4], UTXOPayloadPrefix[:]) {
		return nil, fmt.Errorf("invalid UTXO payload prefix: expected %s, got %s", string(UTXOPayloadPrefix[:]), string(bz[0:4]))
	}

	payload := &UTXOUnlockPayload{}
	offset := 4

	// Destination chain
	payload.DestinationChain = ChainID(binary.BigEndian.Uint16(bz[offset : offset+2]))
	offset += 2

	// Delegated manager set index
	payload.DelegatedManagerSetIndex = binary.BigEndian.Uint32(bz[offset : offset+4])
	offset += 4

	// Number of inputs
	lenInput := binary.BigEndian.Uint32(bz[offset : offset+4])
	offset += 4

	// Deserialize inputs
	const inputSize = 68 // 32 + 32 + 4
	payload.Inputs = make([]UTXOInput, lenInput)
	for i := uint32(0); i < lenInput; i++ {
		if offset+inputSize > len(bz) {
			return nil, fmt.Errorf("UTXO unlock payload truncated while reading input %d", i)
		}
		input, err := DeserializeUTXOInput(bz[offset:])
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize input %d: %w", i, err)
		}
		payload.Inputs[i] = *input
		offset += inputSize
	}

	// Number of outputs
	if offset+4 > len(bz) {
		return nil, errors.New("UTXO unlock payload truncated while reading output count")
	}
	lenOutput := binary.BigEndian.Uint32(bz[offset : offset+4])
	offset += 4

	// Deserialize outputs
	payload.Outputs = make([]UTXOOutput, lenOutput)
	for i := uint32(0); i < lenOutput; i++ {
		output, size, err := DeserializeUTXOOutput(bz[offset:])
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize output %d: %w", i, err)
		}
		payload.Outputs[i] = *output
		offset += size
	}

	// Verify we consumed all bytes
	if offset != len(bz) {
		return nil, fmt.Errorf("UTXO unlock payload has trailing bytes: consumed %d of %d bytes", offset, len(bz))
	}

	return payload, nil
}

// Serialize serializes a UTXOUnlockPayload to bytes.
func (p *UTXOUnlockPayload) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Prefix
	buf.Write(UTXOPayloadPrefix[:])

	// Destination chain
	MustWrite(buf, binary.BigEndian, uint16(p.DestinationChain))

	// Delegated manager set index
	MustWrite(buf, binary.BigEndian, p.DelegatedManagerSetIndex)

	// Number of inputs
	if len(p.Inputs) > math.MaxUint32 {
		return nil, fmt.Errorf("too many inputs: %d", len(p.Inputs))
	}
	MustWrite(buf, binary.BigEndian, uint32(len(p.Inputs))) // #nosec G115 -- checked above

	// Inputs
	for i, input := range p.Inputs {
		inputBytes := input.Serialize()
		if _, err := buf.Write(inputBytes); err != nil {
			return nil, fmt.Errorf("failed to write input %d: %w", i, err)
		}
	}

	// Number of outputs
	if len(p.Outputs) > math.MaxUint32 {
		return nil, fmt.Errorf("too many outputs: %d", len(p.Outputs))
	}
	MustWrite(buf, binary.BigEndian, uint32(len(p.Outputs))) // #nosec G115 -- checked above

	// Outputs
	for i, output := range p.Outputs {
		outputBytes, err := output.Serialize()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize output %d: %w", i, err)
		}
		if _, err := buf.Write(outputBytes); err != nil {
			return nil, fmt.Errorf("failed to write output %d: %w", i, err)
		}
	}

	return buf.Bytes(), nil
}
