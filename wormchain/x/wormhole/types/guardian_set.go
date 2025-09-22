package types

import (
	bytes "bytes"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

func (gs GuardianSet) KeysAsAddresses() (addresses []common.Address) {
	for _, key := range gs.Keys {
		addresses = append(addresses, common.BytesToAddress(key))
	}
	return
}

// ContainsKey checks if the given key exists in the guardian set using binary search.
// The guardian set keys are expected to be stored in sorted order.
func (gs GuardianSet) ContainsKey(key common.Address) bool {
	keyBytes := key.Bytes()
	i := sort.Search(len(gs.Keys), func(i int) bool {
		return bytes.Compare(gs.Keys[i], keyBytes) >= 0
	})
	return i < len(gs.Keys) && bytes.Equal(gs.Keys[i], keyBytes)
}

// ValidateBasic performs basic validation of the guardian set
func (gs GuardianSet) ValidateBasic() error {
	if len(gs.Keys) == 0 {
		return fmt.Errorf("guardian set must not be empty")
	}

	if len(gs.Keys) > 255 {
		return fmt.Errorf("guardian set length must be <= 255, is %d", len(gs.Keys))
	}

	// Validate key lengths and ensure they are sorted
	for i, key := range gs.Keys {
		if len(key) != 20 {
			return fmt.Errorf("key [%d]: len %d != 20", i, len(key))
		}
		
		// Check if keys are in sorted order
		if i > 0 && bytes.Compare(gs.Keys[i-1], key) >= 0 {
			return fmt.Errorf("guardian keys must be stored in sorted order")
		}
	}

	return nil
}

// SortKeys sorts the guardian set keys in ascending order.
// This should be called whenever keys are modified to maintain the binary search invariant.
func (gs *GuardianSet) SortKeys() {
	sort.Slice(gs.Keys, func(i, j int) bool {
		return bytes.Compare(gs.Keys[i], gs.Keys[j]) < 0
	})
}
