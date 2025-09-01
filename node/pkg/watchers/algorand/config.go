package algorand

import (
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type WatcherConfig struct {
	NetworkID    watchers.NetworkID // human readable name
	ChainID      vaa.ChainID        // ChainID
	IndexerRPC   string
	IndexerToken string
	AlgodRPC     string
	AlgodToken   string
	AppID        uint64
}

func (wc *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return wc.NetworkID
}

func (wc *WatcherConfig) GetChainID() vaa.ChainID {
	return wc.ChainID
}

//nolint:unparam // error is always nil here but the return type is required to satisfy the interface.
func (wc *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	_ <-chan *query.PerChainQueryInternal,
	_ chan<- *query.PerChainQueryResponseInternal,
	_ chan<- *common.GuardianSet,
	_ common.Environment,
) (supervisor.Runnable, interfaces.Reobserver, error) {
	return NewWatcher(wc.IndexerRPC, wc.IndexerToken, wc.AlgodRPC, wc.AlgodToken, wc.AppID, msgC, obsvReqC).Run, nil, nil
}
