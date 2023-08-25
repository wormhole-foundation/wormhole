package evm

import (
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type WatcherConfig struct {
	NetworkID              watchers.NetworkID // human readable name
	ChainID                vaa.ChainID        // ChainID
	Rpc                    string             // RPC URL
	Contract               string             // hex representation of the contract address
	GuardianSetUpdateChain bool               // if `true`, we will retrieve the GuardianSet from this chain and watch this chain for GuardianSet updates
	WaitForConfirmations   bool               // (optional)
	RootChainRpc           string             // (optional)
	RootChainContract      string             // (optional)
	L1FinalizerRequired    watchers.NetworkID // (optional)
	l1Finalizer            interfaces.L1Finalizer
}

func (wc *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return wc.NetworkID
}

func (wc *WatcherConfig) GetChainID() vaa.ChainID {
	return wc.ChainID
}

func (wc *WatcherConfig) RequiredL1Finalizer() watchers.NetworkID {
	return wc.L1FinalizerRequired
}

func (wc *WatcherConfig) SetL1Finalizer(l1finalizer interfaces.L1Finalizer) {
	wc.l1Finalizer = l1finalizer
}

func (wc *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	queryReqC <-chan *query.PerChainQueryInternal,
	queryResponseC chan<- *query.PerChainQueryResponseInternal,
	setC chan<- *common.GuardianSet,
	env common.Environment,
) (interfaces.L1Finalizer, supervisor.Runnable, error) {

	// only actually use the guardian set channel if wc.GuardianSetUpdateChain == true
	var setWriteC chan<- *common.GuardianSet = nil
	if wc.GuardianSetUpdateChain {
		setWriteC = setC
	}

	var devMode bool = (env == common.UnsafeDevNet)

	watcher := NewEthWatcher(wc.Rpc, eth_common.HexToAddress(wc.Contract), string(wc.NetworkID), wc.ChainID, msgC, setWriteC, obsvReqC, queryReqC, queryResponseC, devMode)
	watcher.SetWaitForConfirmations(wc.WaitForConfirmations)
	if err := watcher.SetRootChainParams(wc.RootChainRpc, wc.RootChainContract); err != nil {
		return nil, nil, err
	}
	watcher.SetL1Finalizer(wc.l1Finalizer)
	return watcher, watcher.Run, nil
}
