package solana

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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

func TestIsPossibleWormholeMessageSequenceBeforePrefixFail(t *testing.T) {
	logs := []string{
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program log: Sequence: 100",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [1]",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 92816 of 154700 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
	}

	require.False(t, isPossibleWormholeMessage(whLogPrefix, logs))
}

func TestIsPossibleWormholeMessageMultiplePrefixesNoSequenceFail(t *testing.T) {
	logs := []string{
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [1]",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 50000 of 100000 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY invoke [1]",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth invoke [2]",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth consumed 27143 of 33713 compute units",
		"Program worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth success",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY success",
	}

	require.False(t, isPossibleWormholeMessage(whLogPrefix, logs))
}

func TestIsPossibleWormholeMessageMissingPrefixFail(t *testing.T) {
	logs := []string{
		"Program ComputeBudget111111111111111111111111111111 invoke [1]",
		"Program ComputeBudget111111111111111111111111111111 success",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY invoke [1]",
		"Program log: Sequence: 149587",
		"Program BLZRi6frs4X4DNLw56V4EXai1b6QVESN1BhHBTYM9VcY success",
	}

	require.False(t, isPossibleWormholeMessage(whLogPrefix, logs))
}

func TestConsistencyLevelCommitment(t *testing.T) {
	// Scenario: mapping of consistency levels to commitments, including invalid value.
	tests := []struct {
		level   ConsistencyLevel
		want    rpc.CommitmentType
		wantErr bool
	}{
		{level: consistencyLevelConfirmed, want: rpc.CommitmentConfirmed},
		{level: consistencyLevelFinalized, want: rpc.CommitmentFinalized},
		{level: ConsistencyLevel(9), wantErr: true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("level_%d", tc.level), func(t *testing.T) {
			got, err := tc.level.Commitment()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestAccountConsistencyLevelToCommitment(t *testing.T) {
	// Scenario: account consistency levels (u8) are mapped to Solana commitments.
	tests := []struct {
		level   uint8
		want    rpc.CommitmentType
		wantErr bool
	}{
		{level: 1, want: rpc.CommitmentConfirmed},
		{level: 32, want: rpc.CommitmentFinalized},
		{level: 2, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("level_%d", tc.level), func(t *testing.T) {
			got, err := accountConsistencyLevelToCommitment(tc.level)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseMessagePublicationAccount(t *testing.T) {
	proposal := testMessagePublicationAccount([]byte("payload"), 32)

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "valid",
			data: encodeMessagePublicationAccount(t, accountPrefixReliable, proposal),
		},
		{
			name:    "truncated",
			data:    []byte(accountPrefixReliable),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMessagePublicationAccount(tc.data)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, proposal.Nonce, got.Nonce)
			assert.Equal(t, proposal.Sequence, got.Sequence)
			assert.Equal(t, proposal.ConsistencyLevel, got.ConsistencyLevel)
			assert.True(t, bytes.Equal(proposal.EmitterAddress[:], got.EmitterAddress[:]))
			assert.Equal(t, proposal.Payload, got.Payload)
		})
	}
}

func TestUpdateLatestBlock(t *testing.T) {
	// Scenario: out-of-order updates only advance the latest block number.
	s := &SolanaWatcher{}
	s.updateLatestBlock(10)
	s.updateLatestBlock(5)
	s.updateLatestBlock(12)
	assert.Equal(t, uint64(12), s.getLatestFinalizedBlockNumber())
}

func TestProcessMessageAccount(t *testing.T) {
	tests := []struct {
		name             string
		chainID          vaa.ChainID
		commitment       rpc.CommitmentType
		prefix           string
		payload          []byte
		consistencyLevel uint8
		isReobservation  bool
		tweakProposal    func(*MessagePublicationAccount)
		wantCount        uint32
		wantUnreliable   bool
		wantReobs        bool
	}{
		{
			name:             "publishes_reliable",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixReliable,
			payload:          []byte("hello"),
			consistencyLevel: 32,
			wantCount:        1,
		},
		{
			name:             "skips_unfinalized",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixReliable,
			payload:          []byte("hello"),
			consistencyLevel: 32,
			tweakProposal: func(p *MessagePublicationAccount) {
				copy(p.EmitterAuthority[:], bytes.Repeat([]byte{0x01}, len(p.EmitterAuthority)))
			},
		},
		{
			name:             "pythnet_bypasses_finalization_check",
			chainID:          vaa.ChainIDPythNet,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixReliable,
			payload:          []byte("hello"),
			consistencyLevel: 32,
			tweakProposal: func(p *MessagePublicationAccount) {
				copy(p.EmitterAuthority[:], bytes.Repeat([]byte{0x01}, len(p.EmitterAuthority)))
				p.MessageStatus = 1
			},
			wantCount: 1,
		},
		{
			name:             "commitment_mismatch",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentConfirmed,
			prefix:           accountPrefixReliable,
			payload:          []byte("hello"),
			consistencyLevel: 32,
		},
		{
			name:             "invalid_consistency_level",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixReliable,
			payload:          []byte("hello"),
			consistencyLevel: 99,
		},
		{
			name:             "skips_unreliable_empty_payload",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixUnreliable,
			payload:          nil,
			consistencyLevel: 32,
		},
		{
			name:             "reobservation_commitment_override",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixReliable,
			payload:          []byte("hello"),
			consistencyLevel: 1,
			isReobservation:  true,
			wantCount:        1,
			wantReobs:        true,
		},
		{
			name:             "publishes_unreliable",
			chainID:          vaa.ChainIDSolana,
			commitment:       rpc.CommitmentFinalized,
			prefix:           accountPrefixUnreliable,
			payload:          []byte("hello"),
			consistencyLevel: 32,
			wantCount:        1,
			wantUnreliable:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			s := newTestWatcher(t, tc.chainID, tc.commitment, msgC)

			proposal := testMessagePublicationAccount(tc.payload, tc.consistencyLevel)
			if tc.tweakProposal != nil {
				tc.tweakProposal(&proposal)
			}
			data := encodeMessagePublicationAccount(t, tc.prefix, proposal)

			acc := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0x11}, solana.PublicKeyLength))
			num := s.processMessageAccount(s.logger, data, acc, tc.isReobservation, solana.Signature{})
			assert.Equal(t, tc.wantCount, num)

			if tc.wantCount == 0 {
				assert.Equal(t, 0, len(msgC))
				return
			}

			require.Equal(t, 1, len(msgC))
			msg := <-msgC
			require.NotNil(t, msg)
			assert.Equal(t, acc.Bytes(), msg.TxID)
			assert.Equal(t, time.Unix(int64(proposal.SubmissionTime), 0), msg.Timestamp)
			assert.Equal(t, proposal.Nonce, msg.Nonce)
			assert.Equal(t, proposal.Sequence, msg.Sequence)
			assert.Equal(t, tc.chainID, msg.EmitterChain)
			assert.Equal(t, proposal.EmitterAddress, msg.EmitterAddress)
			assert.Equal(t, proposal.ConsistencyLevel, msg.ConsistencyLevel)
			assert.Equal(t, proposal.Payload, msg.Payload)
			assert.Equal(t, tc.wantUnreliable, msg.Unreliable)
			assert.Equal(t, tc.wantReobs, msg.IsReobservation)
		})
	}
}

func TestProcessAccountSubscriptionData(t *testing.T) {
	// Scenario: subscription messages are validated and decoded; invalid input yields errors or no-ops.
	rawContract := "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
	pubkey := "01234567890123456789012345678901"

	proposal := testMessagePublicationAccount([]byte("hello"), 32)
	validAccountData := encodeMessagePublicationAccount(t, accountPrefixReliable, proposal)

	mustJSON := func(v interface{}) []byte {
		data, err := json.Marshal(v)
		require.NoError(t, err)
		return data
	}

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		wantMsg bool
	}{
		{
			name:    "invalid_json",
			data:    []byte("{"),
			wantErr: true,
		},
		{
			name: "subscription_error",
			data: mustJSON(map[string]interface{}{
				"jsonrpc": "2.0",
				"error":   map[string]interface{}{"code": 123, "message": "boom"},
				"id":      "1",
			}),
			wantErr: true,
		},
		{
			name: "no_params",
			data: mustJSON(map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "programNotification",
			}),
		},
		{
			name:    "owner_mismatch",
			data:    buildSubscriptionPayload(t, "other", "ABCDEFGHabcdefgh0123456789ABCDEF", []byte("msg")),
			wantErr: true,
		},
		{
			name: "invalid_base64",
			data: mustJSON(map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "programNotification",
				"params": map[string]interface{}{
					"result": map[string]interface{}{
						"context": map[string]interface{}{"slot": int64(1)},
						"value": map[string]interface{}{
							"pubkey": pubkey,
							"account": map[string]interface{}{
								"lamports":   int64(1),
								"data":       []string{"%%%", "base64"},
								"owner":      rawContract,
								"executable": false,
								"rentEpoch":  int64(0),
							},
						},
					},
					"subscription": 1,
				},
			}),
			wantErr: true,
		},
		{
			name: "empty_data",
			data: buildSubscriptionPayload(t, rawContract, pubkey, []byte{}),
		},
		{
			name: "truncated_data",
			data: buildSubscriptionPayload(t, rawContract, pubkey, []byte{0x01, 0x02}),
		},
		{
			name:    "valid_message",
			data:    buildSubscriptionPayload(t, rawContract, pubkey, validAccountData),
			wantMsg: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
			s.rawContract = rawContract

			err := s.processAccountSubscriptionData(context.TODO(), zap.NewNop(), tc.data, false)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.wantMsg {
				require.Equal(t, 1, len(msgC))
				<-msgC
			} else {
				assert.Equal(t, 0, len(msgC))
			}
		})
	}
}

func TestProcessInstructionEarlyReturns(t *testing.T) {
	// Scenario: instruction filtering should return early for non-matching cases.
	commitmentMismatchData := encodePostMessageData(t, 42, []byte("hi"), consistencyLevelConfirmed)

	tests := []struct {
		name    string
		inst    solana.CompiledInstruction
		wantErr bool
	}{
		{
			name: "program_mismatch",
			inst: solana.CompiledInstruction{ProgramIDIndex: 1},
		},
		{
			name: "empty_data",
			inst: solana.CompiledInstruction{ProgramIDIndex: 0, Data: []byte{}},
		},
		{
			name: "wrong_instruction",
			inst: solana.CompiledInstruction{ProgramIDIndex: 0, Data: []byte{0x02}},
		},
		{
			name:    "too_few_accounts",
			inst:    solana.CompiledInstruction{ProgramIDIndex: 0, Data: []byte{postMessageInstructionID}, Accounts: []uint16{1, 2}},
			wantErr: true,
		},
		{
			name:    "borsh_error",
			inst:    solana.CompiledInstruction{ProgramIDIndex: 0, Data: []byte{postMessageInstructionID}, Accounts: make([]uint16, postMessageInstructionMinNumAccounts)},
			wantErr: true,
		},
		{
			name:    "unsupported_consistency_level",
			inst:    solana.CompiledInstruction{ProgramIDIndex: 0, Data: append([]byte{postMessageInstructionID}, encodePostMessageData(t, 7, []byte("test"), ConsistencyLevel(9))...), Accounts: make([]uint16, postMessageInstructionMinNumAccounts)},
			wantErr: true,
		},
		{
			name: "commitment_mismatch",
			inst: solana.CompiledInstruction{
				ProgramIDIndex: 0,
				Data:           append([]byte{postMessageInstructionID}, commitmentMismatchData...),
				Accounts:       make([]uint16, postMessageInstructionMinNumAccounts),
			},
		},
	}

	s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, nil)
	tx := &solana.Transaction{
		Message:    solana.Message{AccountKeys: []solana.PublicKey{}},
		Signatures: []solana.Signature{{}},
	}
	signature := solana.Signature{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			found, err := s.processInstruction(context.TODO(), nil, 1, tc.inst, 0, tx, signature, 0, false)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.False(t, found)
		})
	}
}

func TestProcessInstructionValidPostMessage(t *testing.T) {
	// Scenario: valid PostMessage instructions should return found=true and publish via account fetch.
	// Not tested here: retry scheduling/backoff paths for failed account fetches.
	tests := []struct {
		name           string
		instructionID  byte
		accountPrefix  string
		wantUnreliable bool
	}{
		{
			name:          "reliable",
			instructionID: postMessageInstructionID,
			accountPrefix: accountPrefixReliable,
		},
		{
			name:           "unreliable",
			instructionID:  postMessageUnreliableInstructionID,
			accountPrefix:  accountPrefixUnreliable,
			wantUnreliable: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
			s.errC = make(chan error, 1)

			contract := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xAA}, solana.PublicKeyLength))
			messageAccount := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xBB}, solana.PublicKeyLength))
			s.contract = contract

			proposal := testMessagePublicationAccount([]byte("hello"), 32)
			accountData := encodeMessagePublicationAccount(t, tc.accountPrefix, proposal)
			srv := newAccountInfoServer(t, contract.String(), accountData)
			defer srv.Close()

			rpcClient := rpc.New(srv.URL)

			tx := &solana.Transaction{
				Message: solana.Message{
					AccountKeys: []solana.PublicKey{contract, messageAccount},
				},
				Signatures: []solana.Signature{{}},
			}

			data := encodePostMessageData(t, 7, []byte("hello"), consistencyLevelFinalized)
			inst := solana.CompiledInstruction{
				ProgramIDIndex: 0,
				Data:           append([]byte{tc.instructionID}, data...),
				Accounts:       []uint16{0, 1, 0, 0, 0, 0, 0, 0},
			}

			found, err := s.processInstruction(context.Background(), rpcClient, 1, inst, 0, tx, tx.Signatures[0], 0, false)
			require.NoError(t, err)
			assert.True(t, found)

			select {
			case msg := <-msgC:
				require.NotNil(t, msg)
				assert.Equal(t, tc.wantUnreliable, msg.Unreliable)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for message publication")
			}
		})
	}
}

func TestProcessTransaction(t *testing.T) {
	contract := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xAA}, solana.PublicKeyLength))
	messageAccount := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xBB}, solana.PublicKeyLength))
	otherKey := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xCC}, solana.PublicKeyLength))
	shimContract := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xDD}, solana.PublicKeyLength))

	proposal := testMessagePublicationAccount([]byte("hello"), 32)
	accountData := encodeMessagePublicationAccount(t, accountPrefixReliable, proposal)

	postMsgData := encodePostMessageData(t, 7, []byte("hello"), consistencyLevelFinalized)

	// programIndex==0 is treated as "contract not found", so the contract must be at index 1+.
	// Non-shim layout: [otherKey, contract, messageAccount]
	//                    idx 0     idx 1     idx 2
	contractIdx := uint16(1)

	matchingInstruction := solana.CompiledInstruction{
		ProgramIDIndex: contractIdx,
		Data:           append([]byte{postMessageInstructionID}, postMsgData...),
		Accounts:       []uint16{0, 2, 0, 0, 0, 0, 0, 0},
	}

	nonMatchingInstruction := solana.CompiledInstruction{
		ProgramIDIndex: 0,
		Data:           []byte{0xFF},
	}

	standardKeys := []solana.PublicKey{otherKey, contract, messageAccount}

	// Shim layout: [otherKey, contract, messageAccount, shimContract]
	//               idx 0     idx 1     idx 2           idx 3
	shimContractIdx := uint16(3)
	shimKeys := []solana.PublicKey{otherKey, contract, messageAccount, shimContract}

	// Pre-built shim instruction data from known-good hex encodings.
	shimPostMsgData := mustDecodeHex(t, "d63264d12622074c2a000000010b00000068656c6c6f20776f726c64")
	shimCoreData := mustDecodeHex(t, "082a0000000000000001")
	shimEventData := mustDecodeHex(t, "e445a52e51cb9a1d441b8f004d4c8970041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde00000000000000007c5b8167")

	shimTopLevelInst := solana.CompiledInstruction{
		ProgramIDIndex: shimContractIdx,
		Data:           shimPostMsgData,
	}
	shimCoreInnerInst := solana.CompiledInstruction{
		ProgramIDIndex: contractIdx,
		Data:           shimCoreData,
	}
	shimEventInnerInst := solana.CompiledInstruction{
		ProgramIDIndex: shimContractIdx,
		Data:           shimEventData,
	}

	tests := []struct {
		name              string
		shimEnabled       bool
		metaErr           interface{}
		accountKeys       []solana.PublicKey
		instructions      []solana.CompiledInstruction
		innerInstructions []rpc.InnerInstruction
		wantObservations  uint32
	}{
		// Non-shim cases.
		{
			name:         "failed_transaction",
			metaErr:      "some error",
			accountKeys:  standardKeys,
			instructions: []solana.CompiledInstruction{matchingInstruction},
		},
		{
			name:        "contract_at_index_zero",
			accountKeys: []solana.PublicKey{contract, messageAccount},
			instructions: []solana.CompiledInstruction{
				{ProgramIDIndex: 0, Data: append([]byte{postMessageInstructionID}, postMsgData...), Accounts: []uint16{0, 1, 0, 0, 0, 0, 0, 0}},
			},
		},
		{
			name:        "contract_not_in_accounts",
			accountKeys: []solana.PublicKey{otherKey, messageAccount},
			instructions: []solana.CompiledInstruction{
				{ProgramIDIndex: 0, Data: append([]byte{postMessageInstructionID}, postMsgData...), Accounts: []uint16{0, 1, 0, 0, 0, 0, 0, 0}},
			},
		},
		{
			name:         "no_matching_instructions",
			accountKeys:  standardKeys,
			instructions: []solana.CompiledInstruction{nonMatchingInstruction},
		},
		{
			name:             "single_top_level_match",
			accountKeys:      standardKeys,
			instructions:     []solana.CompiledInstruction{matchingInstruction},
			wantObservations: 1,
		},
		{
			name:         "top_level_non_matching_with_inner_match",
			accountKeys:  standardKeys,
			instructions: []solana.CompiledInstruction{nonMatchingInstruction},
			innerInstructions: []rpc.InnerInstruction{
				{Index: 0, Instructions: []solana.CompiledInstruction{matchingInstruction}},
			},
			wantObservations: 1,
		},
		{
			name:        "top_level_and_inner_matches",
			accountKeys: standardKeys,
			instructions: []solana.CompiledInstruction{
				matchingInstruction,
				nonMatchingInstruction,
			},
			innerInstructions: []rpc.InnerInstruction{
				{Index: 1, Instructions: []solana.CompiledInstruction{matchingInstruction}},
			},
			wantObservations: 2,
		},
		{
			name:        "multiple_top_level_matches",
			accountKeys: standardKeys,
			instructions: []solana.CompiledInstruction{
				matchingInstruction,
				matchingInstruction,
			},
			wantObservations: 2,
		},
		// Shim cases.
		{
			name:         "shim_failed_transaction",
			shimEnabled:  true,
			metaErr:      "some error",
			accountKeys:  shimKeys,
			instructions: []solana.CompiledInstruction{shimTopLevelInst},
			innerInstructions: []rpc.InnerInstruction{
				{Index: 0, Instructions: []solana.CompiledInstruction{shimCoreInnerInst, shimEventInnerInst}},
			},
		},
		{
			name:         "shim_top_level_direct",
			shimEnabled:  true,
			accountKeys:  shimKeys,
			instructions: []solana.CompiledInstruction{shimTopLevelInst},
			innerInstructions: []rpc.InnerInstruction{
				{Index: 0, Instructions: []solana.CompiledInstruction{shimCoreInnerInst, shimEventInnerInst}},
			},
			wantObservations: 1,
		},
		{
			name:         "shim_inner_integrator",
			shimEnabled:  true,
			accountKeys:  shimKeys,
			instructions: []solana.CompiledInstruction{nonMatchingInstruction},
			innerInstructions: []rpc.InnerInstruction{
				{Index: 0, Instructions: []solana.CompiledInstruction{
					{ProgramIDIndex: shimContractIdx, Data: shimPostMsgData},
					shimCoreInnerInst,
					shimEventInnerInst,
				}},
			},
			wantObservations: 1,
		},
		{
			name:         "shim_top_level_error_logged_not_counted",
			shimEnabled:  true,
			accountKeys:  shimKeys,
			instructions: []solana.CompiledInstruction{shimTopLevelInst},
			// Missing inner instructions causes shimProcessTopLevelInstruction to error.
			innerInstructions: []rpc.InnerInstruction{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 10)
			s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
			s.errC = make(chan error, 10)
			s.contract = contract

			if tc.shimEnabled {
				s.shimContractAddr = shimContract
				s.shimContractStr = shimContract.String()
				s.shimSetup()
			}

			srv := newAccountInfoServer(t, contract.String(), accountData)
			defer srv.Close()
			rpcClient := rpc.New(srv.URL)

			tx := &solana.Transaction{
				Message: solana.Message{
					AccountKeys:  tc.accountKeys,
					Instructions: tc.instructions,
				},
				Signatures: []solana.Signature{{}},
			}

			meta := &rpc.TransactionMeta{
				Err:               tc.metaErr,
				InnerInstructions: tc.innerInstructions,
			}

			num := s.processTransaction(context.Background(), rpcClient, tx, meta, 42, false)
			assert.Equal(t, tc.wantObservations, num)

			// Drain published messages and verify count.
			if tc.wantObservations > 0 {
				for i := uint32(0); i < tc.wantObservations; i++ {
					select {
					case msg := <-msgC:
						require.NotNil(t, msg)
					case <-time.After(2 * time.Second):
						t.Fatalf("timed out waiting for message %d", i)
					}
				}
			}
		})
	}
}

// Test helpers

func newTestWatcher(t *testing.T, chainID vaa.ChainID, commitment rpc.CommitmentType, msgC chan<- *common.MessagePublication) *SolanaWatcher {
	t.Helper()
	return &SolanaWatcher{
		logger:              zap.NewNop(),
		msgObservedLogLevel: zapcore.DebugLevel,
		chainID:             chainID,
		networkName:         chainID.String(),
		commitment:          commitment,
		msgC:                msgC,
	}
}

func testMessagePublicationAccount(payload []byte, consistencyLevel uint8) MessagePublicationAccount {
	var emitterAddress vaa.Address
	copy(emitterAddress[:], bytes.Repeat([]byte{0xAB}, len(emitterAddress)))

	return MessagePublicationAccount{
		VaaVersion:       1,
		ConsistencyLevel: consistencyLevel,
		MessageStatus:    0,
		SubmissionTime:   123,
		Nonce:            7,
		Sequence:         99,
		EmitterChain:     uint16(vaa.ChainIDSolana),
		EmitterAddress:   emitterAddress,
		Payload:          payload,
	}
}

// encodeMessagePublicationAccount produces a Borsh-compatible payload for ParseMessagePublicationAccount tests.
func encodeMessagePublicationAccount(t *testing.T, prefix string, proposal MessagePublicationAccount) []byte {
	t.Helper()
	if len(prefix) != 3 {
		t.Fatalf("prefix must be 3 bytes, got %d", len(prefix))
	}

	// NOTE: This is a minimal Borsh encoder for the fields in MessagePublicationAccount.
	buf := &bytes.Buffer{}
	buf.WriteString(prefix)
	buf.WriteByte(proposal.VaaVersion)
	buf.WriteByte(proposal.ConsistencyLevel)
	buf.Write(proposal.EmitterAuthority[:])
	buf.WriteByte(proposal.MessageStatus)
	buf.Write(proposal.Gap[:])
	writeLE(t, buf, proposal.SubmissionTime)
	writeLE(t, buf, proposal.Nonce)
	writeLE(t, buf, proposal.Sequence)
	writeLE(t, buf, proposal.EmitterChain)
	buf.Write(proposal.EmitterAddress[:])
	writeLE(t, buf, uint32(len(proposal.Payload))) // #nosec G115 -- Test data, payload length is always small.
	buf.Write(proposal.Payload)
	return buf.Bytes()
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func writeLE(t *testing.T, buf *bytes.Buffer, value interface{}) {
	t.Helper()
	require.NoError(t, binary.Write(buf, binary.LittleEndian, value))
}

// encodePostMessageData mirrors the borsh layout for PostMessageData to avoid a networked RPC dependency in tests.
func encodePostMessageData(t *testing.T, nonce uint32, payload []byte, consistency ConsistencyLevel) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	writeLE(t, buf, nonce)
	writeLE(t, buf, uint32(len(payload))) // #nosec G115 -- Test data, payload length is always small.
	buf.Write(payload)
	buf.WriteByte(byte(consistency))
	return buf.Bytes()
}

func buildSubscriptionPayload(t *testing.T, owner, pubkey string, accountData []byte) []byte {
	t.Helper()
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "programNotification",
		"params": map[string]interface{}{
			"result": map[string]interface{}{
				"context": map[string]interface{}{"slot": int64(1)},
				"value": map[string]interface{}{
					"pubkey": pubkey,
					"account": map[string]interface{}{
						"lamports":   int64(1),
						"data":       []string{base64.StdEncoding.EncodeToString(accountData), "base64"},
						"owner":      owner,
						"executable": false,
						"rentEpoch":  int64(0),
					},
				},
			},
			"subscription": 1,
		},
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return data
}

// newAccountInfoServer provides a minimal JSON-RPC getAccountInfo responder.
// It does not validate params or exercise network error handling.
func newAccountInfoServer(t *testing.T, owner string, accountData []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := common.SafeRead(r.Body)
		require.NoError(t, err)
		_ = r.Body.Close()

		var req map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &req))
		id := req["id"]

		method, _ := req["method"].(string)
		if method != "getAccountInfo" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]interface{}{
				"context": map[string]interface{}{"slot": int64(1)},
				"value": map[string]interface{}{
					"data":       []interface{}{base64.StdEncoding.EncodeToString(accountData), "base64"},
					"owner":      owner,
					"lamports":   int64(1),
					"executable": false,
					"rentEpoch":  int64(0),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(resp))
	}))
}
