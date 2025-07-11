package common

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	channelWriteDrops = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_channel_write_drops",
			Help: "Total number of channel writes that were dropped due to channel overflow",
		}, []string{"channel_id"})
)

// WriteToChannelWithoutBlocking attempts to write the specified event to the specified channel. If the write would block,
// it increments the `channelWriteDrops` metric with the specified channel ID.
func WriteToChannelWithoutBlocking[T any](channel chan<- T, evt T, label string) {
	select {
	case channel <- evt:
	default:
		channelWriteDrops.WithLabelValues(label).Inc()
	}
}

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
