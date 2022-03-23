package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, "", registry.guardianAddress)
	assert.Equal(t, 0, len(registry.errorCounters))
	assert.Equal(t, 0, len(registry.networkStats))
}

func TestSetGuardianAddress(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, "", registry.guardianAddress)

	registry.SetGuardianAddress("foo")
	assert.Equal(t, "foo", registry.guardianAddress)
}

func TestSetNetworkStats(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, 0, len(registry.networkStats))

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	contractAddr := base58.Encode(pub[:])

	heart_beat := &gossipv1.Heartbeat_Network{
		ContractAddress: contractAddr,
	}

	registry.SetNetworkStats(vaa.ChainIDEthereum, heart_beat)
	assert.Equal(t, 1, len(registry.networkStats))
}

func TestAddErrorCount(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, 0, len(registry.errorCounters))

	registry.AddErrorCount(vaa.ChainIDEthereum, uint64(1))
	assert.Equal(t, 1, len(registry.errorCounters))
}

func TestGetErrorCount(t *testing.T) {
	registry := NewRegistry()
	assert.Equal(t, 0, len(registry.errorCounters))
	assert.Equal(t, uint64(0), registry.GetErrorCount(vaa.ChainIDEthereum))
	assert.Equal(t, uint64(0), registry.GetErrorCount(vaa.ChainIDSolana))

	registry.AddErrorCount(vaa.ChainIDEthereum, uint64(1))
	assert.Equal(t, uint64(1), registry.GetErrorCount(vaa.ChainIDEthereum))
	assert.Equal(t, uint64(0), registry.GetErrorCount(vaa.ChainIDSolana))
}
