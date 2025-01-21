package solana

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gagliardetto/solana-go"
)

func TestVerifyConstants(t *testing.T) {
	// If either of these ever change, message publication and reobservation will break.
	assert.Equal(t, SolanaAccountLen, solana.PublicKeyLength)
	assert.Equal(t, SolanaSignatureLen, len(solana.Signature{}))
}
