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
	L1FinalizerRequired    watchers.NetworkID // (optional)
	l1Finalizer            interfaces.L1Finalizer
	CcqBackfillCache       bool
	TxVerifierEnabled      bool
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
) (interfaces.L1Finalizer, supervisor.Runnable, interfaces.Reobserver, error) {

	// only actually use the guardian set channel if wc.GuardianSetUpdateChain == true
	var setWriteC chan<- *common.GuardianSet = nil
	if wc.GuardianSetUpdateChain {
		setWriteC = setC
	}

	watcher := NewEthWatcher(
		wc.Rpc,
		eth_common.HexToAddress(wc.Contract),
		string(wc.NetworkID),
		wc.ChainID,
		msgC,
		setWriteC,
		obsvReqC,
		queryReqC,
		queryResponseC,
		env,
		wc.CcqBackfillCache,
		wc.TxVerifierEnabled,
	)
	watcher.SetL1Finalizer(wc.l1Finalizer)
	return watcher, watcher.Run, watcher, nil
}
