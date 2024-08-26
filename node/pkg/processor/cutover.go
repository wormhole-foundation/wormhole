package processor

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
const devnetCutOverTimeStr = "2024-08-01T00:00:00-0500"
const cutOverFmtStr = "2006-01-02T15:04:05-0700"

// batchCutoverCompleteFlag indicates if the cutover time has passed, meaning we should publish observation batches.
var batchCutoverCompleteFlag atomic.Bool

// batchCutoverComplete returns true if the cutover time has passed, meaning we should publish observation batches.
func batchCutoverComplete() bool {
	return batchCutoverCompleteFlag.Load()
}

// evaluateCutOver determines if the cutover time has passed yet and sets the global flag accordingly. If the time has
// not yet passed, it creates a go routine to wait for that time and then sets the flag.
func evaluateBatchCutover(logger *zap.Logger, networkID string) error {
	cutOverTimeStr := getCutOverTimeStr(networkID)

	sco, delay, err := evaluateBatchCutoverImpl(logger, cutOverTimeStr, time.Now())
	if err != nil {
		return err
	}

	batchCutoverCompleteFlag.Store(sco)
	logger.Info("evaluated cutover flag", zap.Bool("cutOverFlag", batchCutoverComplete()), zap.String("cutOverTime", cutOverTimeStr), zap.String("component", "batchco"))

	if delay != time.Duration(0) {
		// Wait for the cut over time and then update the flag.
		go func() {
			time.Sleep(delay)
			logger.Info("time to cut over to batch publishing", zap.String("cutOverTime", cutOverTimeStr), zap.String("component", "batchco"))
			batchCutoverCompleteFlag.Store(true)
		}()
	}

	return nil
}

// evaluateBatchCutoverImpl performs the actual cut over check. It is a separate function for testing purposes.
func evaluateBatchCutoverImpl(logger *zap.Logger, cutOverTimeStr string, now time.Time) (bool, time.Duration, error) {
	if cutOverTimeStr == "" {
		return false, 0, nil
	}

	cutOverTime, err := time.Parse(cutOverFmtStr, cutOverTimeStr)
	if err != nil {
		return false, 0, fmt.Errorf(`failed to parse cut over time: %w`, err)
	}

	if cutOverTime.Before(now) {
		logger.Info("cut over time has passed, should publish observation batches", zap.String("cutOverTime", cutOverTime.Format(cutOverFmtStr)), zap.String("now", now.Format(cutOverFmtStr)), zap.String("component", "batchco"))
		return true, 0, nil
	}

	// If we get here, we need to wait for the cutover and then switch the global flag.
	delay := cutOverTime.Sub(now)
	logger.Info("still waiting for cut over time",
		zap.Stringer("cutOverTime", cutOverTime),
		zap.String("now", now.Format(cutOverFmtStr)),
		zap.Stringer("delay", delay),
		zap.String("component", "batchco"))

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
