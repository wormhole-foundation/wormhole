package fixture

import "context"

func ctxDoneOnly(ctx context.Context) {
	c := make(chan int, 1)
	select {
	case c <- 1:
	case <-ctx.Done():
	}
}
