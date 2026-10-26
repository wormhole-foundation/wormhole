package stellar

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	dto "github.com/prometheus/client_model/go"
	stellarxdr "github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// XDR construction helpers
// ---------------------------------------------------------------------------

func scSym(s string) stellarxdr.ScVal {
	sym := stellarxdr.ScSymbol(s)
	return stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvSymbol, Sym: &sym}
}

func scU32(v uint32) stellarxdr.ScVal {
	u := stellarxdr.Uint32(v)
	return stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvU32, U32: &u}
}

func scU64(v uint64) stellarxdr.ScVal {
	u := stellarxdr.Uint64(v)
	return stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvU64, U64: &u}
}

func scBytesVal(b []byte) stellarxdr.ScVal {
	sb := stellarxdr.ScBytes(b)
	return stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvBytes, Bytes: &sb}
}

// eventFields returns the fields of a Soroban message_published event value.
// This mirrors the on-chain format that parseMessageFromXDR consumes.
func eventFields(nonce uint32, seq uint64, emitter []byte, payload []byte, cl uint32) stellarxdr.ScMap {
	return stellarxdr.ScMap{
		{Key: scSym("nonce"), Val: scU32(nonce)},
		{Key: scSym("sequence"), Val: scU64(seq)},
		{Key: scSym("emitter_address"), Val: scBytesVal(emitter)},
		{Key: scSym("payload"), Val: scBytesVal(payload)},
		{Key: scSym("consistency_level"), Val: scU32(cl)},
	}
}

// makeMapB64 encodes an ScMap as base64 XDR.
func makeMapB64(t *testing.T, scm stellarxdr.ScMap) string {
	t.Helper()
	m := &scm
	b64, err := stellarxdr.MarshalBase64(stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvMap, Map: &m})
	require.NoError(t, err)
	return b64
}

// makeEventValueB64 encodes a Soroban message_published event value as base64 XDR.
func makeEventValueB64(t *testing.T, nonce uint32, seq uint64, emitter []byte, payload []byte, cl uint32) string {
	t.Helper()
	return makeMapB64(t, eventFields(nonce, seq, emitter, payload, cl))
}

// makeTopicB64 encodes a Soroban symbol ScVal as base64 XDR.
func makeTopicB64(t *testing.T, sym string) string {
	t.Helper()
	b64, err := stellarxdr.MarshalBase64(scSym(sym))
	require.NoError(t, err)
	return b64
}

// ---------------------------------------------------------------------------
// Mock Soroban RPC server
// ---------------------------------------------------------------------------

type mockTx struct {
	status    string // "SUCCESS", "FAILED", "NOT_FOUND"
	ledger    uint64
	createdAt int64
}

type mockEvent struct {
	id             string
	ledger         uint64
	ledgerClosedAt string // RFC3339
	contractID     string
	txHash         string
	topic0B64      string
	topic1B64      string
	valueB64       string
}

type mockSorobanRPC struct {
	mu           sync.Mutex
	latestLedger uint64
	events       []mockEvent
	transactions map[string]mockTx
	methodCounts map[string]int
	// failGetEventsCall is the 1-based number of the getEvents call that fails, or 0 for none.
	failGetEventsCall int
}

func newMockRPC(latestLedger uint64) *mockSorobanRPC {
	return &mockSorobanRPC{
		latestLedger: latestLedger,
		transactions: make(map[string]mockTx),
		methodCounts: make(map[string]int),
	}
}

func (m *mockSorobanRPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := common.SafeRead(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var req struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	m.methodCounts[req.Method]++
	fail := req.Method == "getEvents" && m.methodCounts[req.Method] == m.failGetEventsCall
	m.mu.Unlock()
	if fail {
		http.Error(w, "injected failure", http.StatusServiceUnavailable)
		return
	}

	var result any
	switch req.Method {
	case "getLatestLedger":
		m.mu.Lock()
		result = map[string]any{"sequence": m.latestLedger, "protocolVersion": 21}
		m.mu.Unlock()
	case "getEvents":
		events, err := m.handleGetEvents(req.Params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result = events
	case "getTransaction":
		result = m.handleGetTransaction(req.Params)
	default:
		http.Error(w, fmt.Sprintf("unknown method: %s", req.Method), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (m *mockSorobanRPC) handleGetEvents(params json.RawMessage) (map[string]any, error) {
	var p struct {
		StartLedger uint64 `json:"startLedger"`
		Pagination  struct {
			Limit  int    `json:"limit"`
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}
	json.Unmarshal(params, &p) //nolint:errcheck
	if p.StartLedger != 0 && p.Pagination.Cursor != "" {
		return nil, errors.New("ledger ranges and cursor cannot both be set")
	}

	limit := p.Pagination.Limit
	if limit <= 0 {
		limit = 128
	}

	m.mu.Lock()
	allEvents := m.events
	latestLedger := m.latestLedger
	m.mu.Unlock()

	var filtered []mockEvent
	if p.Pagination.Cursor != "" {
		afterCursor := false
		for _, e := range allEvents {
			if afterCursor {
				filtered = append(filtered, e)
			}
			if e.id == p.Pagination.Cursor {
				afterCursor = true
			}
		}
	} else {
		for _, e := range allEvents {
			if e.ledger >= p.StartLedger {
				filtered = append(filtered, e)
			}
		}
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	eventJSONs := make([]map[string]any, 0, len(filtered))
	for _, e := range filtered {
		eventJSONs = append(eventJSONs, map[string]any{
			"id":             e.id,
			"ledger":         e.ledger,
			"ledgerClosedAt": e.ledgerClosedAt,
			"type":           "contract",
			"contractId":     e.contractID,
			"txHash":         e.txHash,
			"topic":          []string{e.topic0B64, e.topic1B64},
			"value":          e.valueB64,
		})
	}

	return map[string]any{
		"events":       eventJSONs,
		"latestLedger": latestLedger,
	}, nil
}

func (m *mockSorobanRPC) handleGetTransaction(params json.RawMessage) map[string]any {
	var p struct {
		Hash string `json:"hash"`
	}
	json.Unmarshal(params, &p) //nolint:errcheck

	m.mu.Lock()
	tx, ok := m.transactions[p.Hash]
	latestLedger := m.latestLedger
	m.mu.Unlock()

	if !ok || tx.status == "NOT_FOUND" {
		return map[string]any{"status": "NOT_FOUND", "latestLedger": latestLedger}
	}

	return map[string]any{
		"status":       tx.status,
		"ledger":       tx.ledger,
		"createdAt":    tx.createdAt,
		"latestLedger": latestLedger,
	}
}

func (m *mockSorobanRPC) methodCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.methodCounts[method]
}

// ---------------------------------------------------------------------------
// Test watcher factory
// ---------------------------------------------------------------------------

const (
	testContract    = "CBWQUIB4R65Z2DGC263FQ7BBI7TGIGOLFTYMLE6QPWBD5QDOUVJY3AKR"
	testNetworkID   = "stellar-test"
	testStartLedger = uint64(100)
)

func newTestWatcher(rpcURL string, maxPerPoll int, msgC chan<- *common.MessagePublication) *watcher {
	w := NewWatcher(
		rpcURL,
		testContract,
		testNetworkID,
		vaa.ChainIDStellar,
		testStartLedger,
		700*time.Millisecond,
		10*time.Second,
		maxPerPoll,
		msgC,
		nil,
		common.MainNet,
	)
	w.logger = zap.NewNop()
	return w
}

// makeMsgEvent builds a message_published event in the given ledger. Its tx hash, nonce, and
// sequence all derive from idx.
func makeMsgEvent(t *testing.T, idx uint32, ledger uint64) mockEvent {
	t.Helper()
	emitter := make([]byte, 32)
	emitter[0] = 0x01
	return mockEvent{
		id:             fmt.Sprintf("%010d-%010d", ledger, idx),
		ledger:         ledger,
		ledgerClosedAt: "2024-01-01T00:00:00Z",
		contractID:     testContract,
		txHash:         fmt.Sprintf("%064x", idx),
		topic0B64:      makeTopicB64(t, "wormhole"),
		topic1B64:      makeTopicB64(t, "message_published"),
		valueB64:       makeEventValueB64(t, idx, uint64(idx), emitter, []byte("pay"), 0),
	}
}

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// ---------------------------------------------------------------------------
// pollOnce tests
// ---------------------------------------------------------------------------

func TestPollOnce_MessagePublished(t *testing.T) {
	emitter := make([]byte, 32)
	emitter[31] = 0xAB
	payload := []byte("hello wormhole")
	txHash := "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	closedAt := "2024-06-15T12:00:00Z"
	expectedTS, _ := time.Parse(time.RFC3339, closedAt)

	mock := newMockRPC(200)
	mock.events = []mockEvent{
		{
			id:             "0000000100-0000000001",
			ledger:         100,
			ledgerClosedAt: closedAt,
			contractID:     testContract,
			txHash:         txHash,
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, 7, 42, emitter, payload, 0),
		},
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	_, err := w.pollOnce(context.Background(), zap.NewNop())
	require.NoError(t, err)

	select {
	case mp := <-msgC:
		assert.Equal(t, uint32(7), mp.Nonce)
		assert.Equal(t, uint64(42), mp.Sequence)
		assert.Equal(t, uint8(0), mp.ConsistencyLevel)
		assert.Equal(t, vaa.ChainIDStellar, mp.EmitterChain)
		assert.Equal(t, payload, mp.Payload)
		assert.False(t, mp.IsReobservation)

		assert.Equal(t, expectedTS.UTC(), mp.Timestamp.UTC(),
			"timestamp must come from ledgerClosedAt, not time.Now()")

		assert.Equal(t, mustDecodeHex(txHash), mp.TxID,
			"TxID must be the hex-decoded transaction hash")

		var expectedEmitter vaa.Address
		copy(expectedEmitter[:], emitter)
		assert.Equal(t, expectedEmitter, mp.EmitterAddress)

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message")
	}

	assert.Equal(t, uint64(101), w.nextLedger, "nextLedger must advance past the processed ledger")
}

func TestPollOnce_Pagination(t *testing.T) {
	// With maxPerPoll=2 and 3 events, the watcher must make two getEvents calls
	// (using cursor-based pagination) and deliver all 3 messages.
	mock := newMockRPC(300)
	mock.events = []mockEvent{
		makeMsgEvent(t, 1, 100),
		makeMsgEvent(t, 2, 101),
		makeMsgEvent(t, 3, 102),
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 2, msgC) // maxPerPoll = 2

	_, err := w.pollOnce(context.Background(), zap.NewNop())
	require.NoError(t, err)

	var received []*common.MessagePublication
	for i := 0; i < 3; i++ {
		select {
		case mp := <-msgC:
			received = append(received, mp)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for message %d of 3", i+1)
		}
	}
	assert.Len(t, received, 3, "all 3 events must be delivered across paginated responses")

	// Verify no extra messages leaked.
	select {
	case <-msgC:
		t.Fatal("unexpected extra message on channel")
	case <-time.After(20 * time.Millisecond):
	}

	// Two getEvents calls: first page (2 events), second page (1 event via cursor).
	assert.Equal(t, 2, mock.methodCount("getEvents"), "expected 2 getEvents calls for pagination")
}

func TestPollOnce_PageFailureMidLedger(t *testing.T) {
	// A page that ends inside a ledger, followed by a failed cursor request, must not
	// skip the rest of that ledger on the next poll.
	tests := []struct {
		name    string
		ledgers []uint64
	}{
		{"ledger spans pages", []uint64{100, 100, 100}},
		{"page ends mid-ledger", []uint64{100, 101, 101}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := newMockRPC(300)
			for i, ledger := range tc.ledgers {
				mock.events = append(mock.events, makeMsgEvent(t, uint32(i+1), ledger))
			}
			mock.failGetEventsCall = 2 // first cursor request

			srv := httptest.NewServer(mock)
			defer srv.Close()

			msgC := make(chan *common.MessagePublication, 10)
			w := newTestWatcher(srv.URL, 2, msgC)

			_, err := w.pollOnce(context.Background(), zap.NewNop())
			require.Error(t, err)
			assert.Equal(t, tc.ledgers[1], w.nextLedger, "a failed page must resume at the last full page's ledger")
			_, err = w.pollOnce(context.Background(), zap.NewNop())
			require.NoError(t, err)

			close(msgC)
			var seqs []uint64
			for mp := range msgC {
				seqs = append(seqs, mp.Sequence)
			}
			assert.Contains(t, seqs, uint64(3), "event after the failed page must be delivered on retry")
			assert.Equal(t, tc.ledgers[2]+1, w.nextLedger)
		})
	}
}

func TestPollOnce_SkipsEmptyEmitter(t *testing.T) {
	emptyEmitter := []byte{}

	mock := newMockRPC(200)
	mock.events = []mockEvent{
		{
			id:             "0000000100-0000000001",
			ledger:         100,
			ledgerClosedAt: "2024-01-01T00:00:00Z",
			contractID:     testContract,
			txHash:         fmt.Sprintf("%064x", 1),
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, 1, 1, emptyEmitter, []byte("pay"), 0),
		},
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	_, err := w.pollOnce(context.Background(), zap.NewNop())
	require.NoError(t, err)

	select {
	case mp := <-msgC:
		t.Fatalf("expected no message but got seq=%d", mp.Sequence)
	case <-time.After(50 * time.Millisecond):
		// correct: nothing published
	}
}

func TestPollOnce_SkipsConsistencyLevelOverflow(t *testing.T) {
	// An event whose consistency_level does not fit in uint8 must be dropped.
	emitter := make([]byte, 32)
	emitter[0] = 0x01

	mock := newMockRPC(200)
	mock.events = []mockEvent{
		{
			id:             "0000000100-0000000001",
			ledger:         100,
			ledgerClosedAt: "2024-01-01T00:00:00Z",
			contractID:     testContract,
			txHash:         fmt.Sprintf("%064x", 1),
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, 1, 1, emitter, []byte("pay"), 256),
		},
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	_, err := w.pollOnce(context.Background(), zap.NewNop())
	require.NoError(t, err)

	select {
	case mp := <-msgC:
		t.Fatalf("expected no message but got seq=%d", mp.Sequence)
	case <-time.After(50 * time.Millisecond):
		// correct: nothing published
	}
}

func TestPollOnce_NoEventsAdvancesNextLedger(t *testing.T) {
	// When getEvents returns no events, nextLedger must advance to latestLedger
	// from the getEvents response. No separate getLatestLedger call should be made.
	mock := newMockRPC(500)

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	advanced, err := w.pollOnce(context.Background(), zap.NewNop())
	require.NoError(t, err)
	assert.True(t, advanced)
	assert.Equal(t, uint64(500), w.nextLedger)

	assert.Equal(t, 0, mock.methodCount("getLatestLedger"),
		"getLatestLedger must not be called separately; latestLedger comes from the getEvents response")
}

func TestParseMessageFromXDR_Rejects(t *testing.T) {
	valid := func() stellarxdr.ScMap { return eventFields(1, 1, make([]byte, 32), nil, 0) }
	validRaw, err := base64.StdEncoding.DecodeString(makeMapB64(t, valid()))
	require.NoError(t, err)

	type testCase struct{ value, wantErr string }
	tests := map[string]testCase{
		"nil map":          {"AAAAEQAAAAA=", "not a map"},
		"trailing byte":    {base64.StdEncoding.EncodeToString(append(validRaw, 0)), "failed to unmarshal"},
		"emitter 31 bytes": {makeEventValueB64(t, 1, 1, make([]byte, 31), nil, 0), "emitter address is 31 bytes"},
		"emitter 33 bytes": {makeEventValueB64(t, 1, 1, make([]byte, 33), nil, 0), "emitter address is 33 bytes"},
		"non-symbol key": {makeMapB64(t, append(valid()[1:], stellarxdr.ScMapEntry{Key: scU32(0), Val: scU32(1)})),
			"non-symbol or duplicate key"},
		"duplicate key": {makeMapB64(t, append(valid()[1:], stellarxdr.ScMapEntry{Key: scSym("sequence"), Val: scU32(1)})),
			"non-symbol or duplicate key"},
		"extra key": {makeMapB64(t, append(valid(), stellarxdr.ScMapEntry{Key: scSym("extra"), Val: scU32(0)})),
			"must hold exactly"},
	}
	for i, f := range valid() {
		key := string(*f.Key.Sym)
		tests["missing "+key] = testCase{makeMapB64(t, slices.Delete(valid(), i, i+1)), "must hold exactly"}
		wrongType := valid()
		wrongType[i].Val = scSym(key)
		tests["wrong type "+key] = testCase{makeMapB64(t, wrongType), "must hold exactly"}
	}

	w := &watcher{}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			value, err := base64.StdEncoding.DecodeString(tc.value)
			require.NoError(t, err)
			_, err = w.parseMessageFromXDR(value)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// makeEvent returns a getEvents entry for a valid message_published event, after applying edit.
func makeEvent(t *testing.T, edit func(e map[string]any)) gjson.Result {
	t.Helper()
	e := map[string]any{
		"type":           "contract",
		"contractId":     testContract,
		"txHash":         fmt.Sprintf("%064x", 1),
		"ledgerClosedAt": "2024-01-01T00:00:00Z",
		"topic":          []string{makeTopicB64(t, "wormhole"), makeTopicB64(t, "message_published")},
		"value":          makeEventValueB64(t, 1, 1, make([]byte, 32), nil, 0),
	}
	edit(e)
	b, err := json.Marshal(e)
	require.NoError(t, err)
	return gjson.ParseBytes(b)
}

func TestMessageFromEvent_Rejects(t *testing.T) {
	w := newTestWatcher("", 128, nil)
	require.NotNil(t, w.messageFromEvent(makeEvent(t, func(map[string]any) {}), false))

	str := stellarxdr.ScString("message_published")
	strTopic, err := stellarxdr.MarshalBase64(stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvString, Str: &str})
	require.NoError(t, err)
	topics := func(names ...string) []string {
		var b64s []string
		for _, n := range names {
			b64s = append(b64s, makeTopicB64(t, n))
		}
		return b64s
	}

	tests := map[string]func(e map[string]any){
		"system event":         func(e map[string]any) { e["type"] = "system" },
		"other contract":       func(e map[string]any) { e["contractId"] = "other" },
		"three topics":         func(e map[string]any) { e["topic"] = topics("wormhole", "message_published", "x") },
		"wrong first topic":    func(e map[string]any) { e["topic"] = topics("wormhole_core", "message_published") },
		"message_published_v2": func(e map[string]any) { e["topic"] = topics("wormhole", "message_published_v2") },
		"string topic":         func(e map[string]any) { e["topic"] = []string{makeTopicB64(t, "wormhole"), strTopic} },
		"31-byte txHash":       func(e map[string]any) { e["txHash"] = fmt.Sprintf("%062x", 1) },
		"pre-epoch closedAt":   func(e map[string]any) { e["ledgerClosedAt"] = "1969-12-31T23:59:59Z" },
		"epoch closedAt":       func(e map[string]any) { e["ledgerClosedAt"] = "1970-01-01T00:00:00Z" },
		"post-2106 closedAt":   func(e map[string]any) { e["ledgerClosedAt"] = "2106-02-07T06:28:16Z" },
	}
	for name, edit := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, w.messageFromEvent(makeEvent(t, edit), false))
		})
	}
}

func TestMessageFromEvent_CountsParseFailures(t *testing.T) {
	failures := func() float64 {
		m := &dto.Metric{}
		require.NoError(t, stellarMessageParseFailures.WithLabelValues(testNetworkID).Write(m))
		return m.GetCounter().GetValue()
	}
	before := failures()

	w := newTestWatcher("", 128, nil)
	w.messageFromEvent(makeEvent(t, func(e map[string]any) { e["contractId"] = "other" }), false)
	w.messageFromEvent(makeEvent(t, func(e map[string]any) { e["txHash"] = fmt.Sprintf("%062x", 1) }), false)
	w.messageFromEvent(makeEvent(t, func(e map[string]any) {
		e["topic"] = []string{makeTopicB64(t, "wormhole_core"), makeTopicB64(t, "message_published")}
	}), false)
	assert.Equal(t, before+2, failures(), "every message_published event from the core contract is counted")
}

// ---------------------------------------------------------------------------
// handleReobservationRequest tests
// ---------------------------------------------------------------------------

func TestHandleReobservation_Success(t *testing.T) {
	emitter := make([]byte, 32)
	emitter[0] = 0xFF
	payload := []byte("reobs payload")
	txHash := "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd"
	createdAt := int64(1718445600) // 2024-06-15T10:00:00Z, which the watcher must ignore
	closedAt := "2024-06-15T14:00:00Z"
	expectedTS, _ := time.Parse(time.RFC3339, closedAt)

	mock := newMockRPC(300)
	mock.transactions[txHash] = mockTx{status: "SUCCESS", ledger: 200, createdAt: createdAt}
	mock.events = []mockEvent{
		{
			id:             "0000000200-0000000001",
			ledger:         200,
			ledgerClosedAt: closedAt,
			contractID:     testContract,
			txHash:         txHash,
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, 99, 77, emitter, payload, 1),
		},
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	count, err := w.handleReobservationRequest(context.Background(), txHash, srv.URL, w.httpClient)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)

	select {
	case mp := <-msgC:
		assert.True(t, mp.IsReobservation)
		assert.Equal(t, uint64(77), mp.Sequence)
		assert.Equal(t, uint32(99), mp.Nonce)
		assert.Equal(t, uint8(1), mp.ConsistencyLevel)
		assert.Equal(t, payload, mp.Payload)
		assert.Equal(t, vaa.ChainIDStellar, mp.EmitterChain)
		assert.Equal(t, mustDecodeHex(txHash), mp.TxID)
		assert.Equal(t, expectedTS, mp.Timestamp,
			"timestamp must come from ledgerClosedAt, like the poll path")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reobserved message")
	}
}

func TestHandleReobservation_TargetBeyondFirstPage(t *testing.T) {
	// The target tx is on the second page of its ledger. Paging must stop at the first page
	// past that ledger.
	mock := newMockRPC(300)
	mock.events = []mockEvent{
		makeMsgEvent(t, 1, 200),
		makeMsgEvent(t, 2, 200),
		makeMsgEvent(t, 3, 200),
		makeMsgEvent(t, 4, 201),
		makeMsgEvent(t, 5, 201),
	}
	target := mock.events[2].txHash
	mock.transactions[target] = mockTx{status: "SUCCESS", ledger: 200}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 2, msgC)

	count, err := w.handleReobservationRequest(context.Background(), target, srv.URL, w.httpClient)
	require.NoError(t, err)
	require.Equal(t, uint32(1), count)
	assert.Equal(t, uint64(3), (<-msgC).Sequence)
	assert.Equal(t, 2, mock.methodCount("getEvents"), "paging must stop at the first page past the tx ledger")
}

func TestHandleReobservation_NotFound(t *testing.T) {
	txHash := "1234123412341234123412341234123412341234123412341234123412341234"

	mock := newMockRPC(300)
	// No entry in mock.transactions → will return NOT_FOUND.

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	count, err := w.handleReobservationRequest(context.Background(), txHash, srv.URL, w.httpClient)
	assert.Error(t, err, "NOT_FOUND must return an error")
	assert.Contains(t, err.Error(), "not found")
	assert.Equal(t, uint32(0), count)
}

func TestHandleReobservation_NoMatchingEvents(t *testing.T) {
	// getTransaction succeeds but the events at that ledger belong to a different txHash.
	ourTxHash := "aaaa000000000000000000000000000000000000000000000000000000000000"
	otherTxHash := "bbbb000000000000000000000000000000000000000000000000000000000000"
	emitter := make([]byte, 32)
	emitter[0] = 0x01

	mock := newMockRPC(300)
	mock.transactions[ourTxHash] = mockTx{status: "SUCCESS", ledger: 200}
	mock.events = []mockEvent{
		{
			id:             "0000000200-0000000001",
			ledger:         200,
			ledgerClosedAt: "2024-06-15T14:00:00Z",
			contractID:     testContract,
			txHash:         otherTxHash, // different tx!
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, 1, 1, emitter, []byte("pay"), 0),
		},
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC)

	count, err := w.handleReobservationRequest(context.Background(), ourTxHash, srv.URL, w.httpClient)
	require.NoError(t, err, "unmatched txHash is not an error, just zero results")
	assert.Equal(t, uint32(0), count)
}

// ---------------------------------------------------------------------------
// Reobserver interface test
// ---------------------------------------------------------------------------

func TestReobserve_UsesCustomEndpoint(t *testing.T) {
	// Verifies that Reobserve() sends requests to the custom endpoint URL,
	// not to the watcher's primary RPC URL.
	emitter := make([]byte, 32)
	emitter[0] = 0xDE
	txHash := "dead000000000000000000000000000000000000000000000000000000000000"

	primaryMock := newMockRPC(100) // primary server: no transactions
	customMock := newMockRPC(300)  // custom server: has the transaction
	customMock.transactions[txHash] = mockTx{status: "SUCCESS", ledger: 200}
	customMock.events = []mockEvent{
		{
			id:             "0000000200-0000000001",
			ledger:         200,
			ledgerClosedAt: "2023-11-14T22:13:20Z",
			contractID:     testContract,
			txHash:         txHash,
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, 5, 10, emitter, []byte("custom"), 0),
		},
	}

	primarySrv := httptest.NewServer(primaryMock)
	defer primarySrv.Close()
	customSrv := httptest.NewServer(customMock)
	defer customSrv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(primarySrv.URL, 128, msgC)

	txIDBytes := mustDecodeHex(txHash)
	count, err := w.Reobserve(context.Background(), vaa.ChainIDStellar, txIDBytes, customSrv.URL)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), count)

	assert.Equal(t, 0, primaryMock.methodCount("getTransaction"),
		"primary RPC must not be called; Reobserve must use the custom endpoint")
	assert.Equal(t, 1, customMock.methodCount("getTransaction"),
		"custom endpoint must receive the getTransaction call")

	select {
	case mp := <-msgC:
		assert.True(t, mp.IsReobservation)
		assert.Equal(t, uint64(10), mp.Sequence)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reobserved message from custom endpoint")
	}
}

func TestReobservationHandler_FailureAddsErrorCount(t *testing.T) {
	// The mock holds no transactions, so getTransaction returns NOT_FOUND.
	srv := httptest.NewServer(newMockRPC(300))
	defer srv.Close()

	obsvReqC := make(chan *gossipv1.ObservationRequest, 1)
	w := newTestWatcher(srv.URL, 128, nil)
	w.obsvReqC = obsvReqC

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.runReobservationHandler(ctx) }()

	before := p2p.DefaultRegistry.GetErrorCount(vaa.ChainIDStellar)
	obsvReqC <- &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDStellar), TxHash: make([]byte, 32)}
	require.Eventually(t, func() bool {
		return p2p.DefaultRegistry.GetErrorCount(vaa.ChainIDStellar) == before+1
	}, time.Second, 10*time.Millisecond)
}

func TestReobservationHandler_PanicsOnWrongChain(t *testing.T) {
	obsvReqC := make(chan *gossipv1.ObservationRequest, 1)
	obsvReqC <- &gossipv1.ObservationRequest{ChainId: uint32(vaa.ChainIDEthereum)}
	w := newTestWatcher("", 128, nil)
	w.obsvReqC = obsvReqC

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.PanicsWithValue(t, "invalid chain ID", func() { _ = w.runReobservationHandler(ctx) })
}

func TestRPCCall_NullResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
	}))
	defer srv.Close()

	_, err := rpcCall(context.Background(), "getTransaction", nil, srv.URL, srv.Client())
	require.EqualError(t, err, "jsonrpc: missing result")
}
