package processor

import (
	"context"
	"encoding/hex"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

var (
	vaaInjectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_vaa_injections_total",
			Help: "Total number of injected VAA queued for broadcast",
		})
)

// handleInjection processes a pre-populated VAA injected locally.
func (p *Processor) handleInjection(ctx context.Context, v *vaa.VAA) {
	// Generate digest of the unsigned VAA.
	digest := v.SigningMsg()

	// The internal originator is responsible for logging the full VAA, just log the digest here.
	supervisor.Logger(ctx).Info("signing injected VAA",
		zap.String("digest", hex.EncodeToString(digest.Bytes())))

	// Sign the digest using our node's guardian key.
	s, err := crypto.Sign(digest.Bytes(), p.gk)
	if err != nil {
		panic(err)
	}

	p.logger.Info("observed and signed injected VAA",
		zap.String("digest", hex.EncodeToString(digest.Bytes())),
		zap.String("signature", hex.EncodeToString(s)))

	vaaInjectionsTotal.Inc()
	p.broadcastSignature(&VAA{VAA: *v}, s, nil)
}
