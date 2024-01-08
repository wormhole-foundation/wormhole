package solana

import (
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	solana_types "github.com/gagliardetto/solana-go"
	solana_rpc "github.com/gagliardetto/solana-go/rpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type WatcherConfig struct {
	NetworkID     watchers.NetworkID // unique identifier of the network
	ChainID       vaa.ChainID        // ChainID
	ReceiveObsReq bool               // if false, this watcher will not get access to the observation request channel
	Rpc           string             // RPC URL
	Websocket     string             // Websocket URL
	Contract      string             // hex representation of the contract address
	Commitment    solana_rpc.CommitmentType
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

func (wc *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	queryReqC <-chan *query.PerChainQueryInternal,
	queryResponseC chan<- *query.PerChainQueryResponseInternal,
	_ chan<- *common.GuardianSet,
	env common.Environment,
) (interfaces.L1Finalizer, supervisor.Runnable, error) {
	solAddress, err := solana_types.PublicKeyFromBase58(wc.Contract)
	if err != nil {
		return nil, nil, err
	}

	if !wc.ReceiveObsReq {
		obsvReqC = nil
	}

	watcher := NewSolanaWatcher(wc.Rpc, &wc.Websocket, solAddress, wc.Contract, msgC, obsvReqC, wc.Commitment, wc.ChainID, queryReqC, queryResponseC)

	return watcher, watcher.Run, nil
}
