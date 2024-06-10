package common

import (
	"fmt"
	"sync"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	gsIndex = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_guardian_set_index",
			Help: "The guardians set index",
		})
	gsSigners = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_guardian_set_signers",
			Help: "Number of signers in the guardian set.",
		})
)

// MaxGuardianCount specifies the maximum number of guardians supported by on-chain contracts.
//
// Matching constants:
//   - MAX_LEN_GUARDIAN_KEYS in Solana contract (limited by transaction size - 19 is the maximum amount possible)
//
// The Eth and Terra contracts do not specify a maximum number and support more than that,
// but presumably, chain-specific transaction size limits will apply at some point (untested).
const MaxGuardianCount = 19

// MaxNodesPerGuardian specifies the maximum amount of nodes per guardian key that we'll accept
// whenever we maintain any per-guardian, per-node state.
//
// There currently isn't any state clean up, so the value is on the high side to prevent
// accidentally reaching the limit due to operational mistakes.
const MaxNodesPerGuardian = 15

// MaxStateAge specified the maximum age of state entries in seconds. Expired entries are purged
// from the state by Cleanup().
const MaxStateAge = 1 * time.Minute

type GuardianSet struct {
	// Guardian's public key hashes truncated by the ETH standard hashing mechanism (20 bytes).
	Keys []common.Address

	// On-chain set index
	Index uint32

	// quorum value for this set of keys
	quorum int

	// A map from address to index. Testing showed that, on average, a map is almost three times faster than a sequential search of the key slice.
	// Testing also showed that the map was twice as fast as using a sorted slice and `slices.BinarySearchFunc`. That being said, on a 4GHz CPU,
	// the sequential search takes an average of 800 nanos and the map look up takes about 260 nanos. Is this worth doing?
	keyMap map[common.Address]int
}

// Quorum returns the current quorum value.
func (gs *GuardianSet) Quorum() int {
	return gs.quorum
}

func NewGuardianSet(keys []common.Address, index uint32) *GuardianSet {
	keyMap := map[common.Address]int{}
	for idx, key := range keys {
		keyMap[key] = idx
	}
	return &GuardianSet{
		Keys:   keys,
		Index:  index,
		quorum: vaa.CalculateQuorum(len(keys)),
		keyMap: keyMap,
	}
}

func (g *GuardianSet) KeysAsHexStrings() []string {
	r := make([]string, len(g.Keys))

	for n, k := range g.Keys {
		r[n] = k.Hex()
	}

	return r
}

// KeyIndex returns a given address index from the guardian set. Returns (-1, false)
// if the address wasn't found and (addr, true) otherwise.
func (g *GuardianSet) KeyIndex(addr common.Address) (int, bool) {
	if g.keyMap != nil {
		if idx, found := g.keyMap[addr]; found {
			return idx, true
		}
	} else {
		for n, k := range g.Keys {
			if k == addr {
				return n, true
			}
		}
	}

	return -1, false
}

type GuardianSetState struct {
	mu      sync.Mutex
	current *GuardianSet

	// Last heartbeat message received per guardian per p2p node. Maintained
	// across guardian set updates - these values don't change.
	lastHeartbeats map[common.Address]map[peer.ID]*gossipv1.Heartbeat
	updateC        chan *gossipv1.Heartbeat
}

// NewGuardianSetState returns a new GuardianSetState.
//
// The provided channel will be pushed heartbeat updates as they are set,
// but be aware that the channel will block guardian set updates if full.
func NewGuardianSetState(guardianSetStateUpdateC chan *gossipv1.Heartbeat) *GuardianSetState {
	return &GuardianSetState{
		lastHeartbeats: map[common.Address]map[peer.ID]*gossipv1.Heartbeat{},
		updateC:        guardianSetStateUpdateC,
	}
}

func (st *GuardianSetState) Set(set *GuardianSet) {
	st.mu.Lock()
	gsIndex.Set(float64(set.Index))
	gsSigners.Set(float64(len(set.Keys)))
	defer st.mu.Unlock()

	st.current = set
}

func (st *GuardianSetState) Get() *GuardianSet {
	st.mu.Lock()
	defer st.mu.Unlock()

	return st.current
}

// LastHeartbeat returns the most recent heartbeat message received for
// a given guardian node, or nil if none have been received.
func (st *GuardianSetState) LastHeartbeat(addr common.Address) map[peer.ID]*gossipv1.Heartbeat {
	st.mu.Lock()
	defer st.mu.Unlock()
	ret := make(map[peer.ID]*gossipv1.Heartbeat)
	for k, v := range st.lastHeartbeats[addr] {
		ret[k] = v
	}
	return ret
}

// SetHeartbeat stores a verified heartbeat observed by a given guardian.
func (st *GuardianSetState) SetHeartbeat(addr common.Address, peerId peer.ID, hb *gossipv1.Heartbeat) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	v, ok := st.lastHeartbeats[addr]

	if !ok {
		v = make(map[peer.ID]*gossipv1.Heartbeat)
		st.lastHeartbeats[addr] = v
	} else {
		if len(v) >= MaxNodesPerGuardian {
			// TODO: age out old entries?
			return fmt.Errorf("too many nodes (%d) for guardian, cannot store entry", len(v))
		}
	}

	v[peerId] = hb
	if st.updateC != nil {
		st.updateC <- hb
	}
	return nil
}

// GetAll returns all stored heartbeats.
func (st *GuardianSetState) GetAll() map[common.Address]map[peer.ID]*gossipv1.Heartbeat {
	st.mu.Lock()
	defer st.mu.Unlock()

	ret := make(map[common.Address]map[peer.ID]*gossipv1.Heartbeat)

	// Deep copy
	for addr, v := range st.lastHeartbeats {
		ret[addr] = make(map[peer.ID]*gossipv1.Heartbeat)
		for peerId, hb := range v {
			ret[addr][peerId] = hb
		}
	}

	return ret
}

// Cleanup removes expired entries from the state.
func (st *GuardianSetState) Cleanup() {
	st.mu.Lock()
	defer st.mu.Unlock()

	for addr, v := range st.lastHeartbeats {
		for peerId, hb := range v {
			ts := time.Unix(0, hb.Timestamp)
			if time.Since(ts) > MaxStateAge {
				delete(st.lastHeartbeats[addr], peerId)
			}
		}
	}
}
