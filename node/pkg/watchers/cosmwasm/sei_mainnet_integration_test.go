//go:build integration

package cosmwasm

import (
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap/zaptest"
)

const (
	seiCoreContract = "sei1gjrrme22cyha4ht2xapn3f08zzw6z3d4uxx6fyy9zd5dyr3yxgzqqncdqn"
	// SEI_LCD must be set to a Sei mainnet LCD endpoint that has the test
	// transactions in its history (most public endpoints prune old txs and
	// return 404 / panic). Tests are skipped when unset.
	seiLCDEnvVar = "SEI_LCD"
)

// fetchSeiTxEvents performs the same LCD query that Watcher.Run does in its
// reobservation handler and returns the parsed events array.
func fetchSeiTxEvents(t *testing.T, lcd, txHash string) []gjson.Result {
	t.Helper()
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(strings.TrimRight(lcd, "/") + "/cosmos/tx/v1beta1/txs/" + txHash)
	require.NoError(t, err, "failed to fetch tx from sei mainnet LCD")
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "non-200 from sei LCD: %s", string(body))

	txJSON := string(body)
	gotHash := gjson.Get(txJSON, "tx_response.txhash").String()
	require.True(t, strings.EqualFold(gotHash, txHash), "tx hash mismatch in response: got %q want %q", gotHash, txHash)

	events := gjson.Get(txJSON, "tx_response.events")
	require.True(t, events.Exists(), "tx response has no events")
	return events.Array()
}

func requireSeiLCD(t *testing.T) string {
	t.Helper()
	lcd := os.Getenv(seiLCDEnvVar)
	if lcd == "" {
		t.Skipf("%s not set; skipping Sei mainnet integration test", seiLCDEnvVar)
	}
	return lcd
}

// newSeiMainnetWatcher builds a Watcher with the same settings the guardian
// uses on Sei mainnet, so the test exercises the constructor's chain-specific
// derivations (notably b64Encoded) instead of hardcoding them.
func newSeiMainnetWatcher(t *testing.T, lcd string) *Watcher {
	t.Helper()
	w := NewWatcher(
		"", // urlWS unused; we only call EventsToMessagePublications
		strings.TrimRight(lcd, "/"),
		seiCoreContract,
		nil, // msgC unused in this path
		nil, // obsvReqC unused in this path
		vaa.ChainIDSei,
		common.MainNet,
	)
	// Sei must be base64-encoded (it is not Injective or Terra2). Catches
	// regressions in the b64Encoded selection in NewWatcher.
	require.True(t, w.b64Encoded, "Sei mainnet watcher must have b64Encoded=true")
	require.Equal(t, "_contract_address", w.contractAddressLogKey, "Sei mainnet watcher must use cosmwasm 1.0 contract address log key")
	return w
}

// TestSeiMainnetMessageParsing confirms the cosmwasm watcher correctly
// observes a real Sei mainnet wormhole core bridge "hello world" tx.
//
// Tx: A0E841E95ECEF63B50D033E03F7B66723F49891CA645B1E0F2B12B74D612E012
//
// Run with:
//
//	SEI_LCD=https://your-sei-lcd \
//	  go test -tags=integration -run TestSeiMainnetMessageParsing ./pkg/watchers/cosmwasm/
func TestSeiMainnetMessageParsing(t *testing.T) {
	lcd := requireSeiLCD(t)

	const (
		txHash          = "A0E841E95ECEF63B50D033E03F7B66723F49891CA645B1E0F2B12B74D612E012"
		expectedSender  = "0000000000000000000000006576f9c768da087790efca8d09a0c4ad81e3729d"
		expectedPayload = "68656c6c6f20776f726c64" // "hello world"
		expectedNonce   = uint32(2604184548)
		expectedSeq     = uint64(3)
		expectedTime    = int64(1778085335)
	)

	logger := zaptest.NewLogger(t)
	w := newSeiMainnetWatcher(t, lcd)
	events := fetchSeiTxEvents(t, lcd, txHash)

	msgs := EventsToMessagePublications(
		w.contract,
		txHash,
		events,
		logger,
		w.chainID,
		w.contractAddressLogKey,
		w.b64Encoded,
	)
	require.Len(t, msgs, 1, "expected exactly one wormhole message publication")
	msg := msgs[0]

	expectedSenderBytes, err := hex.DecodeString(expectedSender)
	require.NoError(t, err)
	var expectedEmitter vaa.Address
	copy(expectedEmitter[:], expectedSenderBytes)

	expectedPayloadBytes, err := hex.DecodeString(expectedPayload)
	require.NoError(t, err)

	expectedTxID, err := hex.DecodeString(txHash)
	require.NoError(t, err)

	assert.Equal(t, vaa.ChainIDSei, msg.EmitterChain, "emitter chain")
	assert.Equal(t, expectedEmitter, msg.EmitterAddress, "emitter address")
	assert.Equal(t, expectedSeq, msg.Sequence, "sequence")
	assert.Equal(t, expectedNonce, msg.Nonce, "nonce")
	assert.Equal(t, expectedPayloadBytes, msg.Payload, "payload")
	assert.Equal(t, "hello world", string(msg.Payload), "payload as string")
	assert.Equal(t, time.Unix(expectedTime, 0), msg.Timestamp, "timestamp")
	assert.Equal(t, uint8(0), msg.ConsistencyLevel, "consistency level (instant finality)")
	assert.False(t, msg.Unreliable, "Unreliable should be false")
	assert.Equal(t, expectedTxID, msg.TxID, "TxID should be the Sei tx hash bytes")

	t.Logf("Successfully parsed Sei mainnet tx %s", txHash)
	t.Logf("  emitter: %s", hex.EncodeToString(msg.EmitterAddress[:]))
	t.Logf("  sequence: %d, nonce: %d", msg.Sequence, msg.Nonce)
	t.Logf("  payload: %q", string(msg.Payload))
}

// TestSeiMainnetReobservationMatchesVAA_B32EFC48 confirms that a reobservation
// of Sei mainnet tx B32EFC48D6E1D2CD0B2F744F3EEAC3D62EFDEBBB1BDFB97B972A2B213E34BE6C
// produces a MessagePublication whose VAA body matches the canonical signed
// VAA from wormholescan:
//
//	https://api.wormholescan.io/v1/signed_vaa/32/86c5fd957e2db8389553e1728f9c27964b22a8154091ccba54d75f4b10c61f5e/29310
//
// Run with:
//
//	SEI_LCD=https://your-sei-lcd \
//	  go test -tags=integration -run TestSeiMainnetReobservationMatchesVAA_B32EFC48 ./pkg/watchers/cosmwasm/
func TestSeiMainnetReobservationMatchesVAA_B32EFC48(t *testing.T) {
	lcd := requireSeiLCD(t)

	const txHash = "B32EFC48D6E1D2CD0B2F744F3EEAC3D62EFDEBBB1BDFB97B972A2B213E34BE6C"

	// Canonical signed VAA for chain=32 emitter=86c5...1f5e sequence=29310.
	const canonicalVAAB64 = "AQAAAAQNABK6V94bpyu7TLLWzhIyZtTngH4KEAgDRhYgmocGcnDsGh5/IYnoIELygJQQ6CnUqll/WoG9b2aQiCz7+2UeHXkBAUx50zirgozrnoWqDnHQmVwU7l0sPevJBfstDE5Wdu0eMB2Kjayg4gkucvdHX4AXJve8+5+cUP4IPkpHG50xJRYAAg4+McGhS+Lg5ITE7G3qlY9xZ1u3itjzsrR06YuZhlKLObPoo9vH4qum8xFE3xj2ij79Wmoze1iWRtU/CbF3U+MABPn6f8DWMeB/Tv7qKqBv0f5ETQMvoU/rSpb2UQDtR6k4P0UlibSGnz+CMfMV/aOKLuWO1ugmIWlZMBhsWta2cC4ABtLze0YLh9Zro6uxKEpSsUi7p+mGJOUNtW5MrfQME4ZGT8OGHbrYVJ0dN2Iwh7h51Rhol8TmlYds34BLmPvXtToBB9PT9iGXW7UW4coICAseSV5fos6vvdGFvpl+9BDsiyBCFiZ/GLzb0KX2kDF1n5HKZ8BzM09cQS6MiAcgE9sej84ACOeDDYF89UP0l6hCqIJbcNnnnZK0tgmPL2IfTSlAixXBP8o9H6bH6IegmE5Oa2b9sZVho8wslqgWKwPAVWh0jlgACtWPokIZ5u5DhbB+yV6acPXLNjcAU4kI9nMqHyeDrMjtEXMx2LO1UCKPXORcKufV51op0PALfdM177/ucP2VvdgADHJEOgkw++gqyg8VXzLmEEzGl+uL+ocq2EjLrXDEmYTRUHmueHaxyO0xJKBOExWr59wU9ihvnglshGnzP3JjrmsADlVKTUZzQ8w/oo+kCM3ZzN8TkbtzPNeTzfRpD3lX0mUDAmBIe8uaFcNDRZN/yhgfE/M5B8Du3SGu3xjGNjuR82QAD+SPtxwCUY9Fw2Xt9q4PtoT+JNzBt+IP3wZLeeFBcFVbCkNpbOPlqBIWIkKPJlXUtm6uEDCXtoLJ+tAcv7PTwKsBEWtiU7x4EzbFaYPRkgIn6w8wyFUSnhFZxSuqZKnn9sqXUAx20ksw8mx/76Skoxkq1lUepqfQHBLF79hvtAMa5kQBEu9AhQ3Z69xIJBM+HNjBw6dIkVb58iJovp9E5n6v3qd3TXhsUyh9gTp8+J0G+3yoXhZPo8txQmm1O22nRpqTLfAAaZdivgAAAAAAIIbF/ZV+Lbg4lVPhco+cJ5ZLIqgVQJHMulTXX0sQxh9eAAAAAAAAcn4AAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAokpAAAAAAAAAAAAAAAAImD6xeVUKnc6pE+8/t98GTvCxZkAApCeO2rE6WlWtVgAQp6i0hT/lNafzi/hUgqQP8luK0p5ABUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="
	canonicalVAA, err := base64.StdEncoding.DecodeString(canonicalVAAB64)
	require.NoError(t, err)

	parsedCanonical, err := vaa.Unmarshal(canonicalVAA)
	require.NoError(t, err, "failed to unmarshal canonical VAA")

	logger := zaptest.NewLogger(t)
	w := newSeiMainnetWatcher(t, lcd)
	events := fetchSeiTxEvents(t, lcd, txHash)
	msgs := EventsToMessagePublications(
		w.contract,
		txHash,
		events,
		logger,
		w.chainID,
		w.contractAddressLogKey,
		w.b64Encoded,
	)
	require.Len(t, msgs, 1, "expected exactly one wormhole message publication")
	msg := msgs[0]

	// Build a VAA from the reobservation using guardian-set-index=0 (matches
	// what Watcher.Run produces). We compare body fields, not signatures —
	// signatures are added later by guardian signing, not by parsing.
	observedVAA := msg.CreateVAA(parsedCanonical.GuardianSetIndex)

	// Field-by-field equality with the canonical VAA's body.
	assert.Equal(t, parsedCanonical.Timestamp.Unix(), observedVAA.Timestamp.Unix(), "timestamp")
	assert.Equal(t, parsedCanonical.Nonce, observedVAA.Nonce, "nonce")
	assert.Equal(t, parsedCanonical.EmitterChain, observedVAA.EmitterChain, "emitter chain")
	assert.Equal(t, parsedCanonical.EmitterAddress, observedVAA.EmitterAddress, "emitter address")
	assert.Equal(t, parsedCanonical.Sequence, observedVAA.Sequence, "sequence")
	assert.Equal(t, parsedCanonical.ConsistencyLevel, observedVAA.ConsistencyLevel, "consistency level")
	assert.Equal(t, parsedCanonical.Payload, observedVAA.Payload, "payload")

	// Byte-level body equality is the strongest check: identical body bytes
	// produce identical signing digests, so any guardian quorum signing the
	// reobservation produces the same VAA body as the canonical one.
	canonicalBodyDigest := parsedCanonical.SigningDigest()
	observedBodyDigest := observedVAA.SigningDigest()
	assert.Equal(t,
		hex.EncodeToString(canonicalBodyDigest.Bytes()),
		hex.EncodeToString(observedBodyDigest.Bytes()),
		"signing digest must match canonical VAA",
	)

	// Sanity: chain id, emitter, sequence match the wormholescan URL.
	assert.Equal(t, vaa.ChainIDSei, msg.EmitterChain)
	assert.Equal(t, "86c5fd957e2db8389553e1728f9c27964b22a8154091ccba54d75f4b10c61f5e", hex.EncodeToString(msg.EmitterAddress[:]))
	assert.Equal(t, uint64(29310), msg.Sequence)

	t.Logf("Reobservation of %s matches canonical VAA digest %s", txHash, hex.EncodeToString(observedBodyDigest.Bytes()))
}

// TestSeiMainnetReobservation_8803D593 confirms that Sei mainnet tx
// 8803D593F90F6013BF09946008525E61D8FC7A6B4F3625E91D18969E9A056454 emits a
// well-formed wormhole message that the cosmwasm watcher will publish.
//
// Run with:
//
//	SEI_LCD=https://your-sei-lcd \
//	  go test -tags=integration -run TestSeiMainnetReobservation_8803D593 ./pkg/watchers/cosmwasm/
func TestSeiMainnetReobservation_8803D593(t *testing.T) {
	lcd := requireSeiLCD(t)

	const txHash = "8803D593F90F6013BF09946008525E61D8FC7A6B4F3625E91D18969E9A056454"

	logger := zaptest.NewLogger(t)
	w := newSeiMainnetWatcher(t, lcd)
	events := fetchSeiTxEvents(t, lcd, txHash)
	msgs := EventsToMessagePublications(
		w.contract,
		txHash,
		events,
		logger,
		w.chainID,
		w.contractAddressLogKey,
		w.b64Encoded,
	)
	require.NotEmpty(t, msgs, "expected at least one wormhole message publication")

	expectedTxID, err := hex.DecodeString(txHash)
	require.NoError(t, err)

	for i, msg := range msgs {
		assert.Equal(t, vaa.ChainIDSei, msg.EmitterChain, "msg[%d] emitter chain", i)
		assert.Equal(t, expectedTxID, msg.TxID, "msg[%d] TxID should be the Sei tx hash bytes", i)
		assert.NotZero(t, msg.Timestamp.Unix(), "msg[%d] timestamp", i)
		assert.NotEmpty(t, msg.Payload, "msg[%d] payload", i)
		assert.NotEqual(t, vaa.Address{}, msg.EmitterAddress, "msg[%d] emitter address", i)

		// CreateVAA must succeed and round-trip — confirms the publication is
		// shaped correctly for guardian signing.
		v := msg.CreateVAA(0)
		require.NotNil(t, v)
		raw, err := v.Marshal()
		require.NoError(t, err, "msg[%d] VAA must marshal", i)
		require.NotEmpty(t, raw, "msg[%d] marshalled VAA bytes", i)

		t.Logf("msg[%d]: emitter=%s sequence=%d nonce=%d payload_len=%d",
			i,
			hex.EncodeToString(msg.EmitterAddress[:]),
			msg.Sequence,
			msg.Nonce,
			len(msg.Payload),
		)
	}
}
