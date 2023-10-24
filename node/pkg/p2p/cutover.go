package p2p

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// The format of this time is very picky. Please use the exact format specified by cutOverFmtStr!
const mainnetCutOverTimeStr = ""
const testnetCutOverTimeStr = "2023-11-07T14:00:00-0000"
const devnetCutOverTimeStr = "2022-12-31T23:59:59-0000"
const cutOverFmtStr = "2006-01-02T15:04:05-0700"

// shouldCutOverPtr is a global variable used to determine if a cut over is in progress. It is initialized by the first call evaluateCutOver.
var shouldCutOverPtr *bool

// shouldCutOver uses the global variable to determine if a cut over is in progress. It assumes evaluateCutOver has already been called, so will panic if the pointer is nil.
func shouldCutOver() bool {
	if shouldCutOverPtr == nil {
		panic("shouldCutOverPtr is nil")
	}

	return *shouldCutOverPtr
}

// evaluateCutOver determines if a cut over is in progress. The first time it is called, it sets the global variable shouldCutOverPtr. It may be called more than once.
func evaluateCutOver(logger *zap.Logger, networkID string) error {
	if shouldCutOverPtr != nil {
		return nil
	}

	cutOverTimeStr := getCutOverTimeStr(networkID)

	sco, delay, err := evaluateCutOverImpl(logger, cutOverTimeStr, time.Now())
	if err != nil {
		return err
	}

	shouldCutOverPtr = &sco

	if delay != time.Duration(0) {
		// Wait for the cut over time and then panic so we restart with the new quic-v1.
		go func() {
			time.Sleep(delay)
			logger.Info("time to cut over to new quic-v1", zap.String("cutOverTime", cutOverTimeStr), zap.String("component", "p2pco"))
			panic("p2pco: time to cut over to new quic-v1")
		}()
	}

	return nil
}

// evaluateCutOverImpl performs the actual cut over check. It is a separate function for testing purposes.
func evaluateCutOverImpl(logger *zap.Logger, cutOverTimeStr string, now time.Time) (bool, time.Duration, error) {
	if cutOverTimeStr == "" {
		return false, 0, nil
	}

	cutOverTime, err := time.Parse(cutOverFmtStr, cutOverTimeStr)
	if err != nil {
		return false, 0, fmt.Errorf(`failed to parse cut over time: %w`, err)
	}

	if cutOverTime.Before(now) {
		logger.Info("cut over time has passed, should use new quic-v1", zap.String("cutOverTime", cutOverTime.Format(cutOverFmtStr)), zap.String("now", now.Format(cutOverFmtStr)), zap.String("component", "p2pco"))
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
func getCutOverTimeStr(networkID string) string {
	if strings.Contains(networkID, "/mainnet/") {
		return mainnetCutOverTimeStr
	}
	if strings.Contains(networkID, "/testnet/") {
		return testnetCutOverTimeStr
	}
	return devnetCutOverTimeStr
}

// cutOverBootstrapPeers checks to see if we are supposed to cut over, and if so updates the bootstrap peers. It assumes that the string has previously been validated.
func cutOverBootstrapPeers(bootstrapPeers string) string {
	if shouldCutOver() {
		bootstrapPeers = strings.ReplaceAll(bootstrapPeers, "/quic/", "/quic-v1/")
	}

	return bootstrapPeers
}

// cutOverAddressPattern checks to see if we are supposed to cut over, and if so updates the address patterns. It assumes that the string is valid.
func cutOverAddressPattern(pattern string) string {
	if shouldCutOver() {
		if !strings.Contains(pattern, "/quic-v1") {
			// These patterns are hardcoded so we are not worried about invalid values.
			pattern = strings.ReplaceAll(pattern, "/quic", "/quic-v1")
		}
	}

	return pattern
}
