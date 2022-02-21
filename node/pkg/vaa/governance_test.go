package vaa

import "testing"
import "time"
import "github.com/stretchr/testify/assert"

// Testing the expected default behavior of a CreateGovernanceVAA
func TestCreateGovernanceVAA(t *testing.T) {
	var nonce uint32 = 1
	var sequence uint64 = 1
	var guardianSetIndex uint32 = 1
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	got_vaa := CreateGovernanceVAA(nonce, sequence, guardianSetIndex, payload)
	
	want_vaa := &VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}

	assert.Equal(t, got_vaa, want_vaa)
}
