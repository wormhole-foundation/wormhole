package evm

import (
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
