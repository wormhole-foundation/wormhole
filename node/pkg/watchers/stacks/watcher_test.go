package stacks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReobservation(t *testing.T) {
	const (
		testTxId           = "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd"
		testIndexBlockHash = "1122334455667788112233445566778811223344556677881122334455667788"
	)

	// blockHeight intentionally >> burnBlockHeight to simulate mainnet,
	// where the old bug (comparing block_height instead of burn_block_height) would fail.
	blockHeight := uint64(5000)

	makeTxResponse := func() StacksV3TransactionResponse {
		return StacksV3TransactionResponse{
			IndexBlockHash: testIndexBlockHash,
			Result:         "(ok true)",
			BlockHeight:    &blockHeight,
			IsCanonical:    true,
		}
	}

	makeReplayResponse := func(burnBlockHeight uint64) StacksV3TenureBlockReplayResponse {
		return StacksV3TenureBlockReplayResponse{
			BlockId:         "block123",
			BlockHash:       "hash123",
			BlockHeight:     blockHeight,
			BurnBlockHeight: burnBlockHeight,
			ValidMerkleRoot: true,
			Timestamp:       1700000000,
			Transactions: []StacksV3TenureBlockTransaction{
				{
					TxId:      testTxId,
					ResultHex: "0x0703",
					Events:    []StacksEvent{},
				},
			},
		}
	}

	t.Run("succeeds when burn height <= stable height", func(t *testing.T) {
		replayResp := makeReplayResponse(100)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/v3/transaction/"):
				require.NoError(t, json.NewEncoder(w).Encode(makeTxResponse()))
			case strings.HasPrefix(r.URL.Path, "/v3/blocks/replay/"):
				require.NoError(t, json.NewEncoder(w).Encode(replayResp))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		watcher := &Watcher{
			rpcURL:     server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		watcher.stableBitcoinHeight.Store(100)

		logger := zap.NewNop()
		count, err := watcher.reobserveStacksTransactionByTxId(context.Background(), testTxId, logger)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), count) // 0 wormhole events, but no error
	})

	t.Run("fails when burn height > stable height", func(t *testing.T) {
		replayResp := makeReplayResponse(101)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/v3/transaction/"):
				require.NoError(t, json.NewEncoder(w).Encode(makeTxResponse()))
			case strings.HasPrefix(r.URL.Path, "/v3/blocks/replay/"):
				require.NoError(t, json.NewEncoder(w).Encode(replayResp))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		watcher := &Watcher{
			rpcURL:     server.URL,
			httpClient: &http.Client{Timeout: 5 * time.Second},
		}
		watcher.stableBitcoinHeight.Store(100)

		logger := zap.NewNop()
		_, err := watcher.reobserveStacksTransactionByTxId(context.Background(), testTxId, logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "block burn height")
	})
}
