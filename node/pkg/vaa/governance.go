package vaa

import (
	"time"
)

var GovernanceEmitter = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
var GovernanceChain = ChainIDSolana

func CreateGovernanceVAA(nonce uint32, sequence uint64, guardianSetIndex uint32, payload []byte) *VAA {
	vaa := &VAA{
		Version:          SupportedVAAVersion,
		GuardianSetIndex: guardianSetIndex,
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: 32,
		EmitterChain:     GovernanceChain,
		EmitterAddress:   GovernanceEmitter,
		Payload:          payload,
	}

	return vaa
}
