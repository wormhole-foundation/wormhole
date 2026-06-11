package fixture

import "context"

func ignoreCtxDone(ctx context.Context) {
	ignoreMe := make(chan int, 1)
	select {
	case ignoreMe <- 1:
	case <-ctx.Done():
	}
}
