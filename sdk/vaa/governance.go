package vaa

import (
	"encoding/binary"
	"fmt"
	"time"

	"golang.org/x/crypto/sha3"
)

var GovernanceEmitter = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
var GovernanceChain = ChainIDSolana

func CreateGovernanceVAA(timestamp time.Time, nonce uint32, sequence uint64, guardianSetIndex uint32, payload []byte) *VAA {
	vaa := &VAA{
		Version:          SupportedVAAVersion,
		GuardianSetIndex: guardianSetIndex,
		Signatures:       nil,
		Timestamp:        timestamp,
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: 32,
		EmitterChain:     GovernanceChain,
		EmitterAddress:   GovernanceEmitter,
		Payload:          payload,
	}

	return vaa
}

// Compute the hash for cosmwasm contract instatiation params.
// The hash is keccak256 hash(hash(hash(BigEndian(CodeID)), Label), Msg).
// We compute the nested hash so there is no chance of bytes leaking between CodeID, Label, and Msg.
func CreateInstatiateCosmwasmContractHash(codeId uint64, label string, msg []byte) [32]byte {
	var expected_hash [32]byte

	// hash(BigEndian(CodeID))
	var codeId_hash [32]byte
	keccak := sha3.NewLegacyKeccak256()
	if err := binary.Write(keccak, binary.BigEndian, codeId); err != nil {
		panic(fmt.Sprintf("failed to write binary data (%d): %v", codeId, err))
	}
	keccak.Sum(codeId_hash[:0])
	keccak.Reset()

	// hash(hash(BigEndian(CodeID)), Label)
	var codeIdLabel_hash [32]byte
	keccak.Write(codeId_hash[:])
	keccak.Write([]byte(label))
	keccak.Sum(codeIdLabel_hash[:0])
	keccak.Reset()

	// hash(hash(hash(BigEndian(CodeID)), Label), Msg)
	keccak.Write(codeIdLabel_hash[:])
	keccak.Write(msg)
	keccak.Sum(expected_hash[:0])

	return expected_hash
}

// Compute the hash for cosmwasm contract migration params.
// The hash is keccak256 hash(hash(hash(BigEndian(CodeID)), Contract), Msg).
// We compute the nested hash so there is no chance of bytes leaking between CodeID, Contract, and Msg.
func CreateMigrateCosmwasmContractHash(codeId uint64, contract string, msg []byte) [32]byte {
	var expected_hash [32]byte

	// hash(BigEndian(CodeID))
	var codeId_hash [32]byte
	keccak := sha3.NewLegacyKeccak256()
	if err := binary.Write(keccak, binary.BigEndian, codeId); err != nil {
		panic(fmt.Sprintf("failed to write binary data (%d): %v", codeId, err))
	}
	keccak.Sum(codeId_hash[:0])
	keccak.Reset()

	// hash(hash(BigEndian(CodeID)), Label)
	var codeIdContract_hash [32]byte
	keccak.Write(codeId_hash[:])
	keccak.Write([]byte(contract))
	keccak.Sum(codeIdContract_hash[:0])
	keccak.Reset()

	// hash(hash(hash(BigEndian(CodeID)), Label), Msg)
	keccak.Write(codeIdContract_hash[:])
	keccak.Write(msg)
	keccak.Sum(expected_hash[:0])

	return expected_hash
}
