package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Represents a pair of Governed chains. Ordering is arbitrary.
type corridor struct {
	first  vaa.ChainID
	second vaa.ChainID
}

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
	if len(input) == 0 {
		return false
	}

	seen := make([]corridor, 0, len(input))
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

// valid checks whether a corridor is valid. A corridor is invalid if both chain IDs are equal.
func (p *corridor) valid() bool {
	return p.first != p.second && p.first != vaa.ChainIDUnset && p.second != vaa.ChainIDUnset
}

// equals checks whether two corrdidors are equal. This method exists to demonstrate that the ordering of the
// corridor's elements doesn't matter. It also makes it easier to check whether two chains are 'connected' by a corridor
// without needing to sort or manipulate the elements.
func (p *corridor) equals(p2 *corridor) bool {
	if !p.valid() || !p2.valid() {
		// We want to make invalid corridors unusable, so make them fail the equality check.
		// This is a protective measure in case a developer tries to do some logic on invalid corridors
		// and forgets to check valid() first.
		return false
	}
	if p.first == p2.first && p.second == p2.second {
		return true
	}
	// Ordering doesn't matter
	if p.first == p2.second && p2.first == p.second {
		return true
	}
	return false
}
