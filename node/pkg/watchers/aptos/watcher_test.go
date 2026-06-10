package aptos

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const (
	testAptosAccount = "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625"
	testAptosHandle  = "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::WormholeMessageHandle"
	testExpectedType = "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::WormholeMessage"
)

// eventData returns a fresh, mutable map of the inner "data" object that
// observeData consumes. All values are strings (gjson .Uint() parses via
// strconv). Callers can mutate or delete fields before marshaling.
func eventData() map[string]any {
	return map[string]any{
		"sender":            "1",
		"payload":           "0xdeadbeef",
		"timestamp":         "1000",
		"nonce":             "42",
		"sequence":          "7",
		"consistency_level": "1",
	}
}

// Marshal JSON to mimic Aptos API
func marshalJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return string(b)
}

// validEvent returns a well-formed Aptos WormholeMessage JSON.
func validEvent() string {
	b, _ := json.Marshal(eventData())
	return string(b)
}

// eventWith replaces or removes a single field in validEvent.
// Pass value "" with remove=true to delete the field.
func eventWith(field, value string, remove bool) string {
	d := eventData()
	if remove {
		delete(d, field)
	} else {
		d[field] = value
	}
	b, _ := json.Marshal(d)
	return string(b)
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
				w.observeData(logger, gjson.Parse(tc.json), tc.nativeSeq, false)
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

	w.observeData(logger, gjson.Parse(json), 123, true)
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

/*
Most implementations have nonce as 32 bits. The Aptos implementation incorrectly has 64 bits.
It's possible to send a Wormhole message with too large of a nonce. These should be rejected, not truncated.
This test confirms that this stays true.
*/
func TestObserveDataNonceTooLargeFails(t *testing.T) {
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
		"nonce": "4294967296",
		"sequence": "42",
		"consistency_level": "15"
	}`

	w.observeData(logger, gjson.Parse(json), 123, true)
	require.Len(t, msgC, 0)
}

func TestVerifyEventType(t *testing.T) {
	w := &Watcher{
		chainID:        vaa.ChainIDAptos,
		networkID:      "aptos-test",
		aptosAccount:   testAptosAccount,
		aptosHandle:    testAptosHandle,
		aptosEventType: testExpectedType,
	}

	tests := []struct {
		name        string
		mutate      func(m map[string]any) // Function to mutate the event JSON
		expectError string
	}{
		{
			name:   "Happy path",
			mutate: nil,
		},
		{
			// Casing of the address is insignificant: guid.account_address is
			// compared case-insensitively.
			name: "Happy path casing change",
			mutate: func(m map[string]any) {
				m["guid"].(map[string]any)["account_address"] = "0x5Bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625" //nolint:forcetypeassert // test data shape is known
			},
		},
		{
			name:        "Missing 'type' in the JSON",
			mutate:      func(m map[string]any) { delete(m, "type") },
			expectError: "event missing 'type'",
		},
		{
			// Different address in the package segment of the type.
			name: "Wrong type package",
			mutate: func(m map[string]any) {
				m["type"] = "0x11c11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::WormholeMessage"
			},
			expectError: "event type mismatch",
		},
		{
			name:        "Wrong type module",
			mutate:      func(m map[string]any) { m["type"] = testAptosAccount + "::notstate::WormholeMessage" },
			expectError: "event type mismatch",
		},
		{
			name:        "Wrong type struct type",
			mutate:      func(m map[string]any) { m["type"] = testAptosAccount + "::state::NotWormholeMessage" },
			expectError: "event type mismatch",
		},
		{
			name:        "Missing account_address",
			mutate:      func(m map[string]any) { delete(m["guid"].(map[string]any), "account_address") }, //nolint:forcetypeassert // test data shape is known
			expectError: "event missing 'guid.account_address'",
		},
		{
			// Config carries the "0x" prefix; this event omits it on
			// guid.account_address. normalize0x should strip both sides.
			name: "0x prefix stripped on guid.account_address",
			mutate: func(m map[string]any) {
				m["guid"].(map[string]any)["account_address"] = strings.TrimPrefix(testAptosAccount, "0x") //nolint:forcetypeassert // test data shape is known
			},
		},
		{
			// Config carries the "0x" prefix; this event omits it on
			// the type field. normalize0x should strip both sides.
			name:   "0x prefix stripped on type",
			mutate: func(m map[string]any) { m["type"] = strings.TrimPrefix(testExpectedType, "0x") },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := pollEvent()
			if tc.mutate != nil {
				tc.mutate(e)
			}
			err := w.verifyEventType(gjson.Parse(marshalJSON(t, e)))

			if tc.expectError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			}
		})
	}
}

// pollEvent returns a fresh map representing one valid polling-batch event.
// Defaults are hardcoded; callers mutate "sequence_number" (and any other
// field, including nested "data" fields) before passing to encodeBatch.
func pollEvent() map[string]any {
	return map[string]any{
		"guid": map[string]any{
			"creation_number": "2",
			"account_address": testAptosAccount,
		},
		"type":            testExpectedType,
		"sequence_number": "100",
		"data":            eventData(),
	}
}

// encodeBatch marshals events into the JSON array string that
// processPollingBatch expects via gjson.Parse.
func encodeBatch(t *testing.T, events ...map[string]any) string {
	return marshalJSON(t, events)
}

// assertLoggedError verifies the failure surface of the batch processors, which
// no longer return an error but instead log it via zap and continue. The
// haystack for each entry is its message plus its "error" context field (the
// verifyEventType error is attached via zap.Error). When want is empty, no
// error-level entry should have been logged at all.
func assertLoggedError(t *testing.T, logs *observer.ObservedLogs, want string) {
	t.Helper()

	if want == "" {
		assert.Empty(t, logs.All(), "expected no error logs")
		return
	}

	for _, e := range logs.All() {
		s := e.Message
		if ev, ok := e.ContextMap()["error"]; ok {
			s += " " + fmt.Sprint(ev)
		}
		if strings.Contains(s, want) {
			return
		}
	}
	assert.Failf(t, "missing expected log", "expected a logged error containing %q, got %v", want, logs.All())
}

func TestProcessPollingBatch(t *testing.T) {
	// happy path: two consecutive events. Built by mutating pollEvent() defaults.
	happyFirst := pollEvent()
	happySecond := pollEvent()
	happySecond["sequence_number"] = "101"

	// Bad event information: wrong event type triggers verifyEventType, which
	// causes processPollingBatch to return an error before touching nextSequence.
	badHandle := pollEvent()
	badHandle["type"] = "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::NotAWormholeMessage"

	invalidEvent := pollEvent()
	delete(invalidEvent, "data")

	// Startup skip: with initialNextSeq=0 and eventSeq != 0, the loop hits the
	// "avoid publishing an old observation on startup" branch (watcher.go:283) —
	// nextSequence is advanced to eventSeq+1 and observeData is never called.
	startupSkip := pollEvent() // sequence_number defaults to "100"

	// Normalization: same valid event, but with the "0x" prefix stripped from
	// guid.account_address only. verifyEventType should normalize and accept.
	prefixStrippedGuid := pollEvent()
	prefixStrippedGuid["guid"] = map[string]any{
		"creation_number": "2",
		"account_address": strings.TrimPrefix(testAptosAccount, "0x"),
	}

	// Normalization: same valid event, but with the "0x" prefix stripped from
	// the type field only. verifyEventType should normalize and accept.
	prefixStrippedType := pollEvent()
	prefixStrippedType["type"] = strings.TrimPrefix(testExpectedType, "0x")

	tests := []struct {
		name             string
		events           []map[string]any // marshaled to a JSON array unless rawJSON is set
		rawJSON          string           // overrides events when non-empty
		initialNextSeq   uint64
		expectedNextSeq  uint64
		expectedMsgCount int
		expectError      string // substring expected in returned error; "" means nil
	}{
		{
			name:             "happy path: one event",
			events:           []map[string]any{happyFirst},
			initialNextSeq:   100,
			expectedNextSeq:  101,
			expectedMsgCount: 1,
			expectError:      "",
		},
		{
			name:             "0x prefix stripped on guid.account_address",
			events:           []map[string]any{prefixStrippedGuid},
			initialNextSeq:   100,
			expectedNextSeq:  101,
			expectedMsgCount: 1,
			expectError:      "",
		},
		{
			name:             "0x prefix stripped on type",
			events:           []map[string]any{prefixStrippedType},
			initialNextSeq:   100,
			expectedNextSeq:  101,
			expectedMsgCount: 1,
			expectError:      "",
		},
		{
			name:             "startup skip: nextSeq=0, eventSeq!=0",
			events:           []map[string]any{startupSkip},
			initialNextSeq:   0,
			expectedNextSeq:  101, // eventSeq (100) + 1
			expectedMsgCount: 0,   // event is skipped, not published
			expectError:      "",
		},
		{
			name:             "happy path: two consecutive events",
			events:           []map[string]any{happyFirst, happySecond},
			initialNextSeq:   100,
			expectedNextSeq:  102,
			expectedMsgCount: 2,
			expectError:      "",
		},
		{
			name:             "empty",
			events:           []map[string]any{},
			initialNextSeq:   100,
			expectedNextSeq:  100,
			expectedMsgCount: 0,
			expectError:      "",
		},
		{
			name:             "Bad event information",
			events:           []map[string]any{badHandle},
			initialNextSeq:   100,
			expectedNextSeq:  100,
			expectedMsgCount: 0,
			expectError:      "event type mismatch",
		},
		{
			name:             "One valid, one invalid",
			events:           []map[string]any{happyFirst, invalidEvent},
			initialNextSeq:   100,
			expectedNextSeq:  101,
			expectedMsgCount: 1,
			expectError:      "",
		},
		{
			name:             "One invalid, one valid",
			events:           []map[string]any{invalidEvent, happyFirst},
			initialNextSeq:   100,
			expectedNextSeq:  101,
			expectedMsgCount: 1,
			expectError:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.ErrorLevel)
			logger := zap.New(core)
			msgC := make(chan *common.MessagePublication, 16)
			w := &Watcher{
				chainID:        vaa.ChainIDAptos,
				networkID:      "aptos-test",
				msgC:           msgC,
				aptosAccount:   testAptosAccount,
				aptosHandle:    testAptosHandle,
				aptosEventType: testExpectedType,
			}

			jsonStr := tc.rawJSON
			if jsonStr == "" {
				jsonStr = encodeBatch(t, tc.events...)
			}

			nextSeq := tc.initialNextSeq
			w.processPollingBatch(logger, gjson.Parse(jsonStr), &nextSeq)

			assertLoggedError(t, logs, tc.expectError)
			assert.Equal(t, tc.expectedNextSeq, nextSeq)
			assert.Len(t, msgC, tc.expectedMsgCount)
		})
	}
}

func TestProcessReobs(t *testing.T) {
	// happy path: two consecutive events. Built by mutating pollEvent() defaults.
	happyFirst := pollEvent()
	happySecond := pollEvent()
	happySecond["sequence_number"] = "101"

	// Bad event information: wrong event type triggers verifyEventType, which
	// causes processPollingBatch to return an error before touching nextSequence.
	badHandle := pollEvent()
	badHandle["type"] = "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::NotAWormholeMessage"

	invalidEvent := pollEvent()
	delete(invalidEvent, "data")

	tests := []struct {
		name             string
		events           []map[string]any // marshaled to a JSON array unless rawJSON is set
		rawJSON          string           // overrides events when non-empty
		nativeSeq        uint64
		expectedMsgCount int
		expectError      string
	}{
		{
			name:             "happy path: one event",
			events:           []map[string]any{happyFirst},
			nativeSeq:        100,
			expectedMsgCount: 1,
			expectError:      "",
		},
		{
			name:             "Bad sequence",
			events:           []map[string]any{happyFirst},
			nativeSeq:        99, // Wrong sequence for the message
			expectedMsgCount: 0,
			expectError:      "newSeq != nativeSeq",
		},
		{
			name:             "Bad event information",
			events:           []map[string]any{badHandle},
			nativeSeq:        100,
			expectedMsgCount: 0,
			expectError:      "event type mismatch",
		},
		{
			name:             "One valid, one invalid",
			events:           []map[string]any{happyFirst, invalidEvent},
			nativeSeq:        100,
			expectedMsgCount: 1,
			expectError:      "",
		},
		{ // Stops after the first event
			name:             "One invalid, one valid",
			events:           []map[string]any{invalidEvent, happyFirst},
			nativeSeq:        100,
			expectedMsgCount: 0,
			expectError:      "",
		},
		{
			name:             "empty",
			events:           []map[string]any{},
			nativeSeq:        100,
			expectedMsgCount: 0,
			expectError:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.ErrorLevel)
			logger := zap.New(core)
			msgC := make(chan *common.MessagePublication, 16)
			w := &Watcher{
				chainID:        vaa.ChainIDAptos,
				networkID:      "aptos-test",
				msgC:           msgC,
				aptosAccount:   testAptosAccount,
				aptosHandle:    testAptosHandle,
				aptosEventType: testExpectedType,
			}

			jsonStr := tc.rawJSON
			if jsonStr == "" {
				jsonStr = encodeBatch(t, tc.events...)
			}

			w.processReobsBatch(logger, gjson.Parse(jsonStr), tc.nativeSeq)

			assertLoggedError(t, logs, tc.expectError)
			assert.Len(t, msgC, tc.expectedMsgCount)
		})
	}
}
