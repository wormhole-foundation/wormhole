package p2p

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/libp2p/go-libp2p/core/peer"
)

// GossipEnvelope contains a message of type *gossipv1.GossipMessage with the peer ID of the sender.
type GossipEnvelope struct {
	Message *gossipv1.GossipMessage
	From    peer.ID
}

// GossipIO is a combination of GossipReceiver and GossipSender.
type GossipIO interface {
	GossipReceiver
	GossipSender
}

// GossipReceiver is an interface for subscribing to GossipMessages.
type GossipReceiver interface {
	Subscribe(ctx context.Context, ch chan<- *GossipEnvelope) error
}

// GossipSender is an interface for sending GossipMessages.
type GossipSender interface {
	Send(ctx context.Context, msg *gossipv1.GossipMessage) error
}

// FilteredEnvelope contains a message of type K with the peer ID of the sender.
type FilteredEnvelope[K any] struct {
	Message K
	From    peer.ID
}

// SubscribeFiltered subscribes to a GossipReceiver and filters out messages that are not of type K.
func SubscribeFiltered[K any](ctx context.Context, in GossipReceiver, ch chan<- K) error {
	msgInCh := make(chan *GossipEnvelope)
	err := in.Subscribe(ctx, msgInCh)
	if err != nil {
		return err
	}

	go func() {
		for k := range msgInCh {
			switch pK := k.Message.Message.(type) {
			case K:
				ch <- pK
			}
		}
	}()

	return nil
}

// SubscribeFilteredWithEnvelope subscribes to a GossipReceiver and filters out messages that are not of type K. It also
// includes the peer ID of the sender in the output.
func SubscribeFilteredWithEnvelope[K any](ctx context.Context, in GossipReceiver, ch chan<- *FilteredEnvelope[K]) error {
	msgInCh := make(chan *GossipEnvelope)
	err := in.Subscribe(ctx, msgInCh)
	if err != nil {
		return err
	}

	go func() {
		for k := range msgInCh {
			switch pK := k.Message.Message.(type) {
			case K:
				ch <- &FilteredEnvelope[K]{
					Message: pK,
					From:    k.From,
				}
			}
		}
	}()

	return nil
}

var meteredChannelBufferSize = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "wormhole_metered_channel_buffer_size",
		Help: "Total number of items currently queued in the metered channel buffer",
	}, []string{"name"})

var meteredChannelBufferDropped = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "wormhole_metered_channel_buffer_dropped",
		Help: "Total number of items dropped by the metered channel buffer",
	}, []string{"name"})

var meteredChannelBufferProcessingTime = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "wormhole_metered_channel_buffer_processing",
		Help: "Histogram of consumption times for items in the metered channel buffer",
	}, []string{"name"})

// MeteredBufferedChannelPair creates a buffered channel pair that is metered. This means it will track the number of
// pending items, the time it takes to process items and how many items are dropped.
// The channel pair is intended to be used as follows:
// - The first channel is used to send items to the buffer. It will never block as dropping items is handled by the buffer.
// - The second channel is used to receive items from the buffer
// - The buffer will drop items if the buffer is full and track the number of dropped items
// - The buffer will track the number of pending items in the buffer
// - The buffer will track the time it takes to process items
// - The buffer will stop processing items if the context is cancelled
func MeteredBufferedChannelPair[K any](ctx context.Context, bufferSize int, name string) (chan<- K, <-chan K) {
	inCh := make(chan K, bufferSize)
	bufferCh := make(chan K, bufferSize)

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				pending := len(bufferCh)
				meteredChannelBufferSize.WithLabelValues(name).Set(float64(pending))
			case k := <-inCh:
				select {
				case bufferCh <- k:
				default:
					// Drop the message if the buffer is full
					meteredChannelBufferDropped.WithLabelValues(name).Inc()
				}
			}
		}
	}()

	out := make(chan K)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case k := <-bufferCh:
				start := time.Now()
				out <- k
				took := time.Since(start)
				meteredChannelBufferProcessingTime.WithLabelValues(name).Observe(took.Seconds())
			}
		}
	}()

	return inCh, out
}
