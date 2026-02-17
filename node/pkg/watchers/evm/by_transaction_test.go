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

func TestIsLogValid_ValidLog(t *testing.T) {
	l := validLog()
	assert.True(t, isLogValid(l, testContract))
}

func TestIsLogValid_Removed(t *testing.T) {
	l := validLog()
	l.Removed = true
	assert.False(t, isLogValid(l, testContract))
}

func TestIsLogValid_WrongContract(t *testing.T) {
	l := validLog()
	l.Address = wrongContract
	assert.False(t, isLogValid(l, testContract))
}

func TestIsLogValid_WrongTopic(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{wrongTopic}
	assert.False(t, isLogValid(l, testContract))
}

func TestIsLogValid_EmptyTopics(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{}
	assert.False(t, isLogValid(l, testContract))
}

func TestIsLogValid_MultipleTopics_CorrectFirst(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{LogMessagePublishedTopic, wrongTopic}
	assert.True(t, isLogValid(l, testContract))
}

func TestIsLogValid_MultipleTopics_WrongFirst(t *testing.T) {
	l := validLog()
	l.Topics = []eth_common.Hash{wrongTopic, LogMessagePublishedTopic}
	assert.False(t, isLogValid(l, testContract))
}

func TestIsLogValid_AllInvalid(t *testing.T) {
	l := types.Log{
		Address: wrongContract,
		Topics:  []eth_common.Hash{wrongTopic},
		Removed: true,
	}
	assert.False(t, isLogValid(l, testContract))
}
