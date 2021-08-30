package processor

import (
	"context"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"time"

	"go.uber.org/zap"
)

var (
	aggregationStateEntries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_aggregation_state_entries",
			Help: "Current number of aggregation state entries (including unexpired succeed ones)",
		})
	aggregationStateExpiration = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_expirations_total",
			Help: "Total number of expired submitted aggregation states",
		})
	aggregationStateTimeout = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_timeout_total",
			Help: "Total number of aggregation states expired due to timeout after exhausting retries",
		})
	aggregationStateRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_retries_total",
			Help: "Total number of aggregation states queued for resubmission",
		})
	aggregationStateUnobserved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_unobserved_total",
			Help: "Total number of aggregation states expired due to no matching local message observations",
		})
	aggregationStateFulfillment = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_settled_signatures_total",
			Help: "Total number of signatures produced by a validator, counted after waiting a fixed amount of time",
		}, []string{"addr", "origin", "status"})
)

// handleCleanup handles periodic retransmissions and cleanup of VAAs
func (p *Processor) handleCleanup(ctx context.Context) {
	p.logger.Info("aggregation state summary", zap.Int("cached", len(p.state.vaaSignatures)))
	aggregationStateEntries.Set(float64(len(p.state.vaaSignatures)))

	for hash, s := range p.state.vaaSignatures {
		delta := time.Since(s.firstObserved)

		switch {
		case !s.settled && delta.Seconds() >= 30:
			// After 30 seconds, the VAA is considered settled - it's unlikely that more observations will
			// arrive, barring special circumstances. This is a better time to count misses than submission,
			// because we submit right when we quorum rather than waiting for all observations to arrive.
			s.settled = true
			p.logger.Info("VAA considered settled", zap.String("digest", hash))

			// Use either the most recent (in case of a VAA we haven't seen) or stored gs, if available.
			var gs *common.GuardianSet
			if s.gs != nil {
				gs = s.gs
			} else {
				gs = p.gs
			}

			for _, k := range gs.Keys {
				if _, ok := s.signatures[k]; ok {
					aggregationStateFulfillment.WithLabelValues(k.Hex(), s.source, "present").Inc()
				} else {
					aggregationStateFulfillment.WithLabelValues(k.Hex(), s.source, "missing").Inc()
				}
			}
		case s.submitted && delta.Hours() >= 1:
			// We could delete submitted VAAs right away, but then we'd lose context about additional (late)
			// observation that come in. Therefore, keep it for a reasonable amount of time.
			// If a very late observation arrives after cleanup, a nil aggregation state will be created
			// and then expired after a while (as noted in observation.go, this can be abused by a byzantine guardian).
			p.logger.Info("expiring submitted VAA", zap.String("digest", hash), zap.Duration("delta", delta))
			delete(p.state.vaaSignatures, hash)
			aggregationStateExpiration.Inc()
		case !s.submitted && s.retryCount >= 10:
			// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
			p.logger.Info("expiring unsubmitted VAA after exhausting retries", zap.String("digest", hash), zap.Duration("delta", delta))
			delete(p.state.vaaSignatures, hash)
			aggregationStateTimeout.Inc()
		case !s.submitted && delta.Minutes() >= 5:
			// Poor VAA has been unsubmitted for five minutes - clearly, something went wrong.
			// If we have previously submitted an observation, we can make another attempt to get it over
			// the finish line by rebroadcasting our sig. If we do not have a VAA, it means we either never observed it,
			// or it got revived by a malfunctioning guardian node, in which case, we can't do anything
			// about it and just delete it to keep our state nice and lean.
			if s.ourMsg != nil {
				p.logger.Info("resubmitting VAA observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Int("retry", 1))
				p.sendC <- s.ourMsg
				s.retryCount += 1
				aggregationStateRetries.Inc()
			} else {
				p.logger.Info("expiring unsubmitted nil VAA", zap.String("digest", hash), zap.Duration("delta", delta))
				delete(p.state.vaaSignatures, hash)
				aggregationStateUnobserved.Inc()
			}
		}
	}
}
