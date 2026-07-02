package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicRPCDomain(t *testing.T) {
	tests := []struct {
		name           string
		host           string
		expectedDomain string
		expectedMatch  bool
	}{
		{name: "exact match", host: "publicnode.com", expectedDomain: "publicnode.com", expectedMatch: true},
		{name: "exact match with port", host: "publicnode.com:443", expectedDomain: "publicnode.com", expectedMatch: true},
		{name: "subdomain match", host: "rpc.monad.xyz", expectedDomain: "monad.xyz", expectedMatch: true},
		{name: "subdomain match with port", host: "mainnet.infura.io:443", expectedDomain: "infura.io", expectedMatch: true},
		{name: "nested subdomain match", host: "rpc-plume-mainnet-1.t.conduit.xyz", expectedDomain: "t.conduit.xyz", expectedMatch: true},
		{name: "nested subdomain match with port", host: "rpc-plume-mainnet-1.t.conduit.xyz:443", expectedDomain: "t.conduit.xyz", expectedMatch: true},
		{name: "case insensitive", host: "RPC.HYPERLIQUID.XYZ", expectedDomain: "hyperliquid.xyz", expectedMatch: true},
		{name: "trailing dot", host: "polygon-mainnet.quiknode.pro.", expectedDomain: "quiknode.pro", expectedMatch: true},
		{name: "trailing dot with port", host: "polygon-mainnet.quiknode.pro.:443", expectedDomain: "quiknode.pro", expectedMatch: true},
		{name: "ipv6 with port", host: "[::1]:8545", expectedMatch: false},
		{name: "no match", host: "guardian.example.com", expectedMatch: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			domain, ok := publicRPCDomain(test.host)
			assert.Equal(t, test.expectedMatch, ok)
			assert.Equal(t, test.expectedDomain, domain)
		})
	}
}
