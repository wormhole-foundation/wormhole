package evm

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func TestCclParseConfigInvalidType(t *testing.T) {
	buf, err := hex.DecodeString("09c9002a00000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, 32, len(buf))
	data := *(*[32]byte)(buf)

	_, err = cclParseConfig(data)
	assert.ErrorContains(t, err, "unexpected data type: 9")
}

func TestCclParseConfigAdditionalBlocksSuccess(t *testing.T) {
	buf, err := hex.DecodeString("01c9002a00000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, 32, len(buf))
	data := *(*[32]byte)(buf)

	r, err := cclParseConfig(data)
	require.NoError(t, err)
	require.Equal(t, AdditionalBlocksType, r.Type())

	switch req := r.(type) {
	case *AdditionalBlocks:
		assert.Equal(t, uint8(201), req.consistencyLevel)
		assert.Equal(t, uint16(42), req.additionalBlocks)
	default:
		panic("unsupported query type")
	}
}

func TestCclParseConfigSuccessNothingSpecial(t *testing.T) {
	buf, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, 32, len(buf))
	data := *(*[32]byte)(buf)

	r, err := cclParseConfig(data)
	require.NoError(t, err)
	require.Equal(t, NothingSpecialType, r.Type())

	switch r.(type) {
	case *NothingSpecial:
	default:
		panic("unsupported query type")
	}
}

func TestCclParseAdditionalBlocksConfigWrongLength(t *testing.T) {
	// First verify our test works by reading valid data.
	err := testCclParseAdditionalBlocksConfig(t, "01c9002a00000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	// Too short (deleted the last byte).
	err = testCclParseAdditionalBlocksConfig(t, "01c9002a000000000000000000000000000000000000000000000000000000")
	assert.ErrorContains(t, err, "unexpected remaining unread bytes in buffer, should be 28, are 27")

	// Too long (added an extra byte).
	err = testCclParseAdditionalBlocksConfig(t, "01c9002a0000000000000000000000000000000000000000000000000000000000")
	assert.ErrorContains(t, err, "unexpected remaining unread bytes in buffer, should be 28, are 29")

	// Way too short (part of num blocks is missing).
	err = testCclParseAdditionalBlocksConfig(t, "01c900")
	assert.ErrorContains(t, err, "failed to read num blocks")

	// Really too short (consistency level is missing).
	err = testCclParseAdditionalBlocksConfig(t, "01")
	assert.ErrorContains(t, err, "failed to read consistency level")
}

func testCclParseAdditionalBlocksConfig(t *testing.T, str string) error {
	t.Helper()
	data, err := hex.DecodeString(str)
	require.NoError(t, err)
	reader := bytes.NewReader(data[:])

	// Skip the request type
	reqType := CCLRequestType(0)
	require.NoError(t, binary.Read(reader, binary.BigEndian, &reqType))
	require.Equal(t, AdditionalBlocksType, reqType)

	_, err = cclParseAdditionalBlocksConfig(reader)
	return err
}

// TestCclHandleMessageSetsEffectiveCL verifies that when cclHandleMessage processes
// an AdditionalBlocks request, it sets pe.effectiveCL (not pe.message.ConsistencyLevel).
func TestCclHandleMessageSetsEffectiveCL(t *testing.T) {
	logger := zap.NewNop()

	// Create a watcher with CCL enabled
	w := &Watcher{
		cclEnabled: true,
		cclLogger:  logger,
		cclCache:   make(CCLCache),
	}

	// Create a message with ConsistencyLevelCustom
	msg := &common.MessagePublication{
		ConsistencyLevel: vaa.ConsistencyLevelCustom,
	}

	// Create a pendingMessage with initial effectiveCL = 0
	pe := &pendingMessage{
		message:     msg,
		effectiveCL: 0,
	}

	// Mock the contract response by pre-populating the cache
	// This simulates the contract returning Finalized (200) with 42 additional blocks
	emitterAddr := ethCommon.HexToAddress("0x1234567890123456789012345678901234567890")
	buf, err := hex.DecodeString("01c8002a00000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, 32, len(buf))
	w.cclCache[emitterAddr] = CCLCacheEntry{
		data:     *(*[32]byte)(buf),
		readTime: time.Now(), // Set timestamp to ensure cache is valid
	}

	// Call cclHandleMessage - this is what we're testing
	ctx := context.Background()
	w.cclHandleMessage(ctx, pe, emitterAddr)

	// CRITICAL ASSERTION: pe.effectiveCL should be set to the contract's consistency level
	assert.Equal(t, uint8(200), pe.effectiveCL, "effectiveCL should be set from contract")

	// CRITICAL ASSERTION: pe.message.ConsistencyLevel should REMAIN unchanged
	assert.Equal(t, vaa.ConsistencyLevelCustom, pe.message.ConsistencyLevel, "message.ConsistencyLevel must stay Custom for VAA hash consistency")
}

// TestCclHandleMessageInvalidCLSetsEffectiveCL verifies that when the contract returns
// an INVALID consistency level for AdditionalBlocks, we set pe.effectiveCL (not pe.message.ConsistencyLevel).
func TestCclHandleMessageInvalidCLSetsEffectiveCL(t *testing.T) {
	logger := zap.NewNop()

	// Create a watcher with CCL enabled
	w := &Watcher{
		cclEnabled: true,
		cclLogger:  logger,
		cclCache:   make(CCLCache),
	}

	// Create a message with ConsistencyLevelCustom
	msg := &common.MessagePublication{
		ConsistencyLevel: vaa.ConsistencyLevelCustom,
	}

	// Create a pendingMessage with initial effectiveCL = 0
	pe := &pendingMessage{
		message:     msg,
		effectiveCL: 0,
	}

	// Mock the contract response with an INVALID consistency level (99)
	// Valid values are: Finalized (200), Safe (201), PublishImmediately (1)
	emitterAddr := ethCommon.HexToAddress("0x1234567890123456789012345678901234567890")
	buf, err := hex.DecodeString("0163002a00000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)
	require.Equal(t, 32, len(buf))
	w.cclCache[emitterAddr] = CCLCacheEntry{
		data:     *(*[32]byte)(buf),
		readTime: time.Now(), // Set timestamp to ensure cache is valid
	}

	// Call cclHandleMessage - this should handle the invalid CL gracefully
	ctx := context.Background()
	w.cclHandleMessage(ctx, pe, emitterAddr)

	// CRITICAL ASSERTION: pe.effectiveCL should be set to Finalized (fallback for invalid CL)
	assert.Equal(t, vaa.ConsistencyLevelFinalized, pe.effectiveCL, "effectiveCL should be set to Finalized for invalid CL")

	// CRITICAL ASSERTION: pe.message.ConsistencyLevel should REMAIN unchanged
	// This ensures all Guardians produce the same VAA hash even when contract returns invalid data
	assert.Equal(t, vaa.ConsistencyLevelCustom, pe.message.ConsistencyLevel, "message.ConsistencyLevel must stay Custom for VAA hash consistency")
}
