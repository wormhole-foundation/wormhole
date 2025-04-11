package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// FlowCancelCorridors returns a list of `pipe`s representing a pair of chains for which flow canceling is enabled.
// In practice this list should contain pairs of chains that have a large amount of volume between each other.
// These are more likely to cause chronic congestion which flow canceling can help to alleviate.
// Pairs of chains that are not frequently congested do not need to enable flow canceling as they should have
// plenty of regular Governor capacity to work with.
func FlowCancelCorridors() []corridor {
	return []corridor{
		{first: vaa.ChainIDEthereum, second: vaa.ChainIDSui},
	}
}

func ValidateCorridors(input []corridor) bool {
	seen := make([]corridor, len(input))
	for _, p := range input {
		// This check is needed when there is exactly one pipe. Otherwise, the seen loop detects this.
		if !p.valid() {
			return false
		}
		for _, s := range seen {
			// Note that equals() also checks that both pipes are valid.
			if p.equals(&s) {
				return false
			}
		}

		seen = append(seen, p)
	}
	return true
}
