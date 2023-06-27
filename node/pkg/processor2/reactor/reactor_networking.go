package reactor

import (
	"context"
	"fmt"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"google.golang.org/protobuf/proto"
)

// NetworkAdapter allows to participate in the outbound communication in a consensus network
type NetworkAdapter interface {
	// BroadcastObservation broadcasts a signed observation on the consensus network
	BroadcastObservation(ctx context.Context, observation *gossipv1.SignedObservation) error
}

// ChannelNetworkAdapter implements NetworkAdapter on a byte slice IO channel. The channel must have a buffer of at
// least length 1 to prevent observations from being dropped.
type ChannelNetworkAdapter struct {
	ch chan<- []byte
}

// NewChannelNetworkAdapter creates a ChannelNetworkAdapter
func NewChannelNetworkAdapter(ch chan<- []byte) *ChannelNetworkAdapter {
	return &ChannelNetworkAdapter{ch: ch}
}

func (c *ChannelNetworkAdapter) BroadcastObservation(_ context.Context, observation *gossipv1.SignedObservation) error {
	w := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedObservation{SignedObservation: observation}}

	msg, err := proto.Marshal(&w)
	if err != nil {
		return fmt.Errorf("failed to serialize observation: %w", err)
	}

	select {
	case c.ch <- msg:
		return nil
	default:
		return fmt.Errorf("broadcast channel is full")
	}
}
