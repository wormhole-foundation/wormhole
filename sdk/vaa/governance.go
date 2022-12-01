package vaa

import (
	"encoding/binary"
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
	codeIdKeccak := sha3.NewLegacyKeccak256()
	binary.Write(codeIdKeccak, binary.BigEndian, codeId)
	codeIdKeccak.Sum(codeId_hash[:0])

	// hash(hash(BigEndian(CodeID)), Label)
	var codeIdLabel_hash [32]byte
	codeIdLabelKeccak := sha3.NewLegacyKeccak256()
	codeIdLabelKeccak.Write(codeId_hash[:])
	codeIdLabelKeccak.Write([]byte(label))
	codeIdLabelKeccak.Sum(codeIdLabel_hash[:0])

	// hash(hash(hash(BigEndian(CodeID)), Label), Msg)
	codeIdLabelMsgKeccak := sha3.NewLegacyKeccak256()
	codeIdLabelMsgKeccak.Write(codeIdLabel_hash[:])
	codeIdLabelMsgKeccak.Write(msg)
	codeIdLabelMsgKeccak.Sum(expected_hash[:0])

	return expected_hash
}
