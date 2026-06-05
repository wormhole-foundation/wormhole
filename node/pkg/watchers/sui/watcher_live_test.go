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
	"go.uber.org/zap"
)

// TestLiveSuiWatcher runs the real Sui watcher against a live gRPC endpoint (mainnet by
// default) and, for every observed Wormhole message, cross-checks the gRPC/BCS-decoded fields
// against the node's own JSON-RPC `parsedJson` rendering of the same event. A field mismatch
// fails the test. It is skipped unless SUI_WATCHER_LIVE is set.
//
// Run it (streams logs; Ctrl-C to stop early):
//
//	SUI_WATCHER_LIVE=1 go test ./pkg/watchers/sui -run TestLiveSuiWatcher -v -timeout 0
//
// Optional environment variables:
//
//	SUI_WATCHER_RPC         gRPC endpoint host:port      (default fullnode.mainnet.sui.io:443)
//	SUI_WATCHER_JSONRPC     JSON-RPC URL for cross-check (default https://fullnode.mainnet.sui.io)
//	SUI_WATCHER_EVENT_TYPE  core bridge WormholeMessage  (default mainnet core bridge type)
//	SUI_WATCHER_SECONDS     how long to observe          (default 120)
//	SUI_WATCHER_TXVERIFIER  set to any value to enable the transfer verifier
func TestLiveSuiWatcher(t *testing.T) {
	if os.Getenv("SUI_WATCHER_LIVE") == "" {
		t.Skip("set SUI_WATCHER_LIVE=1 to run the live Sui watcher test")
	}

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
	var wg sync.WaitGroup
	wg.Add(1)

	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		// Drain the message channel; log and cross-check every observation.
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-msgChan:
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
	logger.Info("live Sui watcher finished", zap.Int("messagesObserved", observed), zap.Int("messagesVerified", verified))
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
