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
	w, err := NewWatcher(vaa.ChainIDAptos, "aptos-test", "http://localhost", testAptosAccount, testAptosHandle, nil, nil)
	require.NoError(t, err)

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
			// Casing of the address is significant: the API always returns lowercased
			// addresses, so an uppercased guid.account_address is a mismatch.
			name: "Uppercased address fails",
			mutate: func(m map[string]any) {
				m["guid"].(map[string]any)["account_address"] = "0x5Bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625" //nolint:forcetypeassert // test data shape is known
			},
			expectError: "event guid.account_address mismatch",
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
			// guid.account_address. The address comparison ignores the prefix.
			name: "0x prefix stripped on guid.account_address",
			mutate: func(m map[string]any) {
				m["guid"].(map[string]any)["account_address"] = strings.TrimPrefix(testAptosAccount, "0x") //nolint:forcetypeassert // test data shape is known
			},
		},
		{
			// Config carries the "0x" prefix; this event omits it on
			// the type field. The embedded address comparison ignores the prefix.
			name:   "0x prefix stripped on type",
			mutate: func(m map[string]any) { m["type"] = strings.TrimPrefix(testExpectedType, "0x") },
		},
		{
			// A type tag without exactly three "::" segments is malformed.
			name:        "Malformed type: too few segments",
			mutate:      func(m map[string]any) { m["type"] = testAptosAccount + "::WormholeMessage" },
			expectError: "event type mismatch",
		},
		{
			name:        "Malformed type: too many segments",
			mutate:      func(m map[string]any) { m["type"] = testExpectedType + "::extra" },
			expectError: "event type mismatch",
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

// TestVerifyEventTypeShortFormAddress confirms the byte comparison treats the Aptos API's
// short-form addresses (leading zeros stripped) as equal to the zero-padded configured form.
// The watcher is configured with the full 64-char address; the event uses the stripped form.
func TestVerifyEventTypeShortFormAddress(t *testing.T) {
	tests := []struct {
		name      string
		fullAddr  string // configured, zero-padded 64-char form
		shortAddr string // form returned by the API, leading zeros stripped
	}{
		{
			// Special address collapsed to a single nibble.
			name:      "single nibble (0x1)",
			fullAddr:  "0x0000000000000000000000000000000000000000000000000000000000000001",
			shortAddr: "0x1",
		},
		{
			// One full leading zero byte (two nibbles) stripped; remainder is even-length.
			name:      "leading zero byte stripped",
			fullAddr:  "0x007730cd28ee1cdc9e999336cbc430f99e7c44397c0aa77516f6f23a78559bb5",
			shortAddr: "0x7730cd28ee1cdc9e999336cbc430f99e7c44397c0aa77516f6f23a78559bb5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWatcher(vaa.ChainIDAptos, "aptos-test", "http://localhost", tc.fullAddr, tc.fullAddr+"::module::EventHandle", nil, nil)
			require.NoError(t, err)

			event := map[string]any{
				"guid": map[string]any{"creation_number": "0", "account_address": tc.shortAddr},
				"type": tc.shortAddr + "::module::Event",
			}
			require.NoError(t, w.verifyEventType(gjson.Parse(marshalJSON(t, event))))
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

func TestNewWatcher(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		handle      string
		expectError string
	}{
		{
			name:    "valid lowercase",
			account: testAptosAccount,
			handle:  testAptosHandle,
		},
		{
			name:        "uppercased account",
			account:     "0x5Bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
			handle:      testAptosHandle,
			expectError: "aptosAccount",
		},
		{
			name:        "uppercased address in handle",
			account:     testAptosAccount,
			handle:      "0x5Bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::WormholeMessageHandle",
			expectError: "aptosHandle address segment",
		},
		{
			name:        "handle missing 'Handle' suffix",
			account:     testAptosAccount,
			handle:      testExpectedType,
			expectError: "does not end with 'Handle'",
		},
		{
			name:        "invalid account hex",
			account:     "0xzz",
			handle:      testAptosHandle,
			expectError: "not a valid lowercase Aptos address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWatcher(vaa.ChainIDAptos, "aptos", "http://localhost", tc.account, tc.handle, nil, nil)

			if tc.expectError == "" {
				require.NoError(t, err)
				require.NotNil(t, w)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			}
		})
	}
}

func TestValidateAptosHandle(t *testing.T) {
	tests := []struct {
		name        string
		handle      string
		expectError string
	}{
		{
			name:   "valid handle",
			handle: testAptosHandle,
		},
		{
			name:        "too few segments",
			handle:      "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::WormholeMessageHandle",
			expectError: "must have the form",
		},
		{
			name:        "too many segments",
			handle:      testAptosHandle + "::extra",
			expectError: "must have the form",
		},
		{
			name:        "no separators",
			handle:      "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
			expectError: "must have the form",
		},
		{
			name:        "uppercased address segment",
			handle:      "0x5Bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::WormholeMessageHandle",
			expectError: "is not a valid lowercase Aptos address",
		},
		{
			name:        "invalid hex address segment",
			handle:      "0xzz::state::WormholeMessageHandle",
			expectError: "is not a valid lowercase Aptos address",
		},
		{
			name:        "empty address segment",
			handle:      "::state::WormholeMessageHandle",
			expectError: "is not a valid lowercase Aptos address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAptosHandle(tc.handle)
			if tc.expectError == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			}
		})
	}
}

func TestParseAptosAddrLeftPads(t *testing.T) {
	// A short address must land in the low-order (rightmost) bytes: 0x01 -> 0x00..01,
	// not 0x01 followed by zeros.
	got, ok := parseAptosAddr("0x01")
	require.True(t, ok)
	want := [32]byte{}
	want[31] = 0x01
	assert.Equal(t, want, got)

	// A wrongly right-padded (left-aligned) layout must NOT be produced.
	wrong := [32]byte{}
	wrong[0] = 0x01
	assert.NotEqual(t, wrong, got)

	// Odd-length (single-nibble) short form, as the Aptos API returns special addresses
	// like "0x1"/"0x0", must be padded and land in the rightmost byte.
	got, ok = parseAptosAddr("0x1")
	require.True(t, ok)
	want = [32]byte{}
	want[31] = 0x01
	assert.Equal(t, want, got)

	// A stripped leading zero byte (two nibbles) yields the same canonical bytes as the
	// full zero-padded form.
	stripped, ok := parseAptosAddr("0x7730cd28ee1cdc9e999336cbc430f99e7c44397c0aa77516f6f23a78559bb5")
	require.True(t, ok)
	padded, ok := parseAptosAddr("0x007730cd28ee1cdc9e999336cbc430f99e7c44397c0aa77516f6f23a78559bb5")
	require.True(t, ok)
	assert.Equal(t, padded, stripped)

	// Multi-byte values stay big-endian and right-aligned.
	got, ok = parseAptosAddr("0x0102")
	require.True(t, ok)
	want = [32]byte{}
	want[30] = 0x01
	want[31] = 0x02
	assert.Equal(t, want, got)

	// Uppercase hex and the "0X" prefix are rejected: only lowercase is valid.
	_, ok = parseAptosAddr("0xABCD")
	assert.False(t, ok, "uppercase hex must be rejected")
	_, ok = parseAptosAddr("0X01")
	assert.False(t, ok, "uppercase 0X prefix must be rejected")
}

// TestParseAptosAddrLengths exercises the full spectrum of input lengths the RPC could
// conceivably return, since verifyEventType feeds it untrusted address strings. Aptos addresses
// are 32 bytes (64 hex nibbles); the parser accepts any non-empty hex string up to 64 nibbles
// (odd or even, since the API strips leading zeros, including down to a single nibble) and rejects
// empty input or anything longer than 64 nibbles. Every accepted value left-pads into the canonical
// 32-byte form.
func TestParseAptosAddrLengths(t *testing.T) {
	// hexN returns a value of n hex nibbles (without the "0x" prefix), e.g. "1212...".
	hexN := func(n int) string {
		return strings.Repeat("12", n/2) + strings.Repeat("1", n%2)
	}

	tests := []struct {
		name   string
		addr   string
		wantOK bool
	}{
		// Empty / prefix-only: nothing to decode.
		{name: "empty string", addr: "", wantOK: false},
		{name: "prefix only", addr: "0x", wantOK: false},

		// Minimum-length short forms the API returns for special addresses (e.g. 0x0, 0x1).
		{name: "single nibble", addr: "0x1", wantOK: true},
		{name: "single zero nibble", addr: "0x0", wantOK: true},

		// Odd-length values in range: the parser pads them to an even nibble count.
		{name: "odd length 3", addr: "0x" + hexN(3), wantOK: true},
		{name: "odd length 63", addr: "0x" + hexN(63), wantOK: true},

		// Even-length values in range.
		{name: "even length 2", addr: "0x" + hexN(2), wantOK: true},
		{name: "even length 62", addr: "0x" + hexN(62), wantOK: true},

		// Boundary at the 64-nibble (32-byte) maximum.
		{name: "max length 64", addr: "0x" + hexN(64), wantOK: true},
		{name: "one over max (65)", addr: "0x" + hexN(65), wantOK: false},
		{name: "two over max (66)", addr: "0x" + hexN(66), wantOK: false},

		// Length is measured after stripping "0x", so an unprefixed 64-nibble value is still valid.
		{name: "max length 64 no prefix", addr: hexN(64), wantOK: true},

		// Length is in range but the content is not valid hex.
		{name: "non-hex in range", addr: "0xzz", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := parseAptosAddr(tc.addr)
			require.Equal(t, tc.wantOK, ok)
			if !ok {
				// Rejected inputs must yield the zero value and never a partially-decoded address.
				assert.Equal(t, [32]byte{}, out)
			}
		})
	}
}

func TestProcessPollingBatch(t *testing.T) {
	// happy path: two consecutive events. Built by mutating pollEvent() defaults.
	happyFirst := pollEvent()
	happySecond := pollEvent()
	happySecond["sequence_number"] = "101"

	// Bad event information: wrong event type fails verifyEventType, so the event is
	// skipped (not published) but its sequence slot is still consumed (nextSequence advances).
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
			expectedNextSeq:  101, // slot consumed even though the event is skipped
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
			w, err := NewWatcher(vaa.ChainIDAptos, "aptos-test", "http://localhost", testAptosAccount, testAptosHandle, msgC, nil)
			require.NoError(t, err)

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
			w, err := NewWatcher(vaa.ChainIDAptos, "aptos-test", "http://localhost", testAptosAccount, testAptosHandle, msgC, nil)
			require.NoError(t, err)

			jsonStr := tc.rawJSON
			if jsonStr == "" {
				jsonStr = encodeBatch(t, tc.events...)
			}

			w.processReobservationBatch(logger, gjson.Parse(jsonStr), tc.nativeSeq)

			assertLoggedError(t, logs, tc.expectError)
			assert.Len(t, msgC, tc.expectedMsgCount)
		})
	}
}
