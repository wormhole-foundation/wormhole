package vaa

import "testing"
import "time"
import "github.com/stretchr/testify/assert"

// Testing the expected default behavior of a CreateGovernanceVAA
func TestCreateGovernanceVAA(t *testing.T){
	var nonce uint32 = 1
	var sequence uint64 = 1
	var guardianSetIndex uint32 = 1
	var payload = []byte{97, 97, 97, 97, 97, 97}

	vaa := CreateGovernanceVAA(nonce, sequence, guardianSetIndex, payload)

	assert.Equal(t, vaa.Version, uint8(1))
	assert.Equal(t, vaa.GuardianSetIndex, uint32(1))
	assert.Nil(t, vaa.Signatures)
	assert.Equal(t, vaa.Timestamp, time.Unix(0, 0))
	assert.Equal(t, vaa.Timestamp, time.Unix(0, 0))
	assert.Equal(t, vaa.Nonce, uint32(1))
	assert.Equal(t, vaa.Sequence, uint64(1))
	assert.Equal(t, vaa.ConsistencyLevel, uint8(32))
	assert.Equal(t, vaa.ConsistencyLevel, uint8(32))
	assert.Equal(t, vaa.EmitterChain, ChainIDSolana)
	assert.Equal(t, vaa.EmitterAddress, Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
	assert.Equal(t, string(vaa.Payload), "aaaaaa")
}              