// The purpose of the Reobserver is to detect when a local observation has not reached quorum in a timely manner, and automatically
// request a reobservation. It works by tracking observations received locally by the watchers, and comparing those to observations
// that reach quorum. If an observation does not reach quorum in a timely manner (as determined by retryIntervalInMinutes), a reobservation
// request is submitted for it. Once expirationIntervalInMinutes minutes have elapsed since an observation reaches quorum, or maxRetries
// reobservation attemps have failed, it is deleted from the cache.
//
// The reobserver uses the following strategy to control reobservation attempts. Every minute, the reobservation monitor is called from the
// processor go routine. Each interval, it scans the list of known observations. If an observation has reached quorum and expirationIntervalInMinutes
// minutes have elapsed, it is considered complete and deleted from the list. Althernatively, if it has not reached quorum, a reobservation request
// is generated. This will continue until either the observation reaches quorum (success), or maxRetries attempts have failed (failure).
//
// Additionally, at most maxRetriesPerInterval reobservation requests will be generated each minute. This is to avoid flooding the gossip network.
//
// To enable the Reobserver, you must specified the --reobserverEnabled guardiand command line argument.

package reobserver

import (
	"context"
	"sync"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/ethereum/go-ethereum/common"

	"go.uber.org/zap"
)

// An observation will be dropped from the map this many minutes after it reaches quorum.
const expirationIntervalInMinutes = 60

// We will try sending a reobservation request for an observation this often (until it reaches quorum or we hit the retry limit).
const retryIntervalInMinutes = 5

// We will give up after this many reobservation retries.
const maxRetries = 10

// We will only send this many reobservation request per interval.
const maxRetriesPerInterval = 5

type (
	// Payload for each observation request
	observationEntry struct {
		msgId         string
		chainId       vaa.ChainID
		txHash        common.Hash
		timeStamp     time.Time
		numRetries    int
		quorumReached bool
		completed     bool
	}
)

func (oe *observationEntry) localMsgReceived() bool {
	return oe.chainId != vaa.ChainIDUnset
}

type Reobserver struct {
	logger             *zap.Logger
	mutex              sync.Mutex
	observations       map[string]*observationEntry
	obsvReqSendC       chan *gossipv1.ObservationRequest
	retryInterval      time.Duration
	expirationInterval time.Duration
}

func NewReobserver(
	logger *zap.Logger,
	obsvReqSendC chan *gossipv1.ObservationRequest,
) *Reobserver {
	return &Reobserver{
		logger:       logger,
		observations: make(map[string]*observationEntry),
		obsvReqSendC: obsvReqSendC,
	}
}

func (reob *Reobserver) Run(ctx context.Context) error {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	reob.retryInterval = time.Minute * time.Duration(retryIntervalInMinutes)
	reob.expirationInterval = time.Minute * time.Duration(expirationIntervalInMinutes)

	reob.logger.Info("reobserver: starting reobservation monitor",
		zap.Int("maxRetries", maxRetries),
		zap.Stringer("retryInterval", reob.retryInterval),
		zap.Stringer("expirationInterval", reob.expirationInterval),
	)

	return nil
}

func (reob *Reobserver) AddMessage(msgId string, chainId vaa.ChainID, txHash common.Hash) {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	now := time.Now()

	oe, exists := reob.observations[msgId]
	if !exists {
		oe := &observationEntry{msgId: msgId, chainId: chainId, txHash: txHash, timeStamp: now}
		reob.observations[msgId] = oe
		reob.logger.Info("reobserver: adding message", zap.String("msgID", msgId))
	} else {
		oe.chainId = chainId
		oe.txHash = txHash
		oe.timeStamp = now
		if oe.quorumReached {
			oe.completed = true
			reob.logger.Info("reobserver: ignoring message because it has already reached quorum", zap.String("msgID", msgId))
		} else {
			reob.logger.Info("reobserver: ignoring message because we have already seen it, although it has not yet reached quorum", zap.String("msgID", msgId))
		}
	}
}

func (reob *Reobserver) QuorumReached(msgId string) {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	now := time.Now()

	oe, exists := reob.observations[msgId]
	if !exists {
		oe := &observationEntry{msgId: msgId, timeStamp: now, quorumReached: true}
		reob.observations[msgId] = oe
		reob.logger.Info("reobserver: received a quorum notification for a message we don't know about yet, adding it", zap.String("msgID", msgId))
	} else if oe.quorumReached {
		reob.logger.Info("reobserver: ignoring a quorum notification because it has already reached quorum", zap.String("msgID", msgId))
	} else {
		oe.completed = true
		oe.quorumReached = true
		oe.timeStamp = now
		reob.logger.Info("reobserver: received a quorum notification", zap.String("msgID", msgId), zap.Int("numRetries", oe.numRetries))

		if oe.numRetries > 0 {
			metricSuccessfulReobservations.Inc()
		}
	}
}

func (reob *Reobserver) CheckForReobservations() error {
	return reob.checkForReobservationsForTime(time.Now())
}

func (reob *Reobserver) checkForReobservationsForTime(now time.Time) error {
	reob.mutex.Lock()
	defer reob.mutex.Unlock()

	entriesToDelete := []*observationEntry{}
	numSentThisInterval := 0
	for msgId, oe := range reob.observations {
		if oe.completed {
			expirationTime := oe.timeStamp.Add(reob.expirationInterval)
			if expirationTime.Before(now) {
				reob.logger.Info("reobserver: completed observation has expired, dropping it", zap.String("msgId", msgId))
				entriesToDelete = append(entriesToDelete, oe)
			}
		} else if reob.shouldReobserve(oe, now, numSentThisInterval) {
			numSentThisInterval++
			oe.numRetries++
			oe.timeStamp = now
			reob.logger.Info("reobserver: requesting reobservation", zap.String("msgId", msgId), zap.Int("numRetries", oe.numRetries))

			req := &gossipv1.ObservationRequest{
				ChainId: uint32(oe.chainId),
				TxHash:  oe.txHash.Bytes(),
			}
			reob.obsvReqSendC <- req
		} else if oe.numRetries >= maxRetries {
			reob.logger.Error("reobserver: giving up on reobservation because the retry limit has been reached",
				zap.String("msgId", msgId),
				zap.Int("numRetries", oe.numRetries),
			)

			entriesToDelete = append(entriesToDelete, oe)
			metricFailedReobservationAttempts.Inc()
		} else if !oe.localMsgReceived() {
			expirationTime := oe.timeStamp.Add(reob.expirationInterval)
			if expirationTime.Before(now) {
				reob.logger.Error("reobserver: giving up on reobservation because we received a quorum notification but never saw the observation locally",
					zap.String("msgId", msgId),
					zap.Stringer("timeStamp", oe.timeStamp),
				)

				entriesToDelete = append(entriesToDelete, oe)
				metricFailedReobservationAttempts.Inc()
			}
		}
	}

	if len(entriesToDelete) != 0 {
		for _, oe := range entriesToDelete {
			reob.logger.Info("reobserver: dropping reobservation request",
				zap.String("msgId", oe.msgId),
				zap.Stringer("timeCompleted", oe.timeStamp),
				zap.Int("numRetries", oe.numRetries),
			)

			delete(reob.observations, oe.msgId)
		}
	}

	return nil
}

func (reob *Reobserver) shouldReobserve(oe *observationEntry, now time.Time, numSentThisInterval int) bool {
	if oe.quorumReached {
		return false
	}

	if oe.numRetries >= maxRetries {
		return false
	}

	if !oe.localMsgReceived() {
		// If we haven't seen the message locally, we don't have a txHash, so we can't request reobservation.
		return false
	}

	if numSentThisInterval >= maxRetriesPerInterval {
		return false
	}

	nextRetryTime := oe.timeStamp.Add(reob.retryInterval)
	return now.After(nextRetryTime)
}
