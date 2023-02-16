package p2p

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/libp2p/go-libp2p/core/peer"
)

type GossipEnvelope struct {
	Message *gossipv1.GossipMessage
	From    peer.ID
}

type GossipIO interface {
	GossipReceiver
	GossipSender
}

type GossipReceiver interface {
	Subscribe(ctx context.Context, ch chan<- *GossipEnvelope) error
}

type GossipSender interface {
	Send(ctx context.Context, msg *gossipv1.GossipMessage) error
}

type FilteredEnvelope[K any] struct {
	Message K
	From    peer.ID
}

func SubscribeFiltered[K any](ctx context.Context, in GossipReceiver, ch chan<- K) error {
	msgInCh := make(chan *GossipEnvelope)
	err := in.Subscribe(ctx, msgInCh)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case k := <-msgInCh:
				switch pK := k.Message.Message.(type) {
				case K:
					ch <- pK
				}
			}
		}
	}()

	return nil
}

func SubscribeFilteredWithEnvelope[K any](ctx context.Context, in GossipReceiver, ch chan<- *FilteredEnvelope[K]) error {
	msgInCh := make(chan *GossipEnvelope)
	err := in.Subscribe(ctx, msgInCh)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case k := <-msgInCh:
				switch pK := k.Message.Message.(type) {
				case K:
					ch <- &FilteredEnvelope[K]{
						Message: pK,
						From:    k.From,
					}
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

var meteredChannelBufferProcessingTime = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "wormhole_metered_channel_buffer_processing",
		Help: "Histogram of consumption times for items in the metered channel buffer",
	}, []string{"name"})

func MeteredBufferedChannelPair[K any](ctx context.Context, bufferSize int, name string) (chan<- K, <-chan K) {
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

	return bufferCh, out
}
