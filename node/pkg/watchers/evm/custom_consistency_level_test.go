package evm

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
