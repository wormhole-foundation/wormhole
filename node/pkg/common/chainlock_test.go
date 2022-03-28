package common

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMessagePublication(t *testing.T) {
	txHash := common.HexToHash("80d6a81b73b3cebba81fba2a330bcfaa25ce93e52e6bd65a02c7c13932a8c1a5")
	timeStamp := time.Now()
	nonce := uint32(1)
	sequence := uint64(1)
	consistencyLevel := uint8(1)
	emitterChain := vaa.ChainIDEthereum
	emitterAddress := vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	payload := []byte{97, 97, 97, 97, 97, 97}

	messagePublication := MessagePublication{
		TxHash:           txHash,
		Timestamp:        timeStamp,
		Nonce:            nonce,
		Sequence:         sequence,
		ConsistencyLevel: consistencyLevel,
		EmitterChain:     emitterChain,
		EmitterAddress:   emitterAddress,
		Payload:          payload,
	}

	assert.Equal(t, txHash, messagePublication.TxHash)
	assert.Equal(t, timeStamp, messagePublication.Timestamp)
	assert.Equal(t, nonce, messagePublication.Nonce)
	assert.Equal(t, sequence, messagePublication.Sequence)
	assert.Equal(t, consistencyLevel, messagePublication.ConsistencyLevel)
	assert.Equal(t, emitterChain, messagePublication.EmitterChain)
	assert.Equal(t, emitterAddress, messagePublication.EmitterAddress)
	assert.Equal(t, payload, messagePublication.Payload)
}
