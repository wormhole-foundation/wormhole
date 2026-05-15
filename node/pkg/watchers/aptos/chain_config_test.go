package aptos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func TestGetChainConfigMap(t *testing.T) {
	m, err := GetChainConfigMap(common.MainNet)
	require.NoError(t, err)
	assert.Equal(t, mainnetChainConfig, m)

	m, err = GetChainConfigMap(common.TestNet)
	require.NoError(t, err)
	assert.Equal(t, testnetChainConfig, m)

	_, err = GetChainConfigMap(common.UnsafeDevNet)
	require.ErrorIs(t, err, ErrInvalidEnv)
}

func TestGetAptosChainID(t *testing.T) {
	tests := []struct {
		name    string
		env     common.Environment
		chainID vaa.ChainID
		want    uint64
		wantErr error
	}{
		{"mainnet aptos", common.MainNet, vaa.ChainIDAptos, 1, nil},
		{"mainnet movement", common.MainNet, vaa.ChainIDMovement, 126, nil},
		{"testnet aptos", common.TestNet, vaa.ChainIDAptos, 2, nil},
		{"testnet movement", common.TestNet, vaa.ChainIDMovement, 250, nil},
		{"unknown chain", common.MainNet, vaa.ChainIDEthereum, 0, ErrNotFound},
		{"invalid env", common.UnsafeDevNet, vaa.ChainIDAptos, 0, ErrInvalidEnv},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetAptosChainID(tc.env, tc.chainID)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// chainIDServer returns an httptest server that responds to `GET /v1` with body.
func chainIDServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1" {
			fmt.Fprint(w, body)
		}
	}))
}

func TestVerifyAptosChainID(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("devnet bypass", func(t *testing.T) {
		w := &Watcher{env: common.UnsafeDevNet, chainID: vaa.ChainIDAptos}
		require.NoError(t, w.verifyAptosChainID(ctx, logger, "http://unused"))
	})

	t.Run("unknown chain lookup error", func(t *testing.T) {
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDEthereum}
		err := w.verifyAptosChainID(ctx, logger, "http://unused")
		require.ErrorContains(t, err, "failed to look up aptos chain id")
	})

	t.Run("request build error", func(t *testing.T) {
		// Embedded control character makes the URL fail to parse in net/http.
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, "http://example.com\x00")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to build chain id request")
	})

	t.Run("connection error", func(t *testing.T) {
		// Spin up and immediately close a server to get a guaranteed-unreachable URL.
		s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to query aptos chain id")
	})

	t.Run("body read error", func(t *testing.T) {
		// Server claims a longer Content-Length than it actually writes, then closes the
		// connection. The client's read of the body will fail with unexpected EOF.
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hj, ok := w.(http.Hijacker)
			require.True(t, ok, "server doesn't support hijacking")
			conn, _, err := hj.Hijack()
			require.NoError(t, err)
			defer conn.Close()
			_, _ = fmt.Fprint(conn, "HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\n")
			_, _ = fmt.Fprint(conn, "short")
		}))
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.ErrorContains(t, err, "failed to read aptos chain id response")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		s := chainIDServer("not json")
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.ErrorContains(t, err, "invalid JSON")
	})

	t.Run("chain_id missing", func(t *testing.T) {
		s := chainIDServer(`{"epoch":"123"}`)
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.ErrorContains(t, err, "chain_id field missing")
	})

	t.Run("chain_id zero", func(t *testing.T) {
		s := chainIDServer(`{"chain_id":0}`)
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.ErrorContains(t, err, "out of expected range")
	})

	t.Run("chain_id too large", func(t *testing.T) {
		s := chainIDServer(`{"chain_id":4294967296}`) // MaxUint32 + 1
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.ErrorContains(t, err, "out of expected range")
	})

	t.Run("mismatch", func(t *testing.T) {
		s := chainIDServer(`{"chain_id":99}`)
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		err := w.verifyAptosChainID(ctx, logger, s.URL)
		require.ErrorContains(t, err, "mismatch")
	})

	t.Run("success aptos mainnet", func(t *testing.T) {
		s := chainIDServer(`{"chain_id":1}`)
		defer s.Close()
		w := &Watcher{env: common.MainNet, chainID: vaa.ChainIDAptos}
		require.NoError(t, w.verifyAptosChainID(ctx, logger, s.URL))
	})

	t.Run("success movement testnet", func(t *testing.T) {
		s := chainIDServer(`{"chain_id":250}`)
		defer s.Close()
		w := &Watcher{env: common.TestNet, chainID: vaa.ChainIDMovement}
		require.NoError(t, w.verifyAptosChainID(ctx, logger, s.URL))
	})
}

// Sanity check: the package-level sentinel errors should be distinct.
func TestSentinelErrors(t *testing.T) {
	require.False(t, errors.Is(ErrInvalidEnv, ErrNotFound))
	require.False(t, errors.Is(ErrNotFound, ErrInvalidEnv))
}
