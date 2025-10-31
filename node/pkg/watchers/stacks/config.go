package stacks

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

// WatcherConfig defines the configuration for the Stacks watcher
type WatcherConfig struct {
	NetworkID     watchers.NetworkID // human readable name
	ChainID       vaa.ChainID        // ChainID
	RPCURL        string             // Stacks RPC URL
	RPCAuthToken  string             // Stacks RPC Authorization Token
	StateContract string             // Stacks contract address for the Wormhole core (state) contract

	// Optional configurable parameters (zero values will use defaults)
	BitcoinBlockPollInterval time.Duration `mapstructure:"bitcoinBlockPollInterval"` // How often to poll for new Bitcoin blocks
}

func (wc *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return wc.NetworkID
}

func (wc *WatcherConfig) GetChainID() vaa.ChainID {
	return wc.ChainID
}

func (wc *WatcherConfig) RequiredL1Finalizer() watchers.NetworkID {
	return ""
}

func (wc *WatcherConfig) SetL1Finalizer(l1finalizer interfaces.L1Finalizer) {
	// empty
}

//nolint:unparam // L1Finalizer return is always nil for Stacks as it doesn't require L1 finalization
func (wc *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	_ <-chan *query.PerChainQueryInternal, // queryReqC - not used for Stacks
	_ chan<- *query.PerChainQueryResponseInternal, // queryResponseC - not used for Stacks
	_ chan<- *common.GuardianSet, // setC - not used for Stacks
	_ common.Environment, // env - not used for Stacks
) (interfaces.L1Finalizer, supervisor.Runnable, interfaces.Reobserver, error) {
	watcher := NewWatcher(wc.RPCURL, wc.RPCAuthToken, wc.StateContract, wc.BitcoinBlockPollInterval, msgC, obsvReqC)
	return nil, watcher.Run, watcher, nil // L1Finalizer is nil for Stacks (no L1 finalization needed)
}
