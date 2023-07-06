package helpers

import (
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
)

func getFullNode(c *cosmos.CosmosChain) *cosmos.ChainNode {
	if len(c.FullNodes) > 0 {
		// use first full node
		return c.FullNodes[0]
	}
	// use first validator
	return c.Validators[0]
}
	

