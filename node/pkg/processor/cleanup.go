// nolint:unparam // this will be refactored in https://github.com/wormhole-foundation/wormhole/pull/1953
package processor

import (
	"context"
	"encoding/hex"
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
	p.cleanupState()
	p.cleanupPythnetVaas()
}

// cleanupState walks through the aggregation state map and cleans up entries that are no longer needed. It grabs the state lock.
func (p *Processor) cleanupState() {
	p.state.signaturesLock.Lock()
	defer p.state.signaturesLock.Unlock()

	p.logger.Info("aggregation state summary", zap.Int("cached", len(p.state.signatures)))
	aggregationStateEntries.Set(float64(len(p.state.signatures)))

	for hash, s := range p.state.signatures {
		if shouldDelete := p.cleanUpStateEntry(hash, s); shouldDelete {
			delete(p.state.signatures, hash) // Can't use p.state.delete() because we're holding the lock.
		}
	}
}

// cleanUpStateEntry cleans up a single aggregation state entry. It grabs the lock for that entry. Returns true if the entry should be deleted.
func (p *Processor) cleanUpStateEntry(hash string, s *state) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

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
				p.logger.Info("Expiring late VAA", zap.String("digest", hash), zap.Duration("delta", delta))
				aggregationStateLate.Inc()
				return true
			}
		}
	}

	switch {
	case !s.settled && delta > settlementTime:
		// After 30 seconds, the observation is considered settled - it's unlikely that more observations will
		// arrive, barring special circumstances. This is a better time to count misses than submission,
		// because we submit right when we quorum rather than waiting for all observations to arrive.
		s.settled = true

		// Peg the appropriate settlement metric using the current guardian set. If we don't have a guardian set (extremely unlikely), we just won't peg the metric.
		gs := p.gst.Get()
		if gs == nil {
			return false
		}
		hasSigs := len(s.signatures)
		wantSigs := vaa.CalculateQuorum(len(gs.Keys))
		quorum := hasSigs >= wantSigs

		var chain vaa.ChainID
		if s.ourObservation != nil {
			chain = s.ourObservation.GetEmitterChain()
		}

		p.logger.Debug("observation considered settled",
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
		p.logger.Debug("expiring submitted observation", zap.String("digest", hash), zap.Duration("delta", delta))
		aggregationStateExpiration.Inc()
		return true
	case !s.submitted && ((s.ourMsg != nil && delta > retryLimitOurs) || (s.ourMsg == nil && delta > retryLimitNotOurs)):
		// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
		p.logger.Info("expiring unsubmitted observation after exhausting retries", zap.String("digest", hash), zap.Duration("delta", delta))
		aggregationStateTimeout.Inc()
		return true
	case !s.submitted && delta >= FirstRetryMinWait && time.Since(s.nextRetry) >= 0:
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
				aggregationStateTimeout.Inc()
				return true
			}

			// If we have already stored this VAA, there is no reason for us to request reobservation.
			alreadyInDB, err := p.signedVaaAlreadyInDB(hash, s)
			if err != nil {
				p.logger.Error("failed to check if observation is already in DB, requesting reobservation", zap.String("hash", hash), zap.Error(err))
			}

			if alreadyInDB {
				p.logger.Debug("observation already in DB, not requesting reobservation", zap.String("digest", hash))
			} else {
				p.logger.Info("resubmitting observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.String("firstObserved", s.firstObserved.String()),
				)
				req := &gossipv1.ObservationRequest{
					ChainId: uint32(s.ourObservation.GetEmitterChain()),
					TxHash:  s.txHash,
				}
				if err := common.PostObservationRequest(p.obsvReqSendC, req); err != nil {
					p.logger.Warn("failed to broadcast re-observation request", zap.Error(err))
				}
				p.gossipSendC <- s.ourMsg
				s.retryCtr++
				s.nextRetry = time.Now().Add(nextRetryDuration(s.retryCtr))
				aggregationStateRetries.Inc()
			}
		} else {
			// For nil state entries, we log the quorum to determine whether the
			// network reached consensus without us. We don't know the correct guardian
			// set, so we simply use the most recent one.
			gs := p.gst.Get()
			hasSigs := len(s.signatures)
			wantSigs := vaa.CalculateQuorum(len(gs.Keys))

			p.logger.Debug("expiring unsubmitted nil observation",
				zap.String("digest", hash),
				zap.Duration("delta", delta),
				zap.Int("have_sigs", hasSigs),
				zap.Int("required_sigs", wantSigs),
				zap.Bool("quorum", hasSigs >= wantSigs),
			)
			aggregationStateUnobserved.Inc()
			return true
		}
	}

	return false
}

// cleanupPythnetVaas deletes expired pythnet vaas.
func (p *Processor) cleanupPythnetVaas() {
	p.pythnetVaaLock.Lock()
	defer p.pythnetVaaLock.Unlock()
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

	vaaID, err := db.VaaIDFromString(s.ourObservation.MessageID())
	if err != nil {
		return false, fmt.Errorf(`failed to generate VAA ID from message id "%s": %w`, s.ourObservation.MessageID(), err)
	}

	vb, err := p.db.GetSignedVAABytes(*vaaID)
	if err != nil {
		if err == db.ErrVAANotFound {
			if p.logger.Level().Enabled(zapcore.DebugLevel) {
				p.logger.Debug("VAA not in DB",
					zap.String("message_id", s.ourObservation.MessageID()),
					zap.String("digest", hash),
				)
			}
			return false, nil
		} else {
			return false, fmt.Errorf(`failed to look up message id "%s" in db: %w`, s.ourObservation.MessageID(), err)
		}
	}

	v, err := vaa.Unmarshal(vb)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal VAA: %w", err)
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
