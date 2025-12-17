package p2p

import (
	"sync"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// The p2p package implements a simple global metrics registry singleton for node status values transmitted on-chain.

type registry struct {
	mu sync.Mutex

	// Mapping of chain IDs to network status messages.
	networkStats map[vaa.ChainID]*gossipv1.Heartbeat_Network

	// Per-chain error counters
	errorCounters  map[vaa.ChainID]uint64
	errorCounterMu sync.Mutex

	// Per-chain last VAA timestamp (Unix nanoseconds)
	lastObservationSignedAt   map[vaa.ChainID]int64
	lastObservationSignedAtMu sync.RWMutex
}

func NewRegistry() *registry {
	return &registry{
		networkStats:            map[vaa.ChainID]*gossipv1.Heartbeat_Network{},
		errorCounters:           map[vaa.ChainID]uint64{},
		lastObservationSignedAt: map[vaa.ChainID]int64{},
	}
}

var (
	DefaultRegistry = NewRegistry()
)

// SetNetworkStats sets the current network status to be broadcast in Heartbeat messages.
// The "Id" field is automatically set to the specified chain ID.
func (r *registry) SetNetworkStats(chain vaa.ChainID, data *gossipv1.Heartbeat_Network) {
	r.mu.Lock()
	data.Id = uint32(chain)
	r.networkStats[chain] = data
	r.mu.Unlock()
}

func (r *registry) AddErrorCount(chain vaa.ChainID, delta uint64) {
	r.errorCounterMu.Lock()
	defer r.errorCounterMu.Unlock()
	r.errorCounters[chain] += delta
}

func (r *registry) GetErrorCount(chain vaa.ChainID) uint64 {
	r.errorCounterMu.Lock()
	defer r.errorCounterMu.Unlock()
	return r.errorCounters[chain]
}

// SetLastObservationSignedAtTimestamp sets the timestamp (Unix nanoseconds) of the last time an observation was signed for a chain.
func (r *registry) SetLastObservationSignedAtTimestamp(chain vaa.ChainID, timestamp int64) {
	r.lastObservationSignedAtMu.Lock()
	defer r.lastObservationSignedAtMu.Unlock()
	if existing, ok := r.lastObservationSignedAt[chain]; ok && existing >= timestamp {
		return
	}
	r.lastObservationSignedAt[chain] = timestamp
}

// GetLastObservationSignedAtTimestamp returns the timestamp (Unix nanoseconds) of the last time an observation was signed for a chain.
// Returns 0 if no observation has been signed yet for this chain.
func (r *registry) GetLastObservationSignedAtTimestamp(chain vaa.ChainID) int64 {
	r.lastObservationSignedAtMu.RLock()
	defer r.lastObservationSignedAtMu.RUnlock()
	return r.lastObservationSignedAt[chain]
}
