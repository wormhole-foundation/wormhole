package vaa

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// CoreModule is the identifier of the Core module (which is used for governance messages)
var CoreModule = []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x43, 0x6f, 0x72, 0x65}

// WasmdModule is the identifier of the Wormchain Wasmd module (which is used for governance messages)
var WasmdModule = [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x57, 0x61, 0x73, 0x6D, 0x64, 0x4D, 0x6F, 0x64, 0x75, 0x6C, 0x65}
var WasmdModuleStr = string(WasmdModule[:])

type GovernanceAction uint8

var (
	// Wormhole governance actions
	ActionContractUpgrade   GovernanceAction = 1
	ActionGuardianSetUpdate GovernanceAction = 2

	// Wormchain cosmwasm governance actions
	ActionStoreCode           GovernanceAction = 1
	ActionInstantiateContract GovernanceAction = 2
	ActionMigrateContract     GovernanceAction = 3

	// Wormhole tokenbridge governance actions
	ActionRegisterChain      GovernanceAction = 1
	ActionUpgradeTokenBridge GovernanceAction = 2
	ActionModifyBalance      GovernanceAction = 3
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
	BodyTokenBridgeModifyBalance struct {
		Module        string
		TargetChainID ChainID
		Sequence      uint64
		ChainId       ChainID
		TokenChain    ChainID
		TokenAddress  Address
		Kind          uint8
		Amount        *big.Int
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

func (r BodyTokenBridgeModifyBalance) Serialize() []byte {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.Sequence)
	MustWrite(payload, binary.BigEndian, r.ChainId)
	MustWrite(payload, binary.BigEndian, r.TokenChain)
	payload.Write(r.TokenAddress[:])
	payload.WriteByte(r.Kind)

	amount_bytes := r.Amount.Bytes()
	// zero pad big endian big-int
	for i := 0; i < 32-len(amount_bytes); i++ {
		payload.WriteByte(0)
	}
	payload.Write(amount_bytes)
	reason := make([]byte, 32)

	// truncate or pad "reason"
	for i := 0; i < len(reason); i += 1 {
		if i < len(r.Reason) {
			reason[i] = r.Reason[i]
		} else {
			reason[i] = ' '
		}
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

func serializeBridgeGovernanceVaa(module string, actionId GovernanceAction, chainId ChainID, payload []byte) []byte {
	if len(module) > 32 {
		panic("module longer than 32 byte")
	}

	buf := &bytes.Buffer{}

	// Write token bridge header
	for i := 0; i < (32 - len(module)); i++ {
		buf.WriteByte(0x00)
	}
	buf.Write([]byte(module))
	// Write action ID
	MustWrite(buf, binary.BigEndian, actionId)
	// Write target chain
	MustWrite(buf, binary.BigEndian, chainId)
	// Write emitter address of chain to be registered
	buf.Write(payload[:])

	return buf.Bytes()
}
