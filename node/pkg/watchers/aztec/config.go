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

// RequiredL1Finalizer implements the watchers.WatcherConfig interface
// Return an empty network ID since Aztec handles its own L1/L2 finality checks
func (c *WatcherConfig) RequiredL1Finalizer() watchers.NetworkID {
	return ""
}

// SetL1Finalizer implements the watchers.WatcherConfig interface
// This is a no-op for Aztec since we use our own internal L1Verifier instead
func (c *WatcherConfig) SetL1Finalizer(l1finalizer interfaces.L1Finalizer) {
	// No-op: we use our own internal L1Verifier/L1Finalizer
}

// Create implements the watchers.WatcherConfig interface with the updated signature
func (c *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	queryReqC <-chan *query.PerChainQueryInternal,
	queryRespC chan<- *query.PerChainQueryResponseInternal,
	gst chan<- *common.GuardianSet,
	env common.Environment,
) (interfaces.L1Finalizer, supervisor.Runnable, interfaces.Reobserver, error) {
	// Create the runnable and L1Finalizer
	l1Finalizer, runnable := NewWatcherFromConfig(c.ChainID, string(c.NetworkID), c.Rpc, "0x0d6fe810321185c97a0e94200f998bcae787aaddf953a03b14ec5da3b6838bad", msgC, obsvReqC)

	// Return the L1Verifier as an L1Finalizer along with the runnable
	// This makes it available to the framework if needed
	return l1Finalizer, runnable, nil, nil
}
