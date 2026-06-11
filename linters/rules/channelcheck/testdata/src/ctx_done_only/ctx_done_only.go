package fixture

import "context"

// Single match for having a 'Done()' without anything else that was useful.
func ctxDoneOnly(ctx context.Context) {
	c := make(chan int, 1)
	select {
	case c <- 1: // want `ctx\.Done\(\)`
	case <-ctx.Done():
	}
}
