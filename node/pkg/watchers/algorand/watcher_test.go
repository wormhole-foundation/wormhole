package algorand

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/algorand/go-algorand-sdk/types"
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const APP_ID = 86525623

// helper to create a watcher for testing.
func newTestWatcher(msgC chan *common.MessagePublication) *Watcher {
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)
	return NewWatcher("", "", "", "", APP_ID, msgC, obsvReqC)
}

// helper to build a valid 8-byte big-endian encoding of a uint64.
func uint64Bytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// helper to build a minimal publishMessage transaction that passes
// the watcher's filter checks.
func makePublishTxn(appID uint64, nonce []byte, payload []byte, seqLog string) types.SignedTxnWithAD {
	txn := types.SignedTxnWithAD{}
	txn.Txn.ApplicationID = types.AppIndex(appID)
	txn.Txn.ApplicationArgs = [][]byte{
		[]byte("publishMessage"),
		payload,
		nonce,
	}
	txn.EvalDelta.Logs = []string{seqLog}
	return txn
}

func TestLookAtTxnInnerTxn(t *testing.T) {
	msgC := make(chan *common.MessagePublication)
	w := newTestWatcher(msgC)

	var expectedSequence uint64 = 993

	b, err := os.ReadFile("test_nested_inner.block.json")
	require.NoError(t, err)

	txn := types.SignedTxnInBlock{}
	require.NoError(t, json.Unmarshal(b, &txn))

	// The json blob has the relevant log encoded as base64 because
	// Go's json package refuses to properly encode/decode invalid utf8.
	b64Data := txn.EvalDelta.InnerTxns[2].EvalDelta.InnerTxns[0].EvalDelta.Logs[0]
	bb, err := base64.StdEncoding.DecodeString(b64Data)
	require.NoError(t, err)
	txn.EvalDelta.InnerTxns[2].EvalDelta.InnerTxns[0].EvalDelta.Logs[0] = string(bb)

	logger, _ := zap.NewProduction()
	observations := gatherObservations(w, txn.SignedTxnWithAD, 0, logger)

	require.Len(t, observations, 1)
	assert.Equal(t, expectedSequence, observations[0].sequence)
}

func TestGatherObservations(t *testing.T) {
	logger, _ := zap.NewProduction()

	tests := []struct {
		name          string
		txn           types.SignedTxnWithAD
		expectedCount int
		expectedNonce uint32
		expectedSeq   uint64
	}{
		{
			name:          "valid 8-byte nonce and sequence",
			txn:           makePublishTxn(APP_ID, uint64Bytes(7), []byte("hello"), string(uint64Bytes(42))),
			expectedCount: 1,
			expectedNonce: 7,
			expectedSeq:   42,
		},
		{
			name:          "valid zero nonce and zero sequence",
			txn:           makePublishTxn(APP_ID, uint64Bytes(0), []byte(""), string(uint64Bytes(0))),
			expectedCount: 1,
			expectedNonce: 0,
			expectedSeq:   0,
		},
		{
			name:          "valid max uint32 nonce",
			txn:           makePublishTxn(APP_ID, uint64Bytes(0xFFFFFFFF), []byte("payload"), string(uint64Bytes(100))),
			expectedCount: 1,
			expectedNonce: 0xFFFFFFFF,
			expectedSeq:   100,
		},
		{
			name:          "wrong app ID",
			txn:           makePublishTxn(99999, uint64Bytes(1), []byte("hello"), string(uint64Bytes(1))),
			expectedCount: 0,
		},
		{
			name: "wrong method name",
			txn: func() types.SignedTxnWithAD {
				txn := makePublishTxn(APP_ID, uint64Bytes(1), []byte("hello"), string(uint64Bytes(1)))
				txn.Txn.ApplicationArgs[0] = []byte("notPublishMessage")
				return txn
			}(),
			expectedCount: 0,
		},
		{
			name: "too few application args (2 instead of 3)",
			txn: func() types.SignedTxnWithAD {
				txn := types.SignedTxnWithAD{}
				txn.Txn.ApplicationID = types.AppIndex(APP_ID)
				txn.Txn.ApplicationArgs = [][]byte{
					[]byte("publishMessage"),
					[]byte("hello"),
				}
				txn.EvalDelta.Logs = []string{string(uint64Bytes(1))}
				return txn
			}(),
			expectedCount: 0,
		},
		{
			name: "too many application args (4 instead of 3)",
			txn: func() types.SignedTxnWithAD {
				txn := types.SignedTxnWithAD{}
				txn.Txn.ApplicationID = types.AppIndex(APP_ID)
				txn.Txn.ApplicationArgs = [][]byte{
					[]byte("publishMessage"),
					[]byte("hello"),
					uint64Bytes(1),
					[]byte("extra"),
				}
				txn.EvalDelta.Logs = []string{string(uint64Bytes(1))}
				return txn
			}(),
			expectedCount: 0,
		},
		{
			name: "no logs",
			txn: func() types.SignedTxnWithAD {
				txn := makePublishTxn(APP_ID, uint64Bytes(1), []byte("hello"), "")
				txn.EvalDelta.Logs = nil
				return txn
			}(),
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			w := newTestWatcher(msgC)

			obs := gatherObservations(w, tc.txn, 0, logger)

			require.Len(t, obs, tc.expectedCount)
			if tc.expectedCount > 0 {
				assert.Equal(t, tc.expectedNonce, obs[0].nonce)
				assert.Equal(t, tc.expectedSeq, obs[0].sequence)
			}
		})
	}
}

// TestGatherObservations_InvalidLengths verifies that malformed nonce or
// sequence lengths are safely skipped rather than panicking.
// See: https://developer.algorand.org/docs/get-details/parameter_tables/
// Algorand enforces no minimum byte length on individual application args,
// and the core bridge contract does not validate ApplicationArgs[2].
func TestGatherObservations_InvalidLengths(t *testing.T) {
	logger, _ := zap.NewProduction()

	tests := []struct {
		name  string
		nonce []byte
		seq   string
	}{
		{"short nonce (2 bytes)", []byte{0xDE, 0xAD}, string(uint64Bytes(1))},
		{"short nonce (4 bytes)", []byte{0, 0, 0, 1}, string(uint64Bytes(1))},
		{"short nonce (7 bytes)", make([]byte, 7), string(uint64Bytes(1))},
		{"empty nonce", []byte{}, string(uint64Bytes(1))},
		{"oversized nonce (16 bytes)", make([]byte, 16), string(uint64Bytes(1))},
		{"short sequence (2 bytes)", uint64Bytes(1), "AB"},
		{"short sequence (4 bytes)", uint64Bytes(1), string(make([]byte, 4))},
		{"empty sequence", uint64Bytes(1), ""},
		{"oversized sequence (16 bytes)", uint64Bytes(1), string(make([]byte, 16))},
		{"both short", []byte{0xFF}, "X"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			w := newTestWatcher(msgC)

			txn := makePublishTxn(APP_ID, tc.nonce, []byte("payload"), tc.seq)

			// Must not panic.
			obs := gatherObservations(w, txn, 0, logger)
			assert.Empty(t, obs, "malformed input should produce no observations")
		})
	}
}

func TestGatherObservations_MaxDepth(t *testing.T) {
	logger, _ := zap.NewProduction()
	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)

	// Build a chain of nested inner transactions at exactly MAX_DEPTH.
	// The valid publishMessage sits at the deepest level.
	validTxn := makePublishTxn(APP_ID, uint64Bytes(1), []byte("deep"), string(uint64Bytes(99)))

	// Nest it MAX_DEPTH levels deep (should be rejected).
	nested := validTxn
	for i := 0; i < MAX_DEPTH; i++ {
		wrapper := types.SignedTxnWithAD{}
		wrapper.EvalDelta.InnerTxns = []types.SignedTxnWithAD{nested}
		nested = wrapper
	}

	obs := gatherObservations(w, nested, 0, logger)
	assert.Empty(t, obs, "should not observe messages beyond MAX_DEPTH")

	// Nest it MAX_DEPTH-1 levels deep (should succeed).
	nested = validTxn
	for i := 0; i < MAX_DEPTH-1; i++ {
		wrapper := types.SignedTxnWithAD{}
		wrapper.EvalDelta.InnerTxns = []types.SignedTxnWithAD{nested}
		nested = wrapper
	}

	obs = gatherObservations(w, nested, 0, logger)
	require.Len(t, obs, 1)
	assert.Equal(t, uint64(99), obs[0].sequence)
}

func TestGatherObservations_MultipleInnerTxns(t *testing.T) {
	logger, _ := zap.NewProduction()
	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)

	// Two valid publishMessage inner transactions at the same level.
	txn1 := makePublishTxn(APP_ID, uint64Bytes(10), []byte("first"), string(uint64Bytes(100)))
	txn2 := makePublishTxn(APP_ID, uint64Bytes(20), []byte("second"), string(uint64Bytes(200)))

	wrapper := types.SignedTxnWithAD{}
	wrapper.EvalDelta.InnerTxns = []types.SignedTxnWithAD{txn1, txn2}

	obs := gatherObservations(w, wrapper, 0, logger)

	require.Len(t, obs, 2)
	assert.Equal(t, uint32(10), obs[0].nonce)
	assert.Equal(t, uint64(100), obs[0].sequence)
	assert.Equal(t, uint32(20), obs[1].nonce)
	assert.Equal(t, uint64(200), obs[1].sequence)
}

func TestGatherObservations_MixedValidAndInvalid(t *testing.T) {
	logger, _ := zap.NewProduction()
	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)

	// One valid, one with a short nonce, one valid again.
	valid1 := makePublishTxn(APP_ID, uint64Bytes(1), []byte("ok1"), string(uint64Bytes(10)))
	invalid := makePublishTxn(APP_ID, []byte{0xBA, 0xD0}, []byte("bad"), string(uint64Bytes(20)))
	valid2 := makePublishTxn(APP_ID, uint64Bytes(3), []byte("ok2"), string(uint64Bytes(30)))

	wrapper := types.SignedTxnWithAD{}
	wrapper.EvalDelta.InnerTxns = []types.SignedTxnWithAD{valid1, invalid, valid2}

	obs := gatherObservations(w, wrapper, 0, logger)

	require.Len(t, obs, 2, "invalid txn should be skipped without affecting valid ones")
	assert.Equal(t, uint64(10), obs[0].sequence)
	assert.Equal(t, uint64(30), obs[1].sequence)
}

func TestLookAtTxn_PublishesToMsgC(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(msgC)
	logger, _ := zap.NewProduction()

	txn := types.SignedTxnInBlock{}
	txn.SignedTxnWithAD = makePublishTxn(APP_ID, uint64Bytes(5), []byte("test payload"), string(uint64Bytes(77)))

	block := types.Block{
		BlockHeader: types.BlockHeader{
			TimeStamp: 1700000000,
			GenesisID: "testnet-v1",
		},
	}

	lookAtTxn(w, txn, block, logger, false)

	select {
	case msg := <-msgC:
		assert.Equal(t, uint32(5), msg.Nonce)
		assert.Equal(t, uint64(77), msg.Sequence)
		assert.Equal(t, vaa.ChainIDAlgorand, msg.EmitterChain)
		assert.Equal(t, []byte("test payload"), msg.Payload)
		assert.Equal(t, uint8(0), msg.ConsistencyLevel)
		assert.Equal(t, time.Unix(1700000000, 0), msg.Timestamp)
		assert.False(t, msg.IsReobservation)
	default:
		t.Fatal("expected message on msgC, got none")
	}
}

func TestLookAtTxn_Reobservation(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(msgC)
	logger, _ := zap.NewProduction()

	txn := types.SignedTxnInBlock{}
	txn.SignedTxnWithAD = makePublishTxn(APP_ID, uint64Bytes(1), []byte("reobs"), string(uint64Bytes(1)))

	block := types.Block{
		BlockHeader: types.BlockHeader{TimeStamp: 1700000000},
	}

	lookAtTxn(w, txn, block, logger, true)

	select {
	case msg := <-msgC:
		assert.True(t, msg.IsReobservation)
	default:
		t.Fatal("expected message on msgC, got none")
	}
}

func TestLookAtTxn_NoMatchProducesNoMessage(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(msgC)
	logger, _ := zap.NewProduction()

	// Transaction with wrong app ID.
	txn := types.SignedTxnInBlock{}
	txn.SignedTxnWithAD = makePublishTxn(99999, uint64Bytes(1), []byte("wrong"), string(uint64Bytes(1)))

	block := types.Block{}

	lookAtTxn(w, txn, block, logger, false)

	select {
	case <-msgC:
		t.Fatal("expected no message on msgC, got one")
	default:
		// OK
	}
}

func TestLookAtTxn_InvalidNonceDoesNotBlock(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(msgC)
	logger, _ := zap.NewProduction()

	// Transaction with a short nonce -- must not panic or block.
	txn := types.SignedTxnInBlock{}
	txn.SignedTxnWithAD = makePublishTxn(APP_ID, []byte{0xDE, 0xAD}, []byte("bad"), string(uint64Bytes(1)))

	block := types.Block{
		BlockHeader: types.BlockHeader{TimeStamp: 1700000000},
	}

	lookAtTxn(w, txn, block, logger, false)

	select {
	case <-msgC:
		t.Fatal("expected no message for invalid nonce, got one")
	default:
		// OK
	}
}

func TestGatherObservations_EmitterAddress(t *testing.T) {
	logger, _ := zap.NewProduction()
	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)

	txn := makePublishTxn(APP_ID, uint64Bytes(0), []byte(""), string(uint64Bytes(0)))

	// Set a known sender address.
	var sender types.Address
	for i := range sender {
		sender[i] = byte(i)
	}
	txn.Txn.Sender = sender

	obs := gatherObservations(w, txn, 0, logger)

	require.Len(t, obs, 1)

	var expected vaa.Address
	copy(expected[:], sender[:])
	assert.Equal(t, expected, obs[0].emitterAddress)
}

func TestGatherObservations_PayloadPreserved(t *testing.T) {
	logger, _ := zap.NewProduction()
	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)

	payload := []byte{0x01, 0x00, 0xFF, 0xAB, 0xCD}
	txn := makePublishTxn(APP_ID, uint64Bytes(0), payload, string(uint64Bytes(0)))

	obs := gatherObservations(w, txn, 0, logger)

	require.Len(t, obs, 1)
	assert.Equal(t, payload, obs[0].payload)
}

func TestNewWatcher(t *testing.T) {
	msgC := make(chan *common.MessagePublication, 1)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 1)

	w := NewWatcher("http://indexer", "token123", "http://algod", "token456", 12345, msgC, obsvReqC)

	assert.Equal(t, "http://indexer", w.indexerRPC)
	assert.Equal(t, "token123", w.indexerToken)
	assert.Equal(t, "http://algod", w.algodRPC)
	assert.Equal(t, "token456", w.algodToken)
	assert.Equal(t, uint64(12345), w.appid)
	assert.Equal(t, uint64(0), w.next_round)
}
