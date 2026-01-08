package processor

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	dgSigners = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_delegated_guardian_set_signers",
			Help: "Number of signers in the delegated guardian set.",
		},
		[]string{"chain"},
	)
	dgQuorum = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_delegated_guardian_set_quorum",
			Help: "Quorum for the delegated guardian set.",
		},
		[]string{"chain"},
	)
)

type DelegateGuardianChainConfig struct {
	// Guardian's public key hashes truncated by the ETH standard hashing mechanism (20 bytes).
	Keys []common.Address

	// quorum value for this set of keys
	quorum int

	// A map from address to index. Testing showed that, on average, a map is almost three times faster than a sequential search of the key slice.
	// Testing also showed that the map was twice as fast as using a sorted slice and `slices.BinarySearchFunc`. That being said, on a 4GHz CPU,
	// the sequential search takes an average of 800 nanos and the map look up takes about 260 nanos. Is this worth doing?
	keyMap map[common.Address]int
}

// Quorum returns the current quorum value.
func (dc *DelegateGuardianChainConfig) Quorum() int {
	return dc.quorum
}

func NewDelegateGuardianChainConfig(keys []common.Address, threshold int) *DelegateGuardianChainConfig {
	keyMap := map[common.Address]int{}
	for idx, key := range keys {
		keyMap[key] = idx
	}
	return &DelegateGuardianChainConfig{
		Keys:   keys,
		quorum: threshold,
		keyMap: keyMap,
	}
}

// KeyIndex returns a given address index from the guardian set. Returns (-1, false)
// if the address wasn't found and (addr, true) otherwise.
func (dc *DelegateGuardianChainConfig) KeyIndex(addr common.Address) (int, bool) { //nolint: unparam // The index is unused but it is retained as it could be used in future tests
	if dc.keyMap != nil {
		if idx, found := dc.keyMap[addr]; found {
			return idx, true
		}
	} else {
		for n, k := range dc.Keys {
			if k == addr {
				return n, true
			}
		}
	}

	return -1, false
}

type DelegateGuardianConfig struct {
	// TODO(delegated-guardian-sets): Try RWMutex since reads > writes
	mu     sync.Mutex
	Chains map[vaa.ChainID]*DelegateGuardianChainConfig
}

// NewDelegateGuardianConfig returns a new DelegateGuardianConfig.
func NewDelegateGuardianConfig() *DelegateGuardianConfig {
	return &DelegateGuardianConfig{
		Chains: map[vaa.ChainID]*DelegateGuardianChainConfig{},
	}
}

func (d *DelegateGuardianConfig) SetChainConfig(chain vaa.ChainID, cfg *DelegateGuardianChainConfig) {
	d.mu.Lock()
	dgSigners.WithLabelValues(chain.String()).Set(float64(len(cfg.Keys)))
	dgQuorum.WithLabelValues(chain.String()).Set(float64(cfg.quorum))
	defer d.mu.Unlock()

	d.Chains[chain] = cfg
}

// GetChainConfig returns the delegate guardian chain config for a specific chain, or nil if none.
func (d *DelegateGuardianConfig) GetChainConfig(chain vaa.ChainID) *DelegateGuardianChainConfig {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Chains[chain]
}
