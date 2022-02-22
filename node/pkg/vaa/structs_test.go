package vaa

import "testing"
import "github.com/stretchr/testify/assert"

func TestChainIDFromString(t *testing.T) {
	type test struct {
		i string
		o ChainID
	}

	// Positive Test Cases
	p_tests := []test{
		{i: "solana", o: ChainIDSolana},
		{i: "ethereum", o: ChainIDEthereum},
		{i: "terra", o: ChainIDTerra},
		{i: "bsc", o: ChainIDBSC},
		{i: "polygon", o: ChainIDPolygon},
		{i: "avalanche", o: ChainIDAvalanche},
		{i: "oasis", o: ChainIDOasis},
		{i: "fantom", o: ChainIDFantom},
		{i: "algorand", o: ChainIDAlgorand},
		{i: "ethereum-ropsten", o: ChainIDEthereumRopsten},
		{i: "Solana", o: ChainIDSolana},
		{i: "Ethereum", o: ChainIDEthereum},
		{i: "Terra", o: ChainIDTerra},
		{i: "Bsc", o: ChainIDBSC},
		{i: "Polygon", o: ChainIDPolygon},
		{i: "Avalanche", o: ChainIDAvalanche},
		{i: "Oasis", o: ChainIDOasis},
		{i: "Fantom", o: ChainIDFantom},
		{i: "Algorand", o: ChainIDAlgorand},
	}

	// Negative Test Cases
	n_tests := []test{
		{i: "Unknown", o: ChainIDUnset},
	}

	for _, tc := range p_tests {
		got, got_err := ChainIDFromString(tc.i)
		assert.Equal(t, tc.o, got)
		assert.Nil(t, got_err)
	}

	for _, tc := range n_tests {
		got, got_err := ChainIDFromString(tc.i)
		assert.Equal(t, tc.o, got)
		assert.NotNil(t, got_err)
	}

}
