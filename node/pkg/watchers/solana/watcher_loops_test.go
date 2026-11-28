package solana

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/coder/websocket"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

const loopTestTimeout = 5 * time.Second

// newLoopTestWatcher returns a watcher with the channels that the run loops use.
func newLoopTestWatcher(t *testing.T, rpcClient solanaRPCClient) (*SolanaWatcher, chan *gossipv1.ObservationRequest) {
	t.Helper()
	s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, nil)
	obsvReqC := make(chan *gossipv1.ObservationRequest)
	s.obsvReqC = obsvReqC
	s.errC = make(chan error, 1)
	s.pumpData = make(chan []byte)
	s.rpcClient = rpcClient
	return s, obsvReqC
}

// goErr runs fn in a goroutine and returns a channel that receives its result.
func goErr(fn func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	return done
}

func requireDone(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(loopTestTimeout):
		require.FailNow(t, "routine did not return")
		return nil
	}
}

func TestRunWatcherStopsWhenContextIsCanceled(t *testing.T) {
	s, _ := newLoopTestWatcher(t, newMockSolanaRPCClient())
	ctx, cancel := context.WithCancel(context.Background())

	done := goErr(func() error { return s.runWatcher(ctx, zap.NewNop(), time.Hour, true, "contract") })
	cancel()

	require.NoError(t, requireDone(t, done))
}

func TestRunWatcherRecordsSlotOnEachTick(t *testing.T) {
	m := newMockSolanaRPCClient()
	m.slot = 100
	s, _ := newLoopTestWatcher(t, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// useWs is true so that the standard block polling does not run.
	done := goErr(func() error { return s.runWatcher(ctx, zap.NewNop(), 5*time.Millisecond, true, "contract") })

	require.Eventually(t, func() bool { return atomic.LoadUint64(&s.lastSlot) == 100 }, loopTestTimeout, 5*time.Millisecond)
	cancel()
	require.NoError(t, requireDone(t, done))
}

func TestRunWatcherKeepsRunningAfterInvalidObservationRequest(t *testing.T) {
	m := newMockSolanaRPCClient()
	m.slot = 7
	s, obsvReqC := newLoopTestWatcher(t, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := goErr(func() error { return s.runWatcher(ctx, zap.NewNop(), 5*time.Millisecond, true, "contract") })

	// A chain ID that is not known to the SDK is logged and skipped. It must not stop the loop.
	select {
	case obsvReqC <- &gossipv1.ObservationRequest{ChainId: 65000, TxHash: []byte{1}}:
	case <-time.After(loopTestTimeout):
		require.FailNow(t, "observation request was not received")
	}

	require.Eventually(t, func() bool { return atomic.LoadUint64(&s.lastSlot) == 7 }, loopTestTimeout, 5*time.Millisecond)
	cancel()
	require.NoError(t, requireDone(t, done))
}

func TestRunWatcherReportsSlotError(t *testing.T) {
	slotErr := errors.New("rpc is down")
	m := newMockSolanaRPCClient()
	m.slotErr = slotErr
	s, _ := newLoopTestWatcher(t, m)

	err := requireDone(t, goErr(func() error {
		return s.runWatcher(context.Background(), zap.NewNop(), 5*time.Millisecond, true, "contract")
	}))

	require.ErrorIs(t, err, slotErr)
	require.Len(t, s.errC, 1, "the error must also reach the supervisor channel")
	assert.ErrorIs(t, <-s.errC, slotErr)
}

func TestRunWatcherDoesNotBlockWhenErrorChannelIsFull(t *testing.T) {
	slotErr := errors.New("rpc is down")
	m := newMockSolanaRPCClient()
	m.slotErr = slotErr
	s, _ := newLoopTestWatcher(t, m)
	earlier := errors.New("earlier error")
	s.errC <- earlier // The buffer holds one error, so it is now full.

	err := requireDone(t, goErr(func() error {
		return s.runWatcher(context.Background(), zap.NewNop(), 5*time.Millisecond, true, "contract")
	}))

	require.ErrorIs(t, err, slotErr)
	assert.ErrorIs(t, <-s.errC, earlier, "the full channel keeps the first error")
}

// dialTestWebSocket starts a local server that sends each message to the client, then reads until
// the client closes. It returns the client connection.
func dialTestWebSocket(t *testing.T, messages ...string) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.CloseNow() }()
		ctx := context.Background()
		for _, msg := range messages {
			if err := c.Write(ctx, websocket.MessageText, []byte(msg)); err != nil {
				return
			}
		}
		for {
			if _, _, err := c.Read(ctx); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), loopTestTimeout)
	defer cancel()
	ws, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
	return ws
}

func TestRunDataPumpForwardsMessages(t *testing.T) {
	s, _ := newLoopTestWatcher(t, newMockSolanaRPCClient())
	ws := dialTestWebSocket(t, "first", "second")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := goErr(func() error { return s.runDataPump(ctx, zap.NewNop(), ws) })

	for _, want := range []string{"first", "second"} {
		select {
		case got := <-s.pumpData:
			assert.Equal(t, want, string(got))
		case <-time.After(loopTestTimeout):
			require.FailNow(t, "message was not forwarded", want)
		}
	}
	cancel()
	err := requireDone(t, done)
	assert.True(t, err == nil || errors.Is(err, context.Canceled), "unexpected error: %v", err)
}

// A consumer that stops reading must not keep the pump alive after the context ends.
func TestRunDataPumpStopsWhenConsumerIsGone(t *testing.T) {
	s, _ := newLoopTestWatcher(t, newMockSolanaRPCClient())
	ws := dialTestWebSocket(t, "unread")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := goErr(func() error { return s.runDataPump(ctx, zap.NewNop(), ws) })

	// Give the pump time to read the message and block on the send to pumpData.
	require.Never(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 200*time.Millisecond, 10*time.Millisecond, "the pump must wait for its consumer")

	cancel()
	err := requireDone(t, done)
	assert.True(t, err == nil || errors.Is(err, context.Canceled), "unexpected error: %v", err)
}
