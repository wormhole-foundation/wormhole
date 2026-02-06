package solana

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

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

const whLogPrefix = "Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"

func TestIsPossibleWormholeMessageSuccess(t *testing.T) {
	logs := []string{
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY invoke [1]",
		"Program log: Instruction: PostUnlock",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
		"Program log: Sequence: 149587",
		"Program 11111111111111111111111111111111 invoke [3]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [3]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [3]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 27143 of 33713 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
		"Program log: unlock message seq: 149587",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY consumed 88681 of 94020 compute units",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY success",
	}

	require.True(t, isPossibleWormholeMessage(whLogPrefix, logs))
}

func TestIsPossibleWormholeMessageFail(t *testing.T) {
	logs := []string{
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [1]",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program 11111111111111111111111111111111 invoke [2]",
		"Program 11111111111111111111111111111111 success",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 92816 of 154700 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
	}

	require.False(t, isPossibleWormholeMessage(whLogPrefix, logs))
}

func Test_validateTransactionMeta(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		meta    *rpc.TransactionMeta
		wantErr bool
		errMsg  string
	}{

		// Happy path
		{"non-nil meta", &rpc.TransactionMeta{}, false, ""},
		// Error cases
		{"metadata is nil", nil, true, "metadata is nil"},
		{"non-nil meta, failed tx", &rpc.TransactionMeta{Err: "err"}, true, "transaction failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := validateTransactionMeta(tt.meta)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("validateTransactionMeta() failed: %v", gotErr)
				}
			}
		})
	}
}

func TestParseMessagePublicationAccount(t *testing.T) {

	// Define well-formed Solana mainnet PostedMessage account data for reliable and unreliable messages
	var (
		// Solana mainnet PostedMessage account: `GU76rcJ4rgw5sZQ2efVRC4yUZyhwrVM6pTGs2kFhckYy`
		validMessageAccountDataReliable, _ = hex.DecodeString("6d73670020000000000000000000000000000000000000000000000000000000000000000000000000404d836900000000f5de1400000000000100ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5850000000100000000000000000000000000000000000000000000000000009ed268a30dd700000000000000000000000089f4e8011c35831130c4c3ab95e53de9411d2fcc00040000000000000000000000006722b2c28d7d299b56a5febedbbe865a84ee0d7d00040000000000000000000000000000000000000000000000000000000000000000")

		// Solana mainnet PostedMessageUnreliable account: `GU76rcJ4rgw5sZQ2efVRC4yUZyhwrVM6pTGs2kFhckYy`
		validMessageAccountDataUnreliable, _ = hex.DecodeString("6d73750001000000000000000000000000000000000000000000000000000000000000000000000000785b83690000000018c4340000000000010034cdc6b2623f36d60ae820e95b60f764e81ec2cd3b57b77e3f8e25ddd43ac37300000000")

		// Emitters
		emitter, _          = hex.DecodeString("ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5")
		emitterAddrReliable = vaa.Address(emitter)

		emitterUnreliable, _  = hex.DecodeString("34cdc6b2623f36d60ae820e95b60f764e81ec2cd3b57b77e3f8e25ddd43ac373")
		emitterAddrUnreliable = vaa.Address(emitterUnreliable)

		// Payload portion of the reliablemessage account data
		payload, _ = hex.DecodeString("0100000000000000000000000000000000000000000000000000009ed268a30dd700000000000000000000000089f4e8011c35831130c4c3ab95e53de9411d2fcc00040000000000000000000000006722b2c28d7d299b56a5febedbbe865a84ee0d7d00040000000000000000000000000000000000000000000000000000000000000000")
	)

	const (
		// Define error string for testing. This is returned by the borsh-go library when it fails to
		// deserialize a struct. In this case, we're relying on the library to fail when
		// the message account data can't be deserialized into [MessagePublicationAccount].
		errStringBorsh  = "failed to read required bytes"
		errStringPrefix = "message account data is nil"
	)

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		messageAccountData *MessageAccountData
		want               *MessagePublicationAccount
		errStr             string
	}{
		{
			name: "success -- reliable message",

			messageAccountData: &MessageAccountData{validMessageAccountDataReliable},
			want: &MessagePublicationAccount{
				VaaVersion:       0,
				ConsistencyLevel: 32,
				EmitterAuthority: vaa.Address{},
				MessageStatus:    0,
				Gap:              [3]byte{0, 0, 0},
				SubmissionTime:   1770212672,
				Nonce:            0,
				Sequence:         1367797,
				EmitterChain:     1,
				EmitterAddress:   emitterAddrReliable,
				Payload:          payload,
			},
			errStr: "",
		},
		{
			name:               "success -- unreliable message",
			messageAccountData: &MessageAccountData{validMessageAccountDataUnreliable},
			want: &MessagePublicationAccount{
				VaaVersion:       0,
				ConsistencyLevel: 1,
				EmitterAuthority: vaa.Address{},
				MessageStatus:    0,
				Gap:              [3]byte{},
				SubmissionTime:   1770216312,
				Nonce:            0,
				Sequence:         3458072,
				EmitterChain:     1,
				EmitterAddress:   emitterAddrUnreliable,
				Payload:          nil, // borsh deserialization results in this being nil rather than an empty slice
			},
			errStr: "",
		},
		{
			name:               "failure -- nil argument",
			messageAccountData: nil,
			want:               &MessagePublicationAccount{},
			errStr:             errStringPrefix,
		},
		{
			name:               "failure -- no data following prefix (msg)",
			messageAccountData: &MessageAccountData{[]byte("msg")},
			want:               &MessagePublicationAccount{},
			errStr:             errStringBorsh,
		},
		{
			name:               "failure -- no data following prefix (msu)",
			messageAccountData: &MessageAccountData{[]byte("msu")},
			want:               &MessagePublicationAccount{},
			errStr:             errStringBorsh,
		},
		{
			name:               "failure -- truncated data (msg)",
			messageAccountData: &MessageAccountData{validMessageAccountDataReliable[:len(validMessageAccountDataReliable)-1]},
			want:               &MessagePublicationAccount{},
			errStr:             errStringBorsh,
		},
		{
			name:               "failure -- truncated data (msu)",
			messageAccountData: &MessageAccountData{validMessageAccountDataReliable[:len(validMessageAccountDataReliable)-1]},
			want:               &MessagePublicationAccount{},
			errStr:             errStringBorsh,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := ParseMessagePublicationAccount(tt.messageAccountData)
			if gotErr != nil {
				// We didn't expect an error, but we got one.
				if tt.errStr == "" {
					t.Errorf("ParseMessagePublicationAccount() failed: %v", gotErr)
				}
				return
			}
			if tt.errStr != "" {
				// We want an error. Make sure it's the right one.
				require.ErrorContains(t, gotErr, tt.errStr, "ParseMessagePublicationAccount() succeeded unexpectedly")
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
