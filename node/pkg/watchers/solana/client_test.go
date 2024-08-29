package solana

import (
	"testing"

	"github.com/gagliardetto/solana-go"

	"github.com/stretchr/testify/assert"
)

func TestVerifyConstants(t *testing.T) {
	// If either of these ever change, message publication and reobservation may break.
	assert.Equal(t, SolanaAccountLen, solana.PublicKeyLength)
	assert.Equal(t, SolanaSignatureLen, len(solana.Signature{}))
}
