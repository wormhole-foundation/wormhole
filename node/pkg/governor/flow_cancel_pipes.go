package governor

import "github.com/wormhole-foundation/wormhole/sdk/vaa"

// FlowCancelPipes returns a list of `pipe`s representing a pair of chains for which flow canceling is enabled.
// In practice this list should contain pairs of chains that have a large amount of volume between each other.
// These are more likely to cause chronic congestion which flow canceling can help to alleviate.
// Pairs of chains that are not frequently congested do not need to enable flow canceling as they should have
// plenty of regular Governor capacity to work with.
func FlowCancelPipes() []pipe {
	return []pipe{
		{first: vaa.ChainIDEthereum, second: vaa.ChainIDSui},
	}
}
