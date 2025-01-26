package solana

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func TestVerifyConstants(t *testing.T) {
	// If either of these ever change, message publication and reobservation will break.
	assert.Equal(t, SolanaAccountLen, solana.PublicKeyLength)
	assert.Equal(t, SolanaSignatureLen, len(solana.Signature{}))
}

func TestCheckCommitment(t *testing.T) {
	type test struct {
		commitment      string
		watcher         string
		isReobservation bool
		result          bool
	}
	tests := []test{
		// New observation success cases
		{commitment: "finalized", watcher: "finalized", isReobservation: false, result: true},
		{commitment: "confirmed", watcher: "confirmed", isReobservation: false, result: true},

		// New observation failure cases
		{commitment: "finalized", watcher: "confirmed", isReobservation: false, result: false},
		{commitment: "confirmed", watcher: "finalized", isReobservation: false, result: false},

		// Reobservation success cases
		{commitment: "finalized", watcher: "finalized", isReobservation: true, result: true},
		{commitment: "confirmed", watcher: "finalized", isReobservation: true, result: true},

		// Reobservation case that never really happen because only the finalized watcher does reobservations
		{commitment: "finalized", watcher: "confirmed", isReobservation: true, result: false},
		{commitment: "confirmed", watcher: "confirmed", isReobservation: true, result: true},
	}

	for _, tc := range tests {
		var label string
		if tc.isReobservation {
			label = "reobserved_"
		} else {
			label = "new_"
		}
		label += tc.commitment + "_message_on_" + tc.watcher + "_watcher"
		t.Run(label, func(t *testing.T) {
			commitment := rpc.CommitmentType(tc.commitment)
			watcher := rpc.CommitmentType(tc.watcher)
			s := &SolanaWatcher{commitment: watcher}
			assert.Equal(t, tc.result, s.checkCommitment(commitment, tc.isReobservation))
		})
	}
}
