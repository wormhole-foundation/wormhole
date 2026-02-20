package processor

import (
	"fmt"
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

// Chains like Ethereum, Solana, and Wormchain should not be delegated due to their governance implications
var nonDelegableChains = map[vaa.ChainID]struct{}{
	vaa.ChainIDUnset:     {},
	vaa.ChainIDSolana:    {},
	vaa.ChainIDEthereum:  {},
	vaa.ChainIDWormchain: {},
}

type DelegatedGuardianChainConfig struct {
	// TODO: Use map[common.Address]struct{} instead
	// Guardian's public key hashes truncated by the ETH standard hashing mechanism (20 bytes).
	Keys []common.Address

	// quorum value for this set of keys
	// NOTE: This must be a positive integer in practice and should be guaranteed via the EVM ABI bindings in the EVM watcher
	quorum int

	// A map from address to index. Testing showed that, on average, a map is almost three times faster than a sequential search of the key slice.
	// Testing also showed that the map was twice as fast as using a sorted slice and `slices.BinarySearchFunc`. That being said, on a 4GHz CPU,
	// the sequential search takes an average of 800 nanos and the map look up takes about 260 nanos. Is this worth doing?
	keyMap map[common.Address]int
}

// Quorum returns the current quorum value.
func (dc *DelegatedGuardianChainConfig) Quorum() int {
	return dc.quorum
}

func NewDelegatedGuardianChainConfig(keys []common.Address, threshold int) (*DelegatedGuardianChainConfig, error) {
	numKeys := len(keys)
	minThreshold := vaa.CalculateQuorum(numKeys)
	if threshold > numKeys {
		return nil, fmt.Errorf("threshold too high: got %d; want at most %d", threshold, numKeys)
	}
	if threshold < minThreshold {
		return nil, fmt.Errorf("threshold too low: got %d; want at least %d", threshold, minThreshold)
	}

	keyMap := map[common.Address]int{}
	for idx, key := range keys {
		if _, exists := keyMap[key]; exists {
			return nil, fmt.Errorf("duplicate delegated guardian key: %s", key.Hex())
		}
		keyMap[key] = idx
	}
	return &DelegatedGuardianChainConfig{
		Keys:   keys,
		quorum: threshold,
		keyMap: keyMap,
	}, nil
}

// deepCopy returns a deep copy of the delegated guardian chain config
func (dc *DelegatedGuardianChainConfig) deepCopy() DelegatedGuardianChainConfig {
	if dc == nil {
		return DelegatedGuardianChainConfig{}
	}

	ret := DelegatedGuardianChainConfig{
		quorum: dc.quorum,
	}

	if dc.Keys != nil {
		ret.Keys = make([]common.Address, len(dc.Keys))
		copy(ret.Keys, dc.Keys)
	}

	if dc.keyMap != nil {
		ret.keyMap = make(map[common.Address]int, len(dc.keyMap))
		for k, v := range dc.keyMap {
			ret.keyMap[k] = v
		}
	}

	return ret
}

func (dc *DelegatedGuardianChainConfig) KeysAsHexStrings() []string {
	r := make([]string, len(dc.Keys))

	for n, k := range dc.Keys {
		r[n] = k.Hex()
	}

	return r
}

// KeyIndex returns a given address index from the guardian set. Returns (-1, false)
// if the address wasn't found and (addr, true) otherwise.
func (dc *DelegatedGuardianChainConfig) KeyIndex(addr common.Address) (int, bool) {
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

type DelegatedGuardianConfig struct {
	mu     sync.RWMutex
	Chains map[vaa.ChainID]*DelegatedGuardianChainConfig
}

// NewDelegatedGuardianConfig returns a new DelegatedGuardianConfig.
func NewDelegatedGuardianConfig() *DelegatedGuardianConfig {
	return &DelegatedGuardianConfig{
		Chains: map[vaa.ChainID]*DelegatedGuardianChainConfig{},
	}
}

// Set sets the chains map
func (d *DelegatedGuardianConfig) Set(chains map[vaa.ChainID]*DelegatedGuardianChainConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for chain, cfg := range chains {
		if cfg == nil {
			return fmt.Errorf("received nil config for chain %d", chain)
		}
	}
	for chain, cfg := range chains {
		label := chain.String()
		dgSigners.WithLabelValues(label).Set(float64(len(cfg.Keys)))
		dgQuorum.WithLabelValues(label).Set(float64(cfg.Quorum()))
	}
	for chain := range d.Chains {
		if _, ok := chains[chain]; !ok {
			label := chain.String()
			dgSigners.DeleteLabelValues(label)
			dgQuorum.DeleteLabelValues(label)
		}
	}
	d.Chains = chains

	return nil
}

// ReadChainConfig returns the deep copy of the delegated guardian chain config for a specific chain.
// Returns (DelegatedGuardianChainConfig{}, false) if it doesn't exist and (cfg.deepCopy(), true) otherwise.
func (d *DelegatedGuardianConfig) ReadChainConfig(chain vaa.ChainID) (DelegatedGuardianChainConfig, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	cfg := d.Chains[chain]
	if cfg == nil {
		return DelegatedGuardianChainConfig{}, false
	}
	return cfg.deepCopy(), true
}

// ReadAll returns the map of deep copies of all delegated guardian chain configs.
func (d *DelegatedGuardianConfig) ReadAll() map[vaa.ChainID]DelegatedGuardianChainConfig {
	d.mu.RLock()
	defer d.mu.RUnlock()

	ret := make(map[vaa.ChainID]DelegatedGuardianChainConfig, len(d.Chains))
	for chain, cfg := range d.Chains {
		if cfg == nil {
			continue
		}
		ret[chain] = cfg.deepCopy()
	}

	return ret
}
