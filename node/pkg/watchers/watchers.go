package watchers

import (
	"sync"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers/interfaces"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// NetworkID is a unique identifier of a watcher that is used to link watchers together for the purpose of L1 Finalizers.
// This is different from vaa.ChainID because there could be multiple watchers for a single chain (e.g. solana-confirmed and solana-finalized)
type NetworkID string

type WatcherConfig interface {
	GetNetworkID() NetworkID
	GetChainID() vaa.ChainID
	Create(
		msgC chan<- *common.MessagePublication,
		obsvReqC <-chan *gossipv1.ObservationRequest,
		queryReqC <-chan *query.PerChainQueryInternal,
		queryResponseC chan<- *query.PerChainQueryResponseInternal,
		setC chan<- *common.GuardianSet,
		env common.Environment,
	) (supervisor.Runnable, interfaces.Reobserver, error)
}

var (
	ReobservationsByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_reobservations_by_chain",
			Help: "Total number of reobservations completed by chain and observation type",
		}, []string{"chain", "type"})
)

var (
	rpcURLsMu sync.RWMutex
	rpcURLs   = map[vaa.ChainID][]string{}
)

// RegisterRPCURL records the RPC URL used by a watcher for a chain.
func RegisterRPCURL(chainID vaa.ChainID, rpcURL string) {
	if rpcURL == "" {
		return
	}

	rpcURLsMu.Lock()
	defer rpcURLsMu.Unlock()
	for _, existingURL := range rpcURLs[chainID] {
		if existingURL == rpcURL {
			return
		}
	}
	rpcURLs[chainID] = append(rpcURLs[chainID], rpcURL)
}

// RPCURLs returns the RPC URLs registered for a chain.
func RPCURLs(chainID vaa.ChainID) []string {
	rpcURLsMu.RLock()
	defer rpcURLsMu.RUnlock()
	chainRPCURLs := rpcURLs[chainID]
	if len(chainRPCURLs) == 0 {
		return nil
	}
	ret := make([]string, len(chainRPCURLs))
	copy(ret, chainRPCURLs)
	return ret
}
