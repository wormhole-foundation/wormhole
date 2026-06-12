package fixture

import "context"

// Mixed sends in one select: one ignored, one tracked. The select must still
// produce the ctx.Done() diagnostic because not every send is ignored.
func ignorePartial(ctx context.Context) {
	ignoreMe := make(chan int, 1)
	tracked := make(chan int, 1)
	select {
	case ignoreMe <- 1:
	case tracked <- 2: // want `ctx\.Done\(\)`
	case <-ctx.Done():
	}
}
