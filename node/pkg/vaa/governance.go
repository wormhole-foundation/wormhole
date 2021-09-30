package vaa

import (
	"time"
)

var governanceEmitter = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
var governanceChain = ChainIDSolana

func CreateGovernanceVAA(nonce uint32, sequence uint64, guardianSetIndex uint32, payload []byte) *VAA {
	vaa := &VAA{
		Version:          SupportedVAAVersion,
		GuardianSetIndex: guardianSetIndex,
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: 32,
		EmitterChain:     governanceChain,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}

	return vaa
}
