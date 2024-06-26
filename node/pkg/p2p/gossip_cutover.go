package p2p

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// The format of this time is very picky. Please use the exact format specified by cutOverFmtStr!
const mainnetCutOverTimeStr = ""
const testnetCutOverTimeStr = ""
const devnetCutOverTimeStr = ""
const cutOverFmtStr = "2006-01-02T15:04:05-0700"

// gossipCutoverCompleteFlag indicates if the cutover time has passed, meaning we should publish only on the new topics.
var gossipCutoverCompleteFlag atomic.Bool

// GossipCutoverComplete returns true if the cutover time has passed, meaning we should publish on the new topic.
func GossipCutoverComplete() bool {
	return gossipCutoverCompleteFlag.Load()
}

// evaluateCutOver determines if the gossip cutover time has passed yet and sets the global flag accordingly. If the time has
// not yet passed, it creates a go routine to wait for that time and then set the flag.
func evaluateGossipCutOver(logger *zap.Logger, networkID string) error {
	cutOverTimeStr := getCutOverTimeStr(networkID)

	sco, delay, err := evaluateGossipCutOverImpl(logger, cutOverTimeStr, time.Now())
	if err != nil {
		return err
	}

	gossipCutoverCompleteFlag.Store(sco)
	logger.Info("evaluated cutover flag", zap.Bool("cutOverFlag", GossipCutoverComplete()), zap.String("cutOverTime", cutOverTimeStr), zap.String("component", "p2pco"))

	if delay != time.Duration(0) {
		// Wait for the cut over time and then update the flag.
		go func() {
			time.Sleep(delay)
			logger.Info("time to cut over to new gossip topics", zap.String("cutOverTime", cutOverTimeStr), zap.String("component", "p2pco"))
			gossipCutoverCompleteFlag.Store(true)
		}()
	}

	return nil
}

// evaluateGossipCutOverImpl performs the actual cut over check. It is a separate function for testing purposes.
func evaluateGossipCutOverImpl(logger *zap.Logger, cutOverTimeStr string, now time.Time) (bool, time.Duration, error) {
	if cutOverTimeStr == "" {
		return false, 0, nil
	}

	cutOverTime, err := time.Parse(cutOverFmtStr, cutOverTimeStr)
	if err != nil {
		return false, 0, fmt.Errorf(`failed to parse cut over time: %w`, err)
	}

	if cutOverTime.Before(now) {
		logger.Info("cut over time has passed, should use new gossip topics", zap.String("cutOverTime", cutOverTime.Format(cutOverFmtStr)), zap.String("now", now.Format(cutOverFmtStr)), zap.String("component", "p2pco"))
		return true, 0, nil
	}

	// If we get here, we need to wait for the cutover and then force a restart.
	delay := cutOverTime.Sub(now)
	logger.Info("still waiting for cut over time",
		zap.Stringer("cutOverTime", cutOverTime),
		zap.String("now", now.Format(cutOverFmtStr)),
		zap.Stringer("delay", delay),
		zap.String("component", "p2pco"))

	return false, delay, nil
}

// getCutOverTimeStr returns the cut over time string based on the network ID passed in.
func getCutOverTimeStr(networkID string) string { //nolint:unparam
	if strings.Contains(networkID, "/mainnet/") {
		return mainnetCutOverTimeStr
	}
	if strings.Contains(networkID, "/testnet/") {
		return testnetCutOverTimeStr
	}
	return devnetCutOverTimeStr
}

// GossipAttestationMsg is the payload of the `gossipAttestationSendC` channel. This will be used instead of just `[]byte`
// until after the cutover is complete and support for publishing `SignedObservations` is removed. Then this can be deleted.
type GossipAttestationMsg struct {
	MsgType GossipAttestationMsgType
	Msg     []byte
}

type GossipAttestationMsgType uint8

const (
	GossipAttestationSignedObservation GossipAttestationMsgType = iota
	GossipAttestationSignedObservationBatch
)
