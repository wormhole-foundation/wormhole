package p2p

import (
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"sync"
)

// The p2p package implements a simple global metrics registry singleton for node status values transmitted on-chain.

type registry struct {
	mu sync.Mutex

	// Mapping of chain IDs to network status messages.
	networkStats map[vaa.ChainID]*gossipv1.Heartbeat_Network

	// Value of Heartbeat.guardian_addr.
	guardianAddress string
}

func NewRegistry() *registry {
	return &registry{
		networkStats: map[vaa.ChainID]*gossipv1.Heartbeat_Network{},
	}
}

var (
	DefaultRegistry = NewRegistry()
)

// SetGuardianAddress stores the node's guardian address to broadcast in Heartbeat messages.
// This should be called once during startup, when the guardian key is loaded.
func (r *registry) SetGuardianAddress(addr string) {
	r.mu.Lock()
	r.guardianAddress = addr
	r.mu.Unlock()
}

// SetNetworkStats sets the current network status to be broadcast in Heartbeat messages.
// The "Id" field is automatically set to the specified chain ID.
func (r *registry) SetNetworkStats(chain vaa.ChainID, data *gossipv1.Heartbeat_Network) {
	r.mu.Lock()
	data.Id = uint32(chain)
	r.networkStats[chain] = data
	r.mu.Unlock()
}
