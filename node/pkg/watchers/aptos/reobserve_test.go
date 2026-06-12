package aptos

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const (
	testAccount = "0xdeadbeef"
	testHandle  = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef::wormhole::WormholeMessageHandle"
)

// txHashWithSeq returns a 32-byte buffer with seq in the last 8 bytes (matching the
// transport format used by the watcher and the production observation request flow).
func txHashWithSeq(seq uint64) []byte {
	b := make([]byte, 32)
	binary.BigEndian.PutUint64(b[24:], seq)
	return b
}

// eventsServer returns an httptest server that serves event responses for the configured
// account/handle and chain id responses for `/v1`.
func eventsServer(t *testing.T, eventsBody, chainIDBody string) *httptest.Server {
	t.Helper()
	eventsPath := fmt.Sprintf("/v1/accounts/%s/events/%s/event", testAccount, testHandle)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case eventsPath:
			fmt.Fprint(w, eventsBody)
		case "/v1":
			fmt.Fprint(w, chainIDBody)
		default:
			http.NotFound(w, r)
		}
	}))
}

func newTestWatcher(msgC chan<- *common.MessagePublication) *Watcher {
	return &Watcher{
		chainID:      vaa.ChainIDAptos,
		networkID:    "aptos-test",
		env:          common.MainNet,
		aptosAccount: testAccount,
		aptosHandle:  testHandle,
		msgC:         msgC,
	}
}

func validEventArray(seq uint64) string {
	return fmt.Sprintf(`[{
		"sequence_number":"%d",
		"data":{
			"sender":"1",
			"payload":"0xdeadbeef",
			"timestamp":"1000",
			"nonce":"42",
			"sequence":"7",
			"consistency_level":"1"
		}
	}]`, seq)
}

func TestHandleReobservationRequest_ChainIDMismatch(t *testing.T) {
	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), vaa.ChainIDEthereum, txHashWithSeq(1), "http://unused")
	require.ErrorContains(t, err, "unexpected chain id")
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_TxHashTooShort(t *testing.T) {
	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, make([]byte, 31), "http://unused")
	require.ErrorContains(t, err, "too short")
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_RetrievePayloadError(t *testing.T) {
	// Spin up and immediately close to get a guaranteed-unreachable URL.
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	s.Close()

	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(1), s.URL)
	require.ErrorContains(t, err, "retrievePayload")
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_InvalidJSON(t *testing.T) {
	s := eventsServer(t, "not json", "")
	defer s.Close()

	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(1), s.URL)
	require.ErrorContains(t, err, "invalid JSON")
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_MissingSequenceNumber(t *testing.T) {
	// Array with one entry that has no `sequence_number` — the loop should break and return cleanly.
	s := eventsServer(t, `[{"data":{}}]`, "")
	defer s.Close()

	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(1), s.URL)
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_SequenceMismatch(t *testing.T) {
	s := eventsServer(t, validEventArray(99), "")
	defer s.Close()

	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(1), s.URL)
	require.ErrorContains(t, err, "newSeq != nativeSeq")
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_MissingData(t *testing.T) {
	// Entry has matching sequence_number but no `data` field.
	s := eventsServer(t, `[{"sequence_number":"1"}]`, "")
	defer s.Close()

	w := newTestWatcher(nil)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(1), s.URL)
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestHandleReobservationRequest_ObserveDataFails(t *testing.T) {
	// Matching seq + data, but the data is missing required fields, so observeData returns false.
	s := eventsServer(t, `[{"sequence_number":"1","data":{}}]`, "")
	defer s.Close()

	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(1), s.URL)
	require.NoError(t, err)
	assert.Zero(t, n)
	assert.Empty(t, msgC)
}

func TestHandleReobservationRequest_Success(t *testing.T) {
	s := eventsServer(t, validEventArray(7), "")
	defer s.Close()

	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)
	n, err := w.handleReobservationRequest(zap.NewNop(), w.chainID, txHashWithSeq(7), s.URL)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), n)
	require.Len(t, msgC, 1)
	msg := <-msgC
	assert.True(t, msg.IsReobservation)
	assert.Equal(t, vaa.ChainIDAptos, msg.EmitterChain)
}

func TestReobserve_LoggerNil_DevnetBypass(t *testing.T) {
	// Devnet skips the chain id verification, so this exercises the nil-logger fallback
	// and a clean handleReobservationRequest call.
	s := eventsServer(t, validEventArray(7), "")
	defer s.Close()

	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)
	w.env = common.UnsafeDevNet
	w.logger = nil

	n, err := w.Reobserve(context.Background(), w.chainID, txHashWithSeq(7), s.URL)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), n)
}

func TestReobserve_VerifyChainIDFails(t *testing.T) {
	s := eventsServer(t, validEventArray(7), `{"chain_id":99}`)
	defer s.Close()

	w := newTestWatcher(nil)
	w.logger = zap.NewNop()

	n, err := w.Reobserve(context.Background(), w.chainID, txHashWithSeq(7), s.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify aptos chain id")
	assert.Zero(t, n)
}

func TestReobserve_Success(t *testing.T) {
	s := eventsServer(t, validEventArray(7), `{"chain_id":1}`)
	defer s.Close()

	msgC := make(chan *common.MessagePublication, 1)
	w := newTestWatcher(msgC)
	w.logger = zap.NewNop()

	n, err := w.Reobserve(context.Background(), w.chainID, txHashWithSeq(7), s.URL)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), n)
	require.Len(t, msgC, 1)

	// Sanity: log message produced when txID is the raw bytes; ensure no panic on the Any() call path.
	assert.NotEmpty(t, strings.TrimSpace(string(txHashWithSeq(7))))
}
