// EEVAA processing code
package processor

import (
	"context"
	"encoding/hex"
	"fmt"
	bridge_common "github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"time"

	"github.com/certusone/wormhole/bridge/pkg/terra"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

var (
	// SECURITY: source_chain/target_chain are untrusted uint8 values. An attacker could cause a maximum of 255**2 label
	// pairs to be created, which is acceptable.

	eevaasObservedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eevaas_observed_total",
			Help: "Total number of EEVAAs received on-chain",
		},
		[]string{"source_chain", "target_chain"})

	eevaasSignedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_eevaas_signed_total",
			Help: "Total number of EEVAAs that were successfully signed",
		},
		[]string{"source_chain", "target_chain"})
)

func init() {
	prometheus.MustRegister(lockupsObservedTotal)
	prometheus.MustRegister(lockupsSignedTotal)
}

func (p *Processor) handleEEVAA(ctx context.Context, k *common.EEVAA) {
}
