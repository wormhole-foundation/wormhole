package fixture

import (
	"context"
	"time"
)

// A derived context (timeoutCtx) — its Done() method still resolves to the
// "context" package, so the rule must still recognize it as EscapeContextDone
// and emit the ctx.Done() diagnostic.
func derivedCtxDone(ctx context.Context) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	c := make(chan int, 1)
	select {
	case c <- 1: // want `ctx\.Done\(\)`
	case <-timeoutCtx.Done():
	}
}
