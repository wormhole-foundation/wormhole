package evm

import (
	"testing"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

var (
	testContract  = eth_common.HexToAddress("0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B")
	wrongContract = eth_common.HexToAddress("0x0000000000000000000000000000000000000001")
	wrongTopic    = eth_common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
)

func validLog() types.Log {
	return types.Log{
		Address: testContract,
		Topics:  []eth_common.Hash{LogMessagePublishedTopic},
		Removed: false,
	}
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_ValidLog(t *testing.T) {
	l := validLog()
	assert.True(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_Removed(t *testing.T) {
	l := validLog()
	l.Removed = true
	assert.False(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_WrongContract(t *testing.T) {
	l := validLog()
	l.Address = wrongContract
	assert.False(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_WrongTopic(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{wrongTopic}
	assert.False(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_EmptyTopics(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{}
	assert.False(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_MultipleTopics_CorrectFirst(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{LogMessagePublishedTopic, wrongTopic}
	assert.True(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_MultipleTopics_WrongFirst(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{wrongTopic, LogMessagePublishedTopic}
	assert.False(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}

func testisValidCoreBridgeMessagePublicationLog_WrongTopic_AllInvalid(t *testing.T) {
	l := types.Log{
		Address: wrongContract,
		Topics:  []eth_common.Hash{wrongTopic},
		Removed: true,
	}
	assert.False(t, isValidCoreBridgeMessagePublicationLog(l, testContract))
}
