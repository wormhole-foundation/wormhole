// nolint:unparam // this will be refactored in https://github.com/wormhole-foundation/wormhole/pull/1953
package processor3

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
			Name: "wormhole_aggregation_state_entries3",
			Help: "Current number of aggregation state entries (including unexpired succeed ones)",
		})
	aggregationStateExpiration = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_expirations_total3",
			Help: "Total number of expired submitted aggregation states",
		})
	aggregationStateLate = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_late_total3",
			Help: "Total number of late aggregation states (cluster achieved consensus without us)",
		})
	aggregationStateTimeout = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_timeout_total3",
			Help: "Total number of aggregation states expired due to timeout after exhausting retries",
		})
	aggregationStateRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_retries_total3",
			Help: "Total number of aggregation states queued for resubmission",
		})
	aggregationStateUnobserved = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_aggregation_state_unobserved_total3",
			Help: "Total number of aggregation states expired due to no matching local message observations",
		})
)

const (
	settlementTime    = time.Second * 30
	retryTime         = time.Minute * 5
	retryLimitOurs    = time.Hour * 24
	retryLimitNotOurs = time.Hour
)

// handleCleanup handles periodic retransmissions and cleanup of observations
func (p *Processor) handleCleanup(ctx context.Context, logger *zap.Logger) {
	logger.Info("aggregation state summary", zap.Int("cached", len(p.state)))
	aggregationStateEntries.Set(float64(len(p.state)))

	for hash, s := range p.state {
		delta := time.Since(s.firstObserved)

		if !s.submitted && delta > settlementTime {
			// Expire pending VAAs post settlement time if we have a stored quorum VAA.
			//
			// This occurs when we observed a message after the cluster has already reached
			// consensus on it, causing us to never achieve quorum.
			if s.msg != nil {
				if _, err := p.getSignedVAA(*db.VaaIDFromVAA(s.msg.CreateVAA(0))); err == nil {
					// If we have a stored quorum VAA, we can safely expire the state.
					//
					// This is a rare case, and we can safely expire the state, since we
					// have a quorum VAA.
					logger.Debug("Expiring late VAA", zap.String("digest", hash), zap.Duration("delta", delta))
					aggregationStateLate.Inc()
					delete(p.state, hash)
					continue
				}
			}
		}

		switch {
		case s.submitted:
			// delete submitted observations from processor state
			delete(p.state, hash)
			aggregationStateExpiration.Inc()
		case !s.submitted && ((s.msg != nil && delta > retryLimitOurs) || (s.msg == nil && delta > retryLimitNotOurs)):
			// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
			logger.Info("expiring unsubmitted observation after exhausting retries", zap.String("digest", hash), zap.Duration("delta", delta), zap.Bool("weObserved", s.msg != nil))
			delete(p.state, hash)
			aggregationStateTimeout.Inc()
		case !s.submitted && delta.Minutes() >= 5 && time.Since(s.lastRetry) >= retryTime:
			// Poor observation has been unsubmitted for five minutes - clearly, something went wrong.
			// If we have previously submitted an observation, and it was reliable, we can make another attempt to get
			// it over the finish line by sending a re-observation request to the network and rebroadcasting our
			// sig. If we do not have an observation, it means we either never observed it, or it got
			// revived by a malfunctioning guardian node, in which case, we can't do anything about it
			// and just delete it to keep our state nice and lean.
			if s.msg != nil {
				// Unreliable observations cannot be resubmitted and can be considered failed after 5 minutes
				if !s.msg.Unreliable {
					logger.Info("expiring unsubmitted unreliable observation", zap.String("digest", hash), zap.Duration("delta", delta))
					delete(p.state, hash)
					aggregationStateTimeout.Inc()
					break
				}
				logger.Info("resubmitting observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.String("firstObserved", s.firstObserved.String()),
				)
				req := &gossipv1.ObservationRequest{
					ChainId: uint32(s.msg.EmitterChain),
					TxHash:  s.msg.TxHash[:],
				}
				if err := common.PostObservationRequest(p.obsvReqSendC, req); err != nil {
					logger.Warn("failed to broadcast re-observation request", zap.Error(err))
				}
				s.lastRetry = time.Now()
				aggregationStateRetries.Inc()
			} else {
				// For nil state entries, we log the quorum to determine whether the
				// network reached consensus without us. We don't know the correct guardian
				// set, so we simply use the most recent one.
				hasSigs := len(s.signatures)
				wantSigs := vaa.CalculateQuorum(len(p.gst.Get().Keys))

				logger.Info("expiring unsubmitted nil observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Int("have_sigs", hasSigs),
					zap.Int("required_sigs", wantSigs),
					zap.Bool("quorum", hasSigs >= wantSigs),
				)
				delete(p.state, hash)
				aggregationStateUnobserved.Inc()
			}
		}
	}

	// Clean up old pythnet VAAs.
	p.pythVaas.deleteBefore(time.Now().Add(-time.Hour))
}
