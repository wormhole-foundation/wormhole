package common

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getTestMP() MessagePublication {
	txHash := common.HexToHash("80d6a81b73b3cebba81fba2a330bcfaa25ce93e52e6bd65a02c7c13932a8c1a5")
	timeStamp := time.Now()
	nonce := uint32(1)
	sequence := uint64(1)
	consistencyLevel := uint8(1)
	emitterChain := vaa.ChainIDEthereum
	emitterAddress := vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	payload := []byte{97, 97, 97, 97, 97, 97}

	return MessagePublication{
		TxHash:           txHash,
		Timestamp:        timeStamp,
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: consistencyLevel,
		EmitterChain:     emitterChain,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
	}
}

// Base test to ensure the basic struct I/O
func TestMessagePublication(t *testing.T) {
	txHash := common.HexToHash("80d6a81b73b3cebba81fba2a330bcfaa25ce93e52e6bd65a02c7c13932a8c1a5")
	timeStamp := time.Now()
	nonce := uint32(1)
	sequence := uint64(1)
	consistencyLevel := uint8(1)
	emitterChain := vaa.ChainIDEthereum
	emitterAddress := vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	payload := []byte{97, 97, 97, 97, 97, 97}

	mp := MessagePublication{
		TxHash:           txHash,
		Timestamp:        timeStamp,
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: consistencyLevel,
		EmitterChain:     emitterChain,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
	}

	assert.Equal(t, txHash, mp.TxHash)
	assert.Equal(t, timeStamp, mp.Timestamp)
	assert.Equal(t, nonce, mp.Nonce)
	assert.Equal(t, sequence, mp.Sequence)
	assert.Equal(t, consistencyLevel, mp.ConsistencyLevel)
	assert.Equal(t, emitterChain, mp.EmitterChain)
	assert.Equal(t, emitterAddress, mp.EmitterAddress)
	assert.Equal(t, payload, mp.Payload)
}

// This is a known limitation of Go 1.x, it lacks over/underflow safety
// Ref:
//   - https://github.com/golang/go/issues/31500
//   - https://github.com/golang/go/issues/30209
//
// Must be mindful of any addition or subtraction operations
//
// Confirmed no add/sub on this attribute as of March 2022
func TestMessagePublication_ConsistencyLevelOverOverflow(t *testing.T) {
	mp := getTestMP()

	// Set to the top of the range
	mp.ConsistencyLevel = 255

	// Overflow the range
	mp.ConsistencyLevel = mp.ConsistencyLevel + 1

	// Confirm the overflow took place, and rolled over to zero
	assert.Equal(t, uint8(0), mp.ConsistencyLevel)
}

// This is a known limitation of Go 1.x, it lacks over/underflow safety
// Ref:
//   - https://github.com/golang/go/issues/31500
//   - https://github.com/golang/go/issues/30209
//
// Must be mindful of any addition or subtraction operations
//
// Confirmed no add/sub on this attribute as of March 2022
func TestMessagePublication_ConsistencyLevelUnderOverflow(t *testing.T) {
	mp := getTestMP()

	// Set to the bottom of the range
	mp.ConsistencyLevel = 0

	// Underflow the range
	mp.ConsistencyLevel = mp.ConsistencyLevel - 1

	// Confirm the overflow took place, and rolled under to 255
	assert.Equal(t, uint8(255), mp.ConsistencyLevel)
}
