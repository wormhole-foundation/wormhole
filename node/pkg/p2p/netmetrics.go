package p2p

import (
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	wormholeNetworkNodeHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_network_node_height",
			Help: "Network height of the given guardian node per network",
		}, []string{"guardian_addr", "node_id", "node_name", "network"})
	wormholeNetworkNodeErrors = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_network_node_errors_count",
			Help: "Number of errors the given guardian node encountered per network",
		}, []string{"guardian_addr", "node_id", "node_name", "network"})
)

func collectNodeMetrics(addr common.Address, peerId peer.ID, hb *gossipv1.Heartbeat) {
	for _, n := range hb.Networks {
		if n == nil {
			continue
		}

		chain := vaa.ChainID(n.Id)

		wormholeNetworkNodeHeight.WithLabelValues(
			addr.Hex(), peerId.Pretty(), hb.NodeName, chain.String()).Set(float64(n.Height))

		wormholeNetworkNodeErrors.WithLabelValues(
			addr.Hex(), peerId.Pretty(), hb.NodeName, chain.String()).Set(float64(n.ErrorCount))
	}
}
