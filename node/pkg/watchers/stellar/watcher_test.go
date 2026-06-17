package stellar

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	stellarxdr "github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// makeEventValueB64 encodes a Soroban message_published event value as base64 XDR.
// This mirrors the on-chain format that parseMessageFromXDR consumes.
func makeEventValueB64(t *testing.T, nonce uint32, seq uint64, emitter []byte, payload []byte, cl uint32) string {
	t.Helper()
	scm := stellarxdr.ScMap{
		{Key: scSym("nonce"), Val: scU32(nonce)},
		{Key: scSym("sequence"), Val: scU64(seq)},
		{Key: scSym("emitter_address"), Val: scBytesVal(emitter)},
		{Key: scSym("payload"), Val: scBytesVal(payload)},
		{Key: scSym("consistency_level"), Val: scU32(cl)},
	}
	m := &scm
	scVal := stellarxdr.ScVal{Type: stellarxdr.ScValTypeScvMap, Map: &m}
	b64, err := stellarxdr.MarshalBase64(scVal)
	require.NoError(t, err)
	return b64
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
}

func newMockRPC(latestLedger uint64) *mockSorobanRPC {
	return &mockSorobanRPC{
		latestLedger: latestLedger,
		transactions: make(map[string]mockTx),
		methodCounts: make(map[string]int),
	}
}

func (m *mockSorobanRPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

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
	m.mu.Unlock()

	var result any
	switch req.Method {
	case "getLatestLedger":
		m.mu.Lock()
		result = map[string]any{"sequence": m.latestLedger, "protocolVersion": 21}
		m.mu.Unlock()
	case "getEvents":
		result = m.handleGetEvents(req.Params)
	case "getTransaction":
		result = m.handleGetTransaction(req.Params)
	default:
		http.Error(w, fmt.Sprintf("unknown method: %s", req.Method), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
}

func (m *mockSorobanRPC) handleGetEvents(params json.RawMessage) map[string]any {
	var p struct {
		StartLedger uint64 `json:"startLedger"`
		Pagination  struct {
			Limit  int    `json:"limit"`
			Cursor string `json:"cursor"`
		} `json:"pagination"`
	}
	json.Unmarshal(params, &p) //nolint:errcheck

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
			"contractId":     e.contractID,
			"txHash":         e.txHash,
			"topic":          []string{e.topic0B64, e.topic1B64},
			"value":          e.valueB64,
		})
	}

	return map[string]any{
		"events":       eventJSONs,
		"latestLedger": latestLedger,
	}
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

func newTestWatcher(rpcURL string, maxPerPoll int, msgC chan<- *common.MessagePublication, obsvReqC <-chan *gossipv1.ObservationRequest) *watcher {
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
		obsvReqC,
		common.MainNet,
	)
	w.logger = zap.NewNop()
	return w
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
	w := newTestWatcher(srv.URL, 128, msgC, nil)

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
	emitter := make([]byte, 32)
	emitter[0] = 0x01

	closedAt := "2024-01-01T00:00:00Z"
	makeEv := func(idx int, ledger uint64) mockEvent {
		return mockEvent{
			id:             fmt.Sprintf("0000000%03d-0000000001", ledger),
			ledger:         ledger,
			ledgerClosedAt: closedAt,
			contractID:     testContract,
			txHash:         fmt.Sprintf("%064x", idx),
			topic0B64:      makeTopicB64(t, "wormhole"),
			topic1B64:      makeTopicB64(t, "message_published"),
			valueB64:       makeEventValueB64(t, uint32(idx), uint64(idx), emitter, []byte("pay"), 0),
		}
	}

	mock := newMockRPC(300)
	mock.events = []mockEvent{
		makeEv(1, 100),
		makeEv(2, 101),
		makeEv(3, 102),
	}

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 2, msgC, nil) // maxPerPoll = 2

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

func TestPollOnce_SkipsEmptyEmitter(t *testing.T) {
	// An event with an empty emitter_address must be silently dropped.
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
	w := newTestWatcher(srv.URL, 128, msgC, nil)

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
	w := newTestWatcher(srv.URL, 128, msgC, nil)

	advanced, err := w.pollOnce(context.Background(), zap.NewNop())
	require.NoError(t, err)
	assert.True(t, advanced)
	assert.Equal(t, uint64(500), w.nextLedger)

	assert.Equal(t, 0, mock.methodCount("getLatestLedger"),
		"getLatestLedger must not be called separately; latestLedger comes from the getEvents response")
}

// ---------------------------------------------------------------------------
// handleReobservationRequest tests
// ---------------------------------------------------------------------------

func TestHandleReobservation_Success(t *testing.T) {
	emitter := make([]byte, 32)
	emitter[0] = 0xFF
	payload := []byte("reobs payload")
	txHash := "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd"
	createdAt := int64(1718445600) // 2024-06-15T14:00:00Z
	expectedTS := time.Unix(createdAt, 0).UTC()

	mock := newMockRPC(300)
	mock.transactions[txHash] = mockTx{status: "SUCCESS", ledger: 200, createdAt: createdAt}
	mock.events = []mockEvent{
		{
			id:             "0000000200-0000000001",
			ledger:         200,
			ledgerClosedAt: "2024-06-15T14:00:00Z",
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
	w := newTestWatcher(srv.URL, 128, msgC, nil)

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
			"timestamp must come from createdAt returned by getTransaction, not time.Now()")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reobserved message")
	}
}

func TestHandleReobservation_NotFound(t *testing.T) {
	txHash := "1234123412341234123412341234123412341234123412341234123412341234"

	mock := newMockRPC(300)
	// No entry in mock.transactions → will return NOT_FOUND.

	srv := httptest.NewServer(mock)
	defer srv.Close()

	msgC := make(chan *common.MessagePublication, 10)
	w := newTestWatcher(srv.URL, 128, msgC, nil)

	count, err := w.handleReobservationRequest(context.Background(), txHash, srv.URL, w.httpClient)
	assert.Error(t, err, "NOT_FOUND must return an error")
	assert.Contains(t, err.Error(), "not found")
	assert.Equal(t, uint32(0), count)
}

func TestHandleReobservation_NoMatchingEvents(t *testing.T) {
	// getTransaction succeeds but the events at that ledger belong to a different txHash.
	ourTxHash := "aaaa000000000000000000000000000000000000000000000000000000000000"
	otherTxHash := "bbbb000000000000000000000000000000000000000000000000000000000000"
	createdAt := int64(1718445600)

	emitter := make([]byte, 32)
	emitter[0] = 0x01

	mock := newMockRPC(300)
	mock.transactions[ourTxHash] = mockTx{status: "SUCCESS", ledger: 200, createdAt: createdAt}
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
	w := newTestWatcher(srv.URL, 128, msgC, nil)

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
	createdAt := int64(1700000000)

	primaryMock := newMockRPC(100) // primary server: no transactions
	customMock := newMockRPC(300)  // custom server: has the transaction
	customMock.transactions[txHash] = mockTx{status: "SUCCESS", ledger: 200, createdAt: createdAt}
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
	w := newTestWatcher(primarySrv.URL, 128, msgC, nil)

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
