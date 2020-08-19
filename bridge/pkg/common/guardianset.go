package common

import (
	"github.com/ethereum/go-ethereum/common"
)

type GuardianSet struct {
	// Guardian's public keys truncated by the ETH standard hashing mechanism (20 bytes).
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

// Get a given address index from the guardian set. Returns (-1, false)
// if the address wasn't found and (addr, true) otherwise.
func (g *GuardianSet) KeyIndex(addr common.Address) (int, bool) {
	for n, k := range g.Keys {
		if k == addr {
			return n, true
		}
	}

	return -1, false
}
