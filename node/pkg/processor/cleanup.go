// nolint:unparam // this will be refactored in https://github.com/wormhole-foundation/wormhole/pull/1953
package processor

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	// retryLimitOurs defines how long this Guardian will keep an observation in the local state before discarding it.
	// Oservations from other Guardians can take up to 24h to arrive if they are held in their Governor. Therefore, this value should be greater than 24h.
	retryLimitOurs    = time.Hour * 30
	retryLimitNotOurs = time.Hour
)

var (
	FirstRetryMinWait = time.Minute * 5
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
				if p.haveSignedVAA(*db.VaaIDFromVAA(&ourVaa.VAA)) {
					// If we have a stored quorum VAA, we can safely expire the state.
					//
					// This is a rare case, and we can safely expire the state, since we
					// have a quorum VAA.
					p.logger.Info("Expiring late VAA",
						zap.String("message_id", ourVaa.VAA.MessageID()),
						zap.String("digest", hash),
						zap.Duration("delta", delta),
					)
					aggregationStateLate.Inc()
					delete(p.state.signatures, hash)
					continue
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
			quorum := hasSigs >= gs.Quorum()

			var chain vaa.ChainID
			if s.ourObservation != nil {
				chain = s.ourObservation.GetEmitterChain()
			}

			if p.logger.Level().Enabled(zapcore.DebugLevel) {
				p.logger.Debug("observation considered settled",
					zap.String("message_id", s.LoggingID()),
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Int("have_sigs", hasSigs),
					zap.Int("required_sigs", gs.Quorum()),
					zap.Bool("quorum", quorum),
					zap.Stringer("emitter_chain", chain),
				)
			}

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
			if p.logger.Level().Enabled(zapcore.DebugLevel) {
				p.logger.Debug("expiring submitted observation",
					zap.String("message_id", s.LoggingID()),
					zap.String("digest", hash),
					zap.Duration("delta", delta),
				)
			}
			delete(p.state.signatures, hash)
			aggregationStateExpiration.Inc()
		case !s.submitted && ((s.ourObs != nil && delta > retryLimitOurs) || (s.ourObs == nil && delta > retryLimitNotOurs)):
			// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
			p.logger.Info("expiring unsubmitted observation after exhausting retries",
				zap.String("message_id", s.LoggingID()),
				zap.String("digest", hash),
				zap.Duration("delta", delta),
				zap.Bool("weObserved", s.ourObs != nil),
			)
			delete(p.state.signatures, hash)
			aggregationStateTimeout.Inc()
		case !s.submitted && delta >= FirstRetryMinWait && time.Since(s.nextRetry) >= 0:
			// Poor observation has been unsubmitted for five minutes - clearly, something went wrong.
			// If we have previously submitted an observation, and it was reliable, we can make another attempt to get
			// it over the finish line by sending a re-observation request to the network and rebroadcasting our
			// sig. If we do not have an observation, it means we either never observed it, or it got
			// revived by a malfunctioning guardian node, in which case, we can't do anything about it
			// and just delete it to keep our state nice and lean.
			if s.ourObs != nil {
				// Unreliable observations cannot be resubmitted and can be considered failed after 5 minutes
				if !s.ourObservation.IsReliable() {
					p.logger.Info("expiring unsubmitted unreliable observation",
						zap.String("message_id", s.LoggingID()),
						zap.String("digest", hash),
						zap.Duration("delta", delta),
					)
					delete(p.state.signatures, hash)
					aggregationStateTimeout.Inc()
					break
				}

				// Reobservation requests should not be resubmitted but we will keep waiting for more observations.
				if s.ourObservation.IsReobservation() {
					if p.logger.Level().Enabled(zapcore.DebugLevel) {
						p.logger.Debug("not submitting reobservation request for reobservation",
							zap.String("message_id", s.LoggingID()),
							zap.String("digest", hash),
							zap.Duration("delta", delta),
						)
					}
					break
				}

				// If we have already stored this VAA, there is no reason for us to request reobservation.
				alreadyInDB, err := p.signedVaaAlreadyInDB(hash, s)
				if err != nil {
					p.logger.Error("failed to check if observation is already in DB, requesting reobservation",
						zap.String("message_id", s.LoggingID()),
						zap.String("hash", hash),
						zap.Error(err))
				}

				if alreadyInDB {
					if p.logger.Level().Enabled(zapcore.DebugLevel) {
						p.logger.Debug("observation already in DB, not requesting reobservation",
							zap.String("message_id", s.LoggingID()),
							zap.String("digest", hash),
						)
					}
				} else {
					p.logger.Info("resubmitting observation",
						zap.String("message_id", s.LoggingID()),
						zap.String("digest", hash),
						zap.Duration("delta", delta),
						zap.String("firstObserved", s.firstObserved.String()),
						zap.Int("numSignatures", len(s.signatures)),
					)
					req := &gossipv1.ObservationRequest{
						ChainId: uint32(s.ourObservation.GetEmitterChain()),
						TxHash:  s.txHash,
					}
					if err := common.PostObservationRequest(p.obsvReqSendC, req); err != nil {
						p.logger.Warn("failed to broadcast re-observation request", zap.String("message_id", s.LoggingID()), zap.Error(err))
					}
					if s.ourMsg != nil {
						// This is the case for immediately published messages (as well as anything still pending from before the cutover).
						p.gossipAttestationSendC <- s.ourMsg
					} else {
						p.postObservationToBatch(s.ourObs)
					}
					s.retryCtr++
					s.nextRetry = time.Now().Add(nextRetryDuration(s.retryCtr))
					aggregationStateRetries.Inc()
				}
			} else {
				// For nil state entries, we log the quorum to determine whether the
				// network reached consensus without us. We don't know the correct guardian
				// set, so we simply use the most recent one.
				hasSigs := len(s.signatures)

				if p.logger.Level().Enabled(zapcore.DebugLevel) {
					p.logger.Debug("expiring unsubmitted nil observation",
						zap.String("message_id", s.LoggingID()),
						zap.String("digest", hash),
						zap.Duration("delta", delta),
						zap.Int("have_sigs", hasSigs),
						zap.Int("required_sigs", p.gs.Quorum()),
						zap.Bool("quorum", hasSigs >= p.gs.Quorum()),
					)
				}
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

// signedVaaAlreadyInDB checks if the VAA is already in the DB. If it is, it makes sure the hash matches.
func (p *Processor) signedVaaAlreadyInDB(hash string, s *state) (bool, error) {
	if s.ourObservation == nil {
		p.logger.Debug("unable to check if VAA is already in DB, no observation", zap.String("digest", hash))
		return false, nil
	}

	msgId := s.ourObservation.MessageID()
	vaaID, err := db.VaaIDFromString(msgId)
	if err != nil {
		return false, fmt.Errorf(`failed to generate VAA ID from message id "%s": %w`, s.ourObservation.MessageID(), err)
	}

	// If the VAA is waiting to be written to the DB, use that version. Otherwise use the DB.
	v := p.getVaaFromUpdateMap(msgId)
	if v == nil {
		vb, err := p.db.GetSignedVAABytes(*vaaID)
		if err != nil {
			if errors.Is(err, db.ErrVAANotFound) {
				if p.logger.Level().Enabled(zapcore.DebugLevel) {
					p.logger.Debug("VAA not in DB",
						zap.String("message_id", s.ourObservation.MessageID()),
						zap.String("digest", hash),
					)
				}
				return false, nil
			}

			return false, fmt.Errorf(`failed to look up message id "%s" in db: %w`, s.ourObservation.MessageID(), err)
		}

		v, err = vaa.Unmarshal(vb)
		if err != nil {
			return false, fmt.Errorf("failed to unmarshal VAA: %w", err)
		}
	}

	oldHash := hex.EncodeToString(v.SigningDigest().Bytes())
	if hash != oldHash {
		if p.logger.Core().Enabled(zapcore.DebugLevel) {
			p.logger.Debug("VAA already in DB but hash is different",
				zap.String("message_id", s.ourObservation.MessageID()),
				zap.String("old_hash", oldHash),
				zap.String("new_hash", hash))
		}
		return false, fmt.Errorf("hash mismatch in_db: %s, new: %s", oldHash, hash)
	}

	return true, nil
}
