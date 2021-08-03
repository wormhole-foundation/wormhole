package common

import (
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

// MaxGuardianCount specifies the maximum number of guardians supported by on-chain contracts.
//
// Matching constants:
//  - MAX_LEN_GUARDIAN_KEYS in Solana contract (limited by transaction size - 19 is the maximum amount possible)
//
// The Eth and Terra contracts do not specify a maximum number and support more than that,
// but presumably, chain-specific transaction size limits will apply at some point (untested).
const MaxGuardianCount = 19

type GuardianSet struct {
	// Guardian's public key hashes truncated by the ETH standard hashing mechanism (20 bytes).
	Keys []common.Address
	// On-chain set index
	Index uint32
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
	for n, k := range g.Keys {
		if k == addr {
			return n, true
		}
	}

	return -1, false
}

type GuardianSetState struct {
	mu      sync.Mutex
	current *GuardianSet

	// Last heartbeat message received per guardian. Maintained
	// across guardian set updates - these values don't change.
	lastHeartbeat map[common.Address]*gossipv1.Heartbeat
}

func NewGuardianSetState() *GuardianSetState {
	return &GuardianSetState{
		lastHeartbeat: map[common.Address]*gossipv1.Heartbeat{},
	}
}

func (st *GuardianSetState) Set(set *GuardianSet) {
	st.mu.Lock()
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
func (st *GuardianSetState) LastHeartbeat(addr common.Address) *gossipv1.Heartbeat {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.lastHeartbeat[addr]
}

// SetHeartBeat stores a verified heartbeat observed by a given guardian.
func (st *GuardianSetState) SetHeartBeat(addr common.Address, hb *gossipv1.Heartbeat) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.lastHeartbeat[addr] = hb
}
