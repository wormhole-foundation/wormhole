//go:build integration

package sui

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/suiclient"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// TestLiveSuiWatcher runs the real Sui watcher against a live gRPC endpoint (mainnet by
// default) and, for every observed Wormhole message, cross-checks the gRPC/BCS-decoded fields
// against the node's own JSON-RPC `parsedJson` rendering of the same event. It also issues a
// real re-observation request for each observed transaction and asserts that the re-observed
// message has the same VAA hash as the original observation — VAAHash is what the guardian uses
// to identify a message, so a reobservation that hashed differently would never reach consensus
// with the original. A field mismatch or hash mismatch fails the test. It is only compiled with
// the `integration` build tag.
//
// Run it (streams logs; Ctrl-C to stop early):
//
//	go test -tags integration ./pkg/watchers/sui -run TestLiveSuiWatcher -v -timeout 0
//
// Optional environment variables:
//
//	SUI_WATCHER_RPC         gRPC endpoint host:port      (default fullnode.mainnet.sui.io:443)
//	SUI_WATCHER_JSONRPC     JSON-RPC URL for cross-check (default https://fullnode.mainnet.sui.io)
//	SUI_WATCHER_EVENT_TYPE  core bridge WormholeMessage  (default mainnet core bridge type)
//	SUI_WATCHER_SECONDS     how long to observe          (default 120)
//	SUI_WATCHER_TXVERIFIER  set to any value to enable the transfer verifier
func TestLiveSuiWatcher(t *testing.T) {
	rpc := liveEnv("SUI_WATCHER_RPC", suiclient.SuiRPCMainnet)
	jsonRPC := liveEnv("SUI_WATCHER_JSONRPC", "https://fullnode.mainnet.sui.io")
	eventType := liveEnv("SUI_WATCHER_EVENT_TYPE", "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage")

	seconds := 120
	if s := os.Getenv("SUI_WATCHER_SECONDS"); s != "" {
		parsed, err := strconv.Atoi(s)
		require.NoError(t, err)
		seconds = parsed
	}
	txVerifierEnabled := os.Getenv("SUI_WATCHER_TXVERIFIER") != ""

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	logger.Info("starting live Sui watcher",
		zap.String("rpc", rpc),
		zap.String("jsonRPC", jsonRPC),
		zap.String("eventType", eventType),
		zap.Int("seconds", seconds),
		zap.Bool("txVerifierEnabled", txVerifierEnabled),
	)

	// Buffered so the watcher never blocks publishing while we verify; we drain it below.
	msgChan := make(chan *common.MessagePublication, 100)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 10)

	watcher, err := NewWatcher(rpc, eventType, false, msgChan, obsvReqC, common.MainNet, txVerifierEnabled)
	require.NoError(t, err)

	rootCtx, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()

	observed := 0
	verified := 0
	reobserved := 0
	reobsVerified := 0
	// VAA hash of each original (non-reobservation) message, keyed by its message ID. A
	// re-observed message is looked up here and its hash compared against the original.
	originalHashes := make(map[string]string)
	var wg sync.WaitGroup
	wg.Add(1)

	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		// Drain the message channel; log and cross-check every observation. Only this goroutine
		// touches originalHashes / the counters, so no locking is needed.
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-msgChan:
					if msg.IsReobservation {
						reobserved++
						origHash, ok := originalHashes[msg.MessageIDString()]
						if !ok {
							logger.Warn("reobs: no original observation recorded for re-observed message (skipping)",
								zap.String("msgID", msg.MessageIDString()),
							)
							continue
						}
						if origHash != msg.VAAHash() {
							t.Errorf("REOBSERVATION HASH MISMATCH msgID=%s: original=%s reobservation=%s",
								msg.MessageIDString(), origHash, msg.VAAHash())
							continue
						}
						reobsVerified++
						logger.Info("VERIFIED reobservation hash matches original",
							zap.String("msgID", msg.MessageIDString()),
							zap.String("vaaHash", msg.VAAHash()),
						)
						continue
					}

					observed++
					logger.Info("OBSERVED WORMHOLE MESSAGE",
						append([]zap.Field{
							zap.Int("n", observed),
							zap.String("txIDHex", hex.EncodeToString(msg.TxID)),
							zap.String("verificationState", msg.VerificationState().String()),
						}, msg.ZapFields()...)...,
					)
					if verifyMessageAgainstJSONRPC(ctx, t, logger, jsonRPC, eventType, msg) {
						verified++
					}

					// Record the original hash and request a real re-observation of this transaction
					// so we can confirm the reobservation path produces an identical message hash.
					originalHashes[msg.MessageIDString()] = msg.VAAHash()
					req := &gossipv1.ObservationRequest{
						ChainId: uint32(vaa.ChainIDSui),
						TxHash:  msg.TxID,
					}
					select {
					case obsvReqC <- req:
					default:
						logger.Warn("reobs: observation request channel full, skipping reobservation",
							zap.String("msgID", msg.MessageIDString()),
						)
					}
				}
			}
		}()

		if rerr := supervisor.Run(ctx, "sui", watcher.Run); rerr != nil {
			return rerr
		}

		<-ctx.Done()
		return nil
	}, supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	// Wait for the drain goroutine to finish any in-flight verification before the test returns,
	// so it never calls t.Errorf after the test function has returned.
	wg.Wait()
	logger.Info("live Sui watcher finished",
		zap.Int("messagesObserved", observed),
		zap.Int("messagesVerified", verified),
		zap.Int("reobservations", reobserved),
		zap.Int("reobservationsVerified", reobsVerified),
	)
}

// suiParsedJson mirrors the `parsedJson` of a WormholeMessage event as rendered by the Sui
// JSON-RPC node — the independent decode the watcher's gRPC/BCS result is checked against.
type suiParsedJson struct {
	ConsistencyLevel uint8  `json:"consistency_level"`
	Nonce            uint32 `json:"nonce"`
	Payload          []byte `json:"payload"`
	Sender           string `json:"sender"`
	Sequence         uint64 `json:"sequence,string"`
	Timestamp        int64  `json:"timestamp,string"`
}

// fetchParsedJSONEvent queries `sui_getEvents` for `digest` and returns the parsedJson of the
// WormholeMessage event with the given sequence. It returns (nil, nil) when the matching event
// is not present yet (e.g. the queried node has not indexed the just-produced transaction), and
// (nil, err) on a transport/decode error — both of which the caller treats as retryable.
func fetchParsedJSONEvent(ctx context.Context, jsonRPC, digest, eventType string, sequence uint64) (*suiParsedJson, error) {
	reqBody := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"sui_getEvents","params":["%s"]}`, digest)
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, jsonRPC, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Result []struct {
			Type       string          `json:"type"`
			ParsedJson json.RawMessage `json:"parsedJson"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal JSON-RPC response: %w (body=%s)", err, string(body))
	}

	// Only decode the parsedJson of WormholeMessage events. A transaction can bundle unrelated
	// events (e.g. a CCTP DepositForBurn, whose `nonce` is a quoted string) whose fields would
	// otherwise collide with the strongly-typed suiParsedJson and fail the whole decode.
	for i := range parsed.Result {
		if parsed.Result[i].Type != eventType {
			continue
		}
		var pj suiParsedJson
		if err := json.Unmarshal(parsed.Result[i].ParsedJson, &pj); err != nil {
			return nil, fmt.Errorf("unmarshal WormholeMessage parsedJson: %w", err)
		}
		if pj.Sequence == sequence {
			return &pj, nil
		}
	}
	return nil, nil
}

// verifyMessageAgainstJSONRPC fetches the events for the message's transaction over JSON-RPC and
// asserts that the node's independently-decoded `parsedJson` fields match the gRPC/BCS-decoded
// MessagePublication. It returns true only if a matching event was found and every field matched.
// Lookup failures are logged and skipped (return false); an actual field mismatch fails the test
// via t.Errorf.
func verifyMessageAgainstJSONRPC(ctx context.Context, t *testing.T, logger *zap.Logger, jsonRPC, eventType string, msg *common.MessagePublication) bool {
	digest := base58.Encode(msg.TxID)

	pj, err := fetchParsedJSONEvent(ctx, jsonRPC, digest, eventType, msg.Sequence)
	if err != nil {
		logger.Warn("verify: JSON-RPC lookup failed (skipping)", zap.String("digest", digest), zap.Error(err))
		return false
	}
	if pj == nil {
		logger.Warn("verify: no matching WormholeMessage event found in JSON-RPC response (skipping)",
			zap.String("digest", digest),
			zap.Uint64("sequence", msg.Sequence),
		)
		return false
	}

	ok := true
	mismatch := func(field string, expected, actual any) {
		ok = false
		t.Errorf("FIELD MISMATCH [%s] tx=%s seq=%d: jsonrpc=%v watcher=%v", field, digest, msg.Sequence, expected, actual)
	}

	// Sequence already matched by fetchParsedJSONEvent.
	if pj.Nonce != msg.Nonce {
		mismatch("nonce", pj.Nonce, msg.Nonce)
	}
	if pj.ConsistencyLevel != msg.ConsistencyLevel {
		mismatch("consistency_level", pj.ConsistencyLevel, msg.ConsistencyLevel)
	}
	// parsedJson sender carries a 0x prefix; vaa.Address.String() does not.
	if strings.TrimPrefix(pj.Sender, "0x") != msg.EmitterAddress.String() {
		mismatch("sender/emitter", pj.Sender, msg.EmitterAddress.String())
	}
	if !bytes.Equal(pj.Payload, msg.Payload) {
		mismatch("payload", hex.EncodeToString(pj.Payload), hex.EncodeToString(msg.Payload))
	}
	// parsedJson timestamp is in seconds; the watcher stores time.Unix(seconds, 0).
	if !time.Unix(pj.Timestamp, 0).Equal(msg.Timestamp) {
		mismatch("timestamp", pj.Timestamp, msg.Timestamp.Unix())
	}

	if ok {
		logger.Info("VERIFIED against JSON-RPC parsedJson",
			zap.String("digest", digest),
			zap.Uint64("sequence", msg.Sequence),
			zap.Uint32("nonce", msg.Nonce),
			zap.Uint8("consistencyLevel", msg.ConsistencyLevel),
			zap.String("emitter", msg.EmitterAddress.String()),
			zap.Int("payloadLen", len(msg.Payload)),
		)
	}
	return ok
}

func liveEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
