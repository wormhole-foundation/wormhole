package vaa

import (
	"bytes"
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
)

// CoreModule is the identifier of the Core module (which is used for governance messages)
var CoreModule = []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x43, 0x6f, 0x72, 0x65}

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

	// BodyRegisterChain is a governance message to register a chain on the token bridge
	BodyRegisterChain struct {
		Header         [32]byte
		ChainID        ChainID
		EmitterAddress Address
	}
)

func (b BodyContractUpgrade) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, uint8(1))
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
	MustWrite(buf, binary.BigEndian, uint8(2))
	// ChainID - 0 for universal
	MustWrite(buf, binary.BigEndian, uint16(0))

	MustWrite(buf, binary.BigEndian, b.NewIndex)
	MustWrite(buf, binary.BigEndian, uint8(len(b.Keys)))
	for _, k := range b.Keys {
		buf.Write(k[:])
	}

	return buf.Bytes()
}

func (r BodyRegisterChain) Serialize() []byte {
	buf := &bytes.Buffer{}

	// Write token bridge header
	buf.Write(r.Header[:])
	// Write action ID
	MustWrite(buf, binary.BigEndian, uint8(1))
	// Write target chain (0 = universal)
	MustWrite(buf, binary.BigEndian, uint16(0))
	// Write chain to be registered
	MustWrite(buf, binary.BigEndian, r.ChainID)
	// Write emitter address of chain to be registered
	buf.Write(r.EmitterAddress[:])

	return buf.Bytes()
}
