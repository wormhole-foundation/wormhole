package common

import "context"

// SendOnChannel writes msg to c and blocks until ctx is canceled, in which case the write is aborted.
func SendOnChannel[T any](ctx context.Context, c chan<- T, msg T) {
	select {
	case c <- msg:
	case <-ctx.Done():
	}
}
