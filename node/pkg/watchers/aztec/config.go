package aztec

import (
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Config holds all configuration for the Aztec watcher
type Config struct {
	// Chain identification
	ChainID   vaa.ChainID
	NetworkID string

	// Connection details
	RpcURL          string
	ContractAddress string

	// Processing parameters
	StartBlock        int
	PayloadInitialCap int

	// Timeouts and intervals
	RPCTimeout            time.Duration
	LogProcessingInterval time.Duration
	RequestTimeout        time.Duration

	// Retry configuration
	MaxRetries        int
	InitialBackoff    time.Duration
	BackoffMultiplier float64
}

// DefaultConfig returns a default configuration
func DefaultConfig(chainID vaa.ChainID, networkID string, rpcURL, contractAddress string) Config {
	return Config{
		// Chain identification
		ChainID:   chainID,
		NetworkID: networkID,

		// Connection details
		RpcURL:          rpcURL,
		ContractAddress: contractAddress,

		// Processing parameters
		StartBlock:        1,
		PayloadInitialCap: 13,

		// Timeouts and intervals
		RPCTimeout:            30 * time.Second,
		LogProcessingInterval: 10 * time.Second,
		RequestTimeout:        10 * time.Second,

		// Retry configuration
		MaxRetries:        3,
		InitialBackoff:    500 * time.Millisecond,
		BackoffMultiplier: 1.5,
	}
}

// GetChainID implements the watchers.WatcherConfig interface
func (c *WatcherConfig) GetChainID() vaa.ChainID {
	return c.ChainID
}

// GetNetworkID implements the watchers.WatcherConfig interface
func (c *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return c.NetworkID
}

// Create implements the watchers.WatcherConfig interface
//
//nolint:unparam // Aztec doesn't implement Reobserver, so we always return nil
func (c *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	_ <-chan *query.PerChainQueryInternal,
	_ chan<- *query.PerChainQueryResponseInternal,
	_ chan<- *common.GuardianSet,
	_ common.Environment,
) (supervisor.Runnable, interfaces.Reobserver, error) {
	// Create the runnable (L1Finalizer is handled internally by the watcher)
	runnable := NewWatcherRunnable(c.ChainID, string(c.NetworkID), c.Rpc, c.Contract, msgC, obsvReqC)

	// Aztec does not implement a Reobserver, so we return nil for that interface
	return runnable, nil, nil
}
