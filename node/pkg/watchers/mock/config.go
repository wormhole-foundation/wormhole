package mock

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

type ObservationDb map[eth_common.Hash]*common.MessagePublication

// The Mock Watcher is a watcher that will make a new observation
type WatcherConfig struct {
	NetworkID        watchers.NetworkID              // human readable name
	ChainID          vaa.ChainID                     // ChainID
	MockObservationC chan *common.MessagePublication // Channel to feed this watcher mock observations that it will then make
	ObservationDb    ObservationDb                   // If the watcher receives a re-observation request with a TxHash in this map, it will make the corresponding observation in this map.
	MockSetC         <-chan *common.GuardianSet
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
	setC chan<- *common.GuardianSet,
	_ common.Environment,
) (supervisor.Runnable, interfaces.Reobserver, error) {
	return NewWatcherRunnable(msgC, obsvReqC, setC, wc), nil, nil
}
