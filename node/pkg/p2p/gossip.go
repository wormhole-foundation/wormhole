package p2p

import (
	"context"

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

// SubscribeFilteredWithEnvelope subscribes to a GossipReceiver and filters out messages that are not of type K. It also
// includes the peer ID of the sender in the output.
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
