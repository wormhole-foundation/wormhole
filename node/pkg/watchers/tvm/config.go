package tvm

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xssnick/tonutils-go/address"
)

type WatcherConfig struct {
	NetworkID       watchers.NetworkID // human readable name
	ChainID         vaa.ChainID        // ChainID
	ContractAddress string             // hex representation of the contract address
	ConfigURL       string
}

func (wc *WatcherConfig) GetNetworkID() watchers.NetworkID {
	return wc.NetworkID
}

func (wc *WatcherConfig) GetChainID() vaa.ChainID {
	return wc.ChainID
}

//nolint:unparam // interfaces.Reobserver is always nil here but the return type is required to satisfy the interface.
func (wc *WatcherConfig) Create(
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
	_ <-chan *query.PerChainQueryInternal,
	_ chan<- *query.PerChainQueryResponseInternal,
	_ chan<- *common.GuardianSet,
	env common.Environment,
) (supervisor.Runnable, interfaces.Reobserver, error) {
	contractAddress, err := address.ParseAddr(wc.ContractAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse contract address: %w", err)
	}

	return NewWatcher(wc.ChainID, wc.ConfigURL, 0, contractAddress, msgC, obsvReqC).Run, nil, nil
}
