package watchers

import (
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
	RequiredL1Finalizer() NetworkID // returns NetworkID of the L1 Finalizer that should be used for this Watcher.
	SetL1Finalizer(l1finalizer interfaces.L1Finalizer)
	Create(
		msgC chan<- *common.MessagePublication,
		obsvReqC <-chan *gossipv1.ObservationRequest,
		queryReqC <-chan *query.PerChainQueryInternal,
		queryResponseC chan<- *query.PerChainQueryResponseInternal,
		setC chan<- *common.GuardianSet,
		env common.Environment,
	) (interfaces.L1Finalizer, supervisor.Runnable, interfaces.Reobserver, error)
}

var (
	ReobservationsByChain = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_reobservations_by_chain",
			Help: "Total number of reobservations completed by chain and observation type",
		}, []string{"chain", "type"})
)
