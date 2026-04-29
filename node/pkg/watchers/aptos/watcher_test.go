package aptos

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// validEvent returns a well-formed Aptos WormholeMessage JSON.
func validEvent() string {
	return `{
		"sender": "1",
		"payload": "0xdeadbeef",
		"timestamp": "1000",
		"nonce": "42",
		"sequence": "7",
		"consistency_level": "1"
	}`
}

// eventWith replaces or removes a single field in validEvent.
// Pass value "" with remove=true to delete the field.
func eventWith(field, value string, remove bool) string {
	base := map[string]string{
		"sender":            "1",
		"payload":           "0xdeadbeef",
		"timestamp":         "1000",
		"nonce":             "42",
		"sequence":          "7",
		"consistency_level": "1",
	}
	if remove {
		delete(base, field)
	} else {
		base[field] = value
	}
	json := "{"
	first := true
	for k, v := range base {
		if !first {
			json += ","
		}
		json += fmt.Sprintf("%q:%q", k, v)
		first = false
	}
	json += "}"
	return json
}

func TestObserveData(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		nativeSeq   uint64
		expectMsg   bool   // true if we expect an observation on msgC
		expectError string // substring expected in logged error, empty if no error
	}{
		// Happy path
		{
			name:      "valid event",
			json:      validEvent(),
			nativeSeq: 1,
			expectMsg: true,
		},

		// Missing fields
		{
			name:        "missing sender",
			json:        eventWith("sender", "", true),
			expectError: "sender field missing",
		},
		{
			name:        "missing payload",
			json:        eventWith("payload", "", true),
			expectError: "payload field missing",
		},
		{
			name:        "missing timestamp",
			json:        eventWith("timestamp", "", true),
			expectError: "timestamp field missing",
		},
		{
			name:        "missing nonce",
			json:        eventWith("nonce", "", true),
			expectError: "nonce field missing",
		},
		{
			name:        "missing sequence",
			json:        eventWith("sequence", "", true),
			expectError: "sequence field missing",
		},
		{
			name:        "missing consistency_level",
			json:        eventWith("consistency_level", "", true),
			expectError: "consistencyLevel field missing",
		},

		// Payload validation
		{
			name:        "payload empty string",
			json:        eventWith("payload", "", false),
			expectError: "payload missing 0x prefix",
		},
		{
			name:        "payload no 0x prefix",
			json:        eventWith("payload", "deadbeef", false),
			expectError: "payload missing 0x prefix",
		},
		{
			name:        "payload single char",
			json:        eventWith("payload", "0", false),
			expectError: "payload missing 0x prefix",
		},
		{
			name:      "payload 0x only",
			json:      eventWith("payload", "0x", false),
			expectMsg: true, // empty payload is valid
		},
		{
			name:        "payload invalid hex",
			json:        eventWith("payload", "0xZZZZ", false),
			expectError: "payload decode",
		},
		{
			name:        "payload odd length hex",
			json:        eventWith("payload", "0xabc", false),
			expectError: "payload decode",
		},

		// Range validation
		{
			name:        "nonce exceeds uint32",
			json:        eventWith("nonce", "4294967296", false), // MaxUint32 + 1
			expectError: "nonce is larger than expected MaxUint32",
		},
		{
			name:        "consistency_level exceeds uint8",
			json:        eventWith("consistency_level", "256", false), // MaxUint8 + 1
			expectError: "consistency level is larger than expected MaxUint8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.ErrorLevel)
			logger := zap.New(core)
			msgC := make(chan *common.MessagePublication, 1)
			w := &Watcher{
				chainID:   vaa.ChainIDAptos,
				networkID: "aptos-test",
				msgC:      msgC,
			}

			// Must not panic for any input.
			require.NotPanics(t, func() {
				w.observeData(logger, gjson.Parse(tc.json), tc.nativeSeq, nil)
			})

			if tc.expectError != "" {
				require.Equal(t, 1, logs.Len(), "expected exactly one error log")
				assert.Contains(t, logs.All()[0].Message, tc.expectError)
				assert.Empty(t, msgC, "should not produce observation on error")
			}

			if tc.expectMsg {
				require.Len(t, msgC, 1, "expected one observation")
				msg := <-msgC
				assert.Equal(t, vaa.ChainIDAptos, msg.EmitterChain)
			}
		})
	}
}

func TestObserveDataFields(t *testing.T) {
	core, _ := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	msgC := make(chan *common.MessagePublication, 1)
	w := &Watcher{
		chainID:   vaa.ChainIDAptos,
		networkID: "aptos-test",
		msgC:      msgC,
	}

	json := `{
		"sender": "5",
		"payload": "0xcafebabe",
		"timestamp": "1700000000",
		"nonce": "99",
		"sequence": "42",
		"consistency_level": "15"
	}`

	validatedObservation, err := watchers.ValidateObservationRequest(&gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDAptos), TxHash: make([]byte, common.TxIDLenMin)}, vaa.ChainIDAptos)
	require.NoError(t, err)

	w.observeData(logger, gjson.Parse(json), 123, &validatedObservation)
	require.Len(t, msgC, 1)
	msg := <-msgC

	// Emitter: sender=5, big-endian u64 in last 8 bytes of 32-byte address.
	var expectedAddr vaa.Address
	expectedAddr[31] = 5
	assert.Equal(t, expectedAddr, msg.EmitterAddress)

	// Payload
	expectedPayload, _ := hex.DecodeString("cafebabe")
	assert.Equal(t, expectedPayload, msg.Payload)

	// Scalar fields
	assert.Equal(t, uint32(99), msg.Nonce)
	assert.Equal(t, uint64(42), msg.Sequence)
	assert.Equal(t, uint8(15), msg.ConsistencyLevel)
	assert.Equal(t, int64(1700000000), msg.Timestamp.Unix())
	assert.True(t, msg.IsReobservation)
}
