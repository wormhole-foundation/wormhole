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

func TestSetLastVaaTimestamp(t *testing.T) {
	registry := NewRegistry()

	// Test setting timestamp for a chain (Unix nanoseconds, like time.Now().UnixNano())
	timestamp := int64(1234567890123456789)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, timestamp)

	// Verify the timestamp was set
	assert.Equal(t, timestamp, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))

	// Verify other chains still have zero timestamp
	assert.Equal(t, int64(0), registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDSolana))
}

func TestGetLastVaaTimestamp(t *testing.T) {
	registry := NewRegistry()

	// Test that unset chains return zero
	assert.Equal(t, int64(0), registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))
	assert.Equal(t, int64(0), registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDSolana))

	// Set timestamp for Ethereum
	timestamp1 := int64(1234567890123456789)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, timestamp1)
	assert.Equal(t, timestamp1, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))

	// Set timestamp for Solana
	timestamp2 := int64(1987654321098765432)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDSolana, timestamp2)
	assert.Equal(t, timestamp2, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDSolana))

	// Ethereum timestamp should remain unchanged
	assert.Equal(t, timestamp1, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))
}

func TestUpdateLastVaaTimestamp(t *testing.T) {
	registry := NewRegistry()

	// Set initial timestamp
	timestamp1 := int64(1000000000000000000)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, timestamp1)
	assert.Equal(t, timestamp1, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))

	// Update to a newer timestamp
	timestamp2 := int64(2000000000000000000)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, timestamp2)
	assert.Equal(t, timestamp2, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))
}

func TestSetLastObservationSignedAtTimestampRejectsOlderTimestamp(t *testing.T) {
	registry := NewRegistry()

	// Set initial timestamp
	newerTimestamp := int64(2000000000000000000)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, newerTimestamp)
	assert.Equal(t, newerTimestamp, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))

	// Try to set an older timestamp - should be rejected
	olderTimestamp := int64(1000000000000000000)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, olderTimestamp)

	// Timestamp should remain unchanged (not updated to older value)
	assert.Equal(t, newerTimestamp, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))

	// Try to set the same timestamp - should be rejected (not less than, >= check)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, newerTimestamp)
	assert.Equal(t, newerTimestamp, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))

	// But setting a newer timestamp should work
	evenNewerTimestamp := int64(3000000000000000000)
	registry.SetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum, evenNewerTimestamp)
	assert.Equal(t, evenNewerTimestamp, registry.GetLastObservationSignedAtTimestamp(vaa.ChainIDEthereum))
}
