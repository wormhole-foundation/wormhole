package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, 0, len(registry.errorCounters))
	assert.Equal(t, 0, len(registry.networkStats))
}

func TestSetNetworkStats(t *testing.T) {
	registry := NewRegistry()

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	contractAddr := base58.Encode(pub[:])

	heartBeat := &gossipv1.Heartbeat_Network{
		ContractAddress: contractAddr,
	}

	expect := make(map[vaa.ChainID]*gossipv1.Heartbeat_Network)
	expect[vaa.ChainIDEthereum] = heartBeat

	registry.SetNetworkStats(vaa.ChainIDEthereum, heartBeat)
	assert.Equal(t, expect, registry.networkStats)
}

func TestAddErrorCount(t *testing.T) {
	registry := NewRegistry()

	expect := make(map[vaa.ChainID]uint64)
	expect[vaa.ChainIDEthereum] = 1

	registry.AddErrorCount(vaa.ChainIDEthereum, uint64(1))
	assert.Equal(t, expect, registry.errorCounters)
}

func TestGetErrorCount(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, uint64(0), registry.GetErrorCount(vaa.ChainIDEthereum))
	assert.Equal(t, uint64(0), registry.GetErrorCount(vaa.ChainIDSolana))

	registry.AddErrorCount(vaa.ChainIDEthereum, uint64(1))
	assert.Equal(t, uint64(1), registry.GetErrorCount(vaa.ChainIDEthereum))
	assert.Equal(t, uint64(0), registry.GetErrorCount(vaa.ChainIDSolana))
}
