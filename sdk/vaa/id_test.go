package vaa

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVAAIDFromString(t *testing.T) {
	t.Parallel()

	id, err := VAAIDFromString("1/0000000000000000000000000000000000000000000000000000000000000004/1")
	require.NoError(t, err)
	require.Equal(t, ChainIDSolana, id.EmitterChain)
	require.Equal(t, uint64(1), id.Sequence)
	require.Equal(t, "1/0000000000000000000000000000000000000000000000000000000000000004/1", id.String())
}

func TestNewVAAID(t *testing.T) {
	t.Parallel()

	id, err := NewVAAID(uint32(2), "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16", 7)
	require.NoError(t, err)
	require.Equal(t, ChainIDEthereum, id.EmitterChain)
	require.Equal(t, uint64(7), id.Sequence)
	require.Equal(t, "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/7", id.String())
}

func TestVAAIDFromVAAUsesMethod(t *testing.T) {
	t.Parallel()

	v := &VAA{
		EmitterChain:   ChainIDSolana,
		EmitterAddress: Address{31: 4},
		Sequence:       9,
	}

	require.Equal(t, v.ID(), VAAIDFromVAA(v))
	require.Equal(t, "1/0000000000000000000000000000000000000000000000000000000000000004/9", v.ID().String())
}

func TestVAAIDValidateRejectsZeroValues(t *testing.T) {
	t.Parallel()

	err := (VAAID{}).Validate()
	require.EqualError(t, err, "VAA ID emitter chain is unset")

	err = (VAAID{EmitterChain: ChainIDSolana}).Validate()
	require.EqualError(t, err, "VAA ID emitter address is zero")
}
