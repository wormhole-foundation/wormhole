package nearapi

import (
	"os"
	"testing"

	"github.com/test-go/testify/assert"
)

func TestNewBlockFromBytes(t *testing.T) {
	blockBytes, err := os.ReadFile("dummy/block.json")
	assert.NoError(t, err)

	block, err := newBlockFromBytes(blockBytes)
	assert.NoError(t, err)

	assert.Equal(t, block.Header.Hash, "NSM5RDZDF7uxGWiUwhBqJcqCEw6g7axx4TxGYB7XZVt")
	assert.Equal(t, block.Header.Height, uint64(75398642))
	assert.Equal(t, block.Header.LastFinalBlock, "ARo7pHDH5hk1qpfdwRYtcuWh5dEjTHXwNw8wCTJb78jf")
	assert.Equal(t, block.Header.PrevBlockHash, "FqPKohapMjpemtYh8nuQAB7iVJ3rDWtAZQnRjsXVFVbB")
	assert.Equal(t, block.Header.Timestamp, uint64(1664754166))
}
