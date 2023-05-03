// nolint:unparam // this will be refactored in https://github.com/wormhole-foundation/wormhole/pull/1953
package processor

import (
	"context"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

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
	aggregationStateLate = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_late_total",
			Help: "Total number of late aggregation states (cluster achieved consensus without us)",
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

const (
	settlementTime = time.Second * 30
	retryTime      = time.Minute * 5
)

// handleCleanup handles periodic retransmissions and cleanup of observations
func (p *Processor) handleCleanup(ctx context.Context) {
	p.logger.Info("aggregation state summary", zap.Int("cached", len(p.state.signatures)))
	aggregationStateEntries.Set(float64(len(p.state.signatures)))

	for hash, s := range p.state.signatures {
		delta := time.Since(s.firstObserved)

		if !s.submitted && s.ourObservation != nil && delta > settlementTime {
			// Expire pending VAAs post settlement time if we have a stored quorum VAA.
			//
			// This occurs when we observed a message after the cluster has already reached
			// consensus on it, causing us to never achieve quorum.
			if ourVaa, ok := s.ourObservation.(*VAA); ok {
				if _, err := p.getSignedVAA(*db.VaaIDFromVAA(&ourVaa.VAA)); err == nil {
					// If we have a stored quorum VAA, we can safely expire the state.
					//
					// This is a rare case, and we can safely expire the state, since we
					// have a quorum VAA.
					p.logger.Info("Expiring late VAA", zap.String("digest", hash), zap.Duration("delta", delta))
					aggregationStateLate.Inc()
					delete(p.state.signatures, hash)
					continue
				} else if err != db.ErrVAANotFound {
					p.logger.Error("failed to look up VAA in database",
						zap.String("digest", hash),
						zap.Error(err),
					)
				}
			}
		}

		switch {
		case !s.settled && delta > settlementTime:
			// After 30 seconds, the observation is considered settled - it's unlikely that more observations will
			// arrive, barring special circumstances. This is a better time to count misses than submission,
			// because we submit right when we quorum rather than waiting for all observations to arrive.
			s.settled = true

			// Use either the most recent (in case of a observation we haven't seen) or stored gs, if available.
			var gs *common.GuardianSet
			if s.gs != nil {
				gs = s.gs
			} else {
				gs = p.gs
			}

			hasSigs := len(s.signatures)
			wantSigs := vaa.CalculateQuorum(len(gs.Keys))
			quorum := hasSigs >= wantSigs

			var chain vaa.ChainID
			if s.ourObservation != nil {
				chain = s.ourObservation.GetEmitterChain()
			}

			p.logger.Info("observation considered settled",
				zap.String("digest", hash),
				zap.Duration("delta", delta),
				zap.Int("have_sigs", hasSigs),
				zap.Int("required_sigs", wantSigs),
				zap.Bool("quorum", quorum),
				zap.Stringer("emitter_chain", chain),
			)

			for _, k := range gs.Keys {
				if _, ok := s.signatures[k]; ok {
					aggregationStateFulfillment.WithLabelValues(k.Hex(), s.source, "present").Inc()
				} else {
					aggregationStateFulfillment.WithLabelValues(k.Hex(), s.source, "missing").Inc()
				}
			}
		case s.submitted && delta.Hours() >= 1:
			// We could delete submitted observations right away, but then we'd lose context about additional (late)
			// observation that come in. Therefore, keep it for a reasonable amount of time.
			// If a very late observation arrives after cleanup, a nil aggregation state will be created
			// and then expired after a while (as noted in observation.go, this can be abused by a byzantine guardian).
			p.logger.Info("expiring submitted observation", zap.String("digest", hash), zap.Duration("delta", delta))
			delete(p.state.signatures, hash)
			aggregationStateExpiration.Inc()
		case !s.submitted && ((s.ourMsg != nil && s.retryCount >= 14400 /* 120 hours */) || (s.ourMsg == nil && s.retryCount >= 10 /* 5 minutes */)):
			// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
			p.logger.Info("expiring unsubmitted observation after exhausting retries", zap.String("digest", hash), zap.Duration("delta", delta))
			delete(p.state.signatures, hash)
			aggregationStateTimeout.Inc()
		case !s.submitted && delta.Minutes() >= 5 && time.Since(s.lastRetry) >= retryTime:
			// Poor observation has been unsubmitted for five minutes - clearly, something went wrong.
			// If we have previously submitted an observation, and it was reliable, we can make another attempt to get
			// it over the finish line by sending a re-observation request to the network and rebroadcasting our
			// sig. If we do not have an observation, it means we either never observed it, or it got
			// revived by a malfunctioning guardian node, in which case, we can't do anything about it
			// and just delete it to keep our state nice and lean.
			if s.ourMsg != nil {
				// Unreliable observations cannot be resubmitted and can be considered failed after 5 minutes
				if !s.ourObservation.IsReliable() {
					p.logger.Info("expiring unsubmitted unreliable observation", zap.String("digest", hash), zap.Duration("delta", delta))
					delete(p.state.signatures, hash)
					aggregationStateTimeout.Inc()
					break
				}
				p.logger.Info("resubmitting observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Uint("retry", s.retryCount))
				req := &gossipv1.ObservationRequest{
					ChainId: uint32(s.ourObservation.GetEmitterChain()),
					TxHash:  s.txHash,
				}
				if err := common.PostObservationRequest(p.obsvReqSendC, req); err != nil {
					p.logger.Warn("failed to broadcast re-observation request", zap.Error(err))
				}
				p.gossipSendC <- s.ourMsg
				s.retryCount += 1
				s.lastRetry = time.Now()
				aggregationStateRetries.Inc()
			} else {
				// For nil state entries, we log the quorum to determine whether the
				// network reached consensus without us. We don't know the correct guardian
				// set, so we simply use the most recent one.
				hasSigs := len(s.signatures)
				wantSigs := vaa.CalculateQuorum(len(p.gs.Keys))

				p.logger.Info("expiring unsubmitted nil observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Int("have_sigs", hasSigs),
					zap.Int("required_sigs", wantSigs),
					zap.Bool("quorum", hasSigs >= wantSigs),
				)
				delete(p.state.signatures, hash)
				aggregationStateUnobserved.Inc()
			}
		}
	}

	// Clean up old pythnet VAAs.
	oldestTime := time.Now().Add(-time.Hour)
	for key, pe := range p.pythnetVaas {
		if pe.updateTime.Before(oldestTime) {
			delete(p.pythnetVaas, key)
		}
	}
}
