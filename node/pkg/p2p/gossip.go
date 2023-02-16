package p2p

import (
	"context"

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
