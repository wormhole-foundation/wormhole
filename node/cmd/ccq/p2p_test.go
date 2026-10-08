package ccq

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// TestMonitorMissingPeersExitsOnCancel is a regression test for the CCQ peer-monitor
// goroutine (node/cmd/ccq/p2p.go). Once the context is cancelled, the goroutine must
// return (and stop its ticker) instead of spinning in a tight select loop:
// a cancelled context's Done() channel is permanently ready, so without a `return`
// the loop would log "Context cancelled, exiting peer monitoring." forever and never
// exit.
//
// Note: ctx is cancelled before the call, so the ctx.Done() branch is always ready and
// the t.C branch (which would touch thReq/h/bootstrappers) never runs. Passing nil for
// those values is therefore safe here.
func TestMonitorMissingPeersExitsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		monitorMissingPeers(ctx, nil, nil, nil, zap.NewNop())
		close(done)
	}()

	select {
	case <-done:
		// Goroutine exited as expected.
	case <-time.After(2 * time.Second):
		t.Fatal("monitorMissingPeers did not exit after context cancellation (hot loop)")
	}
}
