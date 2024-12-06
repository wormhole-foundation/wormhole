package tss

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/yossigi/tss-lib/v2/common"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"go.uber.org/zap"
)

var (
	// counter of expired. and state it might be bad due to the distributed nature of the system.
	// Same with inProgressSigs.

	sigProducedCntr = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_tss_signature_produced",
			Help: "total number of tss signatures produced",
		}, []string{"chain_name"}, //followed example from ccq
	)

	tooManySimulSigsErrCntr = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_too_many_somultaneous_signatures",
			Help: "total number of tss signing requests that were rejected due to too many simultaneous signature requests",
		},
	)

	activeGuardiansByChain = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_tss_active_guardians_by_chain",
			Help: "Gauge showing the current number of active guardians by chain",
		}, []string{"chain_name"}, //followed example from ccq
	)

	sigLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "wormhole_tss_signature_latency",
			Help: "Histogram of the times taken to produce a signature",
		}, []string{"chain_name"}, //followed example from ccq
	)

	signatureEndingWithError = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_tss_signature_ending_with_error",
			Help: "total number of tss signatures that ended with an error",
		},
	)
)

// used for collecting metrics.
type signatureMetadata struct {
	timeOfCreation time.Time
}

func (t *Engine) createSignatureMetrics(vaaDigest []byte, chainID vaa.ChainID) {
	key := intoSigKey(party.Digest(vaaDigest), chainID)
	t.SignatureMetrics.Store(key, &signatureMetadata{
		timeOfCreation: time.Now(),
	})
}

func (t *Engine) sigMetricDone(trackid *common.TrackingID, hadIssue bool) {
	key := trackingIdIntoSigKey(trackid)

	metrics, ok := t.loadMetric(key, trackid)
	if !ok {
		return
	}

	if hadIssue {
		signatureEndingWithError.Inc()

		return
	}

	chain := extractChainIDFromTrackingID(trackid)
	sigProducedCntr.
		WithLabelValues(chain.String()).
		Inc()

	latency := time.Since(metrics.timeOfCreation)
	sigLatency.WithLabelValues(chain.String()).Observe(float64(latency.Milliseconds()))

	t.SignatureMetrics.Delete(key)
}

func (t *Engine) loadMetric(key sigKey, trackid *common.TrackingID) (*signatureMetadata, bool) {
	tmp, loaded := t.SignatureMetrics.Load(key)
	if !loaded {
		return nil, false
	}

	metrics, ok := tmp.(*signatureMetadata)
	if !ok {
		t.logger.Error("signature metrics stored is of wrong type",
			zap.String("digest", fmt.Sprintf("%x", trackid.Digest)),
			zap.Any("metrics", tmp),
		)

		return nil, false
	}

	return metrics, true
}
