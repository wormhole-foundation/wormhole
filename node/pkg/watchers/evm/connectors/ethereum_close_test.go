package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/zap/zaptest"
)

// idleWSServer accepts WebSocket upgrades and holds them open, which is enough
// for go-ethereum's rpc.Client to spawn its dispatch/read/write goroutines.
func idleWSServer(t *testing.T) string {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			if _, _, err := c.NextReader(); err != nil {
				_ = c.Close()
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// TestEthereumBaseConnector_CloseReleasesGoroutines asserts that dialing and
// Close()-ing connectors leaks no goroutines. Without Close(), each dial
// strands the rpc.Client's dispatch/read/write goroutines.
func TestEthereumBaseConnector_CloseReleasesGoroutines(t *testing.T) {
	url := idleWSServer(t)
	logger := zaptest.NewLogger(t)
	addr := ethCommon.HexToAddress("0x0")

	// Warm up shared transport singletons before the baseline so IgnoreCurrent
	// does not flag them; the check then sees only goroutines from the loop.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	warm, err := NewEthereumBaseConnector(ctx, "warmup", url, addr, nil, logger)
	require.NoError(t, err)
	warm.Close()

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	for i := 0; i < 25; i++ {
		dctx, dcancel := context.WithTimeout(context.Background(), 5*time.Second)
		c, err := NewEthereumBaseConnector(dctx, "test", url, addr, nil, logger)
		dcancel()
		require.NoError(t, err, "dial %d", i)
		c.Close()
	}
}
