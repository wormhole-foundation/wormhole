package common

import (
	"context"
)

// ReadFromChannelWithTimeout reads events from the channel until a timeout occurs or the max maxCount is reached.
func ReadFromChannelWithTimeout[T any](ctx context.Context, ch <-chan T, maxCount int) ([]T, error) {
	out := make([]T, 0, maxCount)
	for len(out) < maxCount {
		select {
		case <-ctx.Done():
			return out, ctx.Err()
		case msg := <-ch:
			out = append(out, msg)
		}
	}

	return out, nil
}
