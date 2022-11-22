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
	aggregationBatchStateEntries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "wormhole_aggregation_batch_state_entries",
			Help: "Current number of BatchVaa aggregation state entries (including unexpired succeed ones)",
		})
	aggregationBatchStateExpiration = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_batch_state_expirations_total",
			Help: "Total number of expired submitted BatchVAA aggregation states",
		})
	aggregationBatchStateLate = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_batch_state_late_total",
			Help: "Total number of late BatchVAA aggregation states (cluster achieved consensus without us)",
		})
	aggregationBatchStateTimeout = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_batch_state_timeout_total",
			Help: "Total number of BatchVAA aggregation states expired due to timeout after exhausting retries",
		})
	aggregationBatchStateRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_batch_state_retries_total",
			Help: "Total number of BatchVAA aggregation states queued for resubmission",
		})
	aggregationBatchStateUnobserved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_batch_state_unobserved_total",
			Help: "Total number of BatchVAA aggregation states expired due to no matching local message observations",
		})
	aggregationBatchStateFulfillment = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_batch_state_settled_signatures_total",
			Help: "Total number of BatchVAA signatures produced by a validator, counted after waiting a fixed amount of time",
		}, []string{"addr", "origin", "status"})
)

// deleteBatchVaaState removes key/values from Processor state objects
func deleteBatchVaaState(p *Processor, s *batchState, batchHash string) {
	delete(p.state.batchSignatures, batchHash)
	batchID := s.ourObservation.GetBatchID()
	delete(p.state.batches, batchID)
	delete(p.state.batchMessages, batchID)
}

// handleCleanup handles periodic retransmissions and cleanup of observations
func (p *Processor) handleBatchCleanup(ctx context.Context) {
	if !p.batchVAAEnabled {
		// respect the feature flag
		return
	}

	p.logger.Info("aggregation batch state summary", zap.Int("cached", len(p.state.batchSignatures)))
	aggregationBatchStateEntries.Set(float64(len(p.state.signatures)))

	// First, loop through the batches we've seen and evaluate them for completion.
	// Batches are otherwise only evaluated for progress toward completion when observed messages reach quorum.
	// This extra check is an effort to settle any batches that are pending due to the
	// nuances of observing chain state - missed messages, late observations, etc.
	// This check handles a potential edge case in which some of the messages within a batch
	// are not not observed directly, but are received from gossip.
	for batchID := range p.state.batches {
		p.evaluateBatchProgress(&batchID)
	}

	for hash, s := range p.state.batchSignatures {
		delta := time.Since(s.firstObserved)

		if !s.submitted && s.ourObservation != nil && delta > settlementTime {
			// Expire pending VAAs post settlement time if we have a stored quorum VAA.
			//
			// This occurs when we observed a message after the cluster has already reached
			// consensus on it, causing us to never achieve quorum.
			if ourVaa, ok := s.ourObservation.(*Batch); ok {
				if _, err := p.db.GetSignedBatchBytes(ourVaa.BatchID); err == nil {
					// If we have a stored quorum VAA, we can safely expire the state.
					//
					// This is a rare case, and we can safely expire the state, since we
					// have a quorum VAA.
					p.logger.Info("Expiring late BatchVAA", zap.String("digest", hash), zap.Duration("delta", delta))
					aggregationBatchStateLate.Inc()
					deleteBatchVaaState(p, s, hash)
					continue
				} else if err != db.ErrVAANotFound {
					p.logger.Error("failed to look up BatchVAA in database",
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
			chain := s.ourObservation.GetEmitterChain()

			p.logger.Info("batch observation considered settled",
				zap.String("digest", hash),
				zap.Duration("delta", delta),
				zap.Int("have_sigs", hasSigs),
				zap.Int("required_sigs", wantSigs),
				zap.Bool("quorum", quorum),
				zap.Stringer("emitter_chain", chain),
			)

			for _, k := range gs.Keys {
				if _, ok := s.signatures[k]; ok {
					aggregationBatchStateFulfillment.WithLabelValues(k.Hex(), s.source, "present").Inc()
				} else {
					aggregationBatchStateFulfillment.WithLabelValues(k.Hex(), s.source, "missing").Inc()
				}
			}
		case s.submitted && delta.Hours() >= 1:
			// We could delete submitted observations right away, but then we'd lose context about additional (late)
			// observation that come in. Therefore, keep it for a reasonable amount of time.
			// If a very late observation arrives after cleanup, a nil aggregation state will be created
			// and then expired after a while (as noted in observation.go, this can be abused by a byzantine guardian).
			p.logger.Info("expiring submitted batch observation", zap.String("digest", hash), zap.Duration("delta", delta))
			deleteBatchVaaState(p, s, hash)
			aggregationBatchStateExpiration.Inc()
		case !s.submitted && ((s.ourMsg != nil && s.retryCount >= 14400 /* 120 hours */) || (s.ourMsg == nil && s.retryCount >= 10 /* 5 minutes */)):
			// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
			p.logger.Info("expiring unsubmitted batch observation after exhausting retries", zap.String("digest", hash), zap.Duration("delta", delta))
			deleteBatchVaaState(p, s, hash)
			aggregationBatchStateTimeout.Inc()
		case !s.submitted && delta.Minutes() >= 5:
			// Poor observation has been unsubmitted for five minutes - clearly, something went wrong.
			// If we have previously submitted an observation, we can make another attempt to get it over
			// the finish line by sending a re-observation request to the network and rebroadcasting our
			// sig. If we do not have an observation, it means we either never observed it, or it got
			// revived by a malfunctioning guardian node, in which case, we can't do anything about it
			// and just delete it to keep our state nice and lean.
			if s.ourMsg != nil {
				p.logger.Info("resubmitting batch observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Uint("retry", s.retryCount))
				req := &gossipv1.ObservationRequest{
					ChainId: uint32(s.ourObservation.GetEmitterChain()),
					TxHash:  s.txHash,
				}
				if err := common.PostObservationRequest(p.obsvReqSendC, req); err != nil {
					p.logger.Warn("failed to broadcast batch re-observation request", zap.Error(err))
				}
				p.sendC <- s.ourMsg
				s.retryCount += 1
				aggregationBatchStateRetries.Inc()
			} else {
				// For nil state entries, we log the quorum to determine whether the
				// network reached consensus without us. We don't know the correct guardian
				// set, so we simply use the most recent one.
				hasSigs := len(s.signatures)
				wantSigs := vaa.CalculateQuorum(len(p.gs.Keys))

				p.logger.Info("expiring unsubmitted nil batch observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Int("have_sigs", hasSigs),
					zap.Int("required_sigs", wantSigs),
					zap.Bool("quorum", hasSigs >= wantSigs),
				)
				deleteBatchVaaState(p, s, hash)
				aggregationBatchStateUnobserved.Inc()
			}
		}
	}
}
