package p2p

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// The format of this time is very picky. Please use the exact format specified by cutOverFmtStr!
const cutOverTimeStr = "2024-12-31T23:59:59-0000"
const cutOverFmtStr = "2006-01-02T15:04:05-0700"

// checkForCutOver checks to see if / when we need to cut over to the new quic-v1 based on a preset time. If cutOverTimeStr is set, then this feature is enabled.
// If the current time is after the configured time, we should start up using the new quic-v1 bootstrap peers immediately. If the configured time is in the future,
// we need to start up with the existing bootstrap string and cut over to the new quic-v1 at the specified time.
func checkForCutOver(logger *zap.Logger, bootstrapPeers string, ccqBootstrapPeers string, components *Components) (newBootstrapPeers string, newCcqBootstrapPeers string, err error) {
	newBootstrapPeers, newCcqBootstrapPeers, delay, err := checkForCutOverImpl(logger, bootstrapPeers, ccqBootstrapPeers, components, cutOverTimeStr, time.Now())

	if delay != time.Duration(0) {
		// Wait for the cut over time and then panic so we restart with the new quic-v1. TODO: Can we just restart p2p??
		go func() {
			time.Sleep(delay)
			logger.Info("time to cut over to new quic-v1", zap.String("cutOverTime", cutOverTimeStr), zap.String("component", "p2pco"))
			panic("p2pco: time to cut over to new quic-v1")
		}()
	}

	return newBootstrapPeers, newCcqBootstrapPeers, err
}

func checkForCutOverImpl(logger *zap.Logger, bootstrapPeers string, ccqBootstrapPeers string, components *Components, coTimeStr string, now time.Time) (newBootstrapPeers string, newCcqBootstrapPeers string, delay time.Duration, err error) {
	newBootstrapPeers = bootstrapPeers
	newCcqBootstrapPeers = ccqBootstrapPeers

	if coTimeStr == "" {
		return
	}

	bootstrapIsV1, err := validateQuic(bootstrapPeers)
	if err != nil {
		logger.Error(`bootstrap peers string is invalid:`, zap.String("bootstrapPeers", bootstrapPeers), zap.Error(err), zap.String("component", "p2pco"))
		err = fmt.Errorf("unexpected format of bootstrap peers: %w", err)
		return
	}

	if ccqBootstrapPeers != "" {
		ccqBootstrapIsV1, ccqerr := validateQuic(ccqBootstrapPeers)
		if ccqerr != nil {
			logger.Error(`ccq bootstrap peers string is invalid:`, zap.String("ccqBootstrapPeers", ccqBootstrapPeers), zap.Error(ccqerr), zap.String("component", "p2pco"))
			err = fmt.Errorf("unexpected format of ccq bootstrap peers: %w", ccqerr)
			return
		}

		if bootstrapIsV1 != ccqBootstrapIsV1 {
			logger.Error(`there is a mismatch between bootstrap peers and ccq bootstrap peers:`,
				zap.Bool("bootstrapIsV1", bootstrapIsV1),
				zap.Bool("ccqBootstrapIsV1", ccqBootstrapIsV1),
				zap.String("bootstrapPeers", bootstrapPeers),
				zap.String("ccqBootstrapPeers", ccqBootstrapPeers),
				zap.Error(err), zap.String("component", "p2pco"),
			)
			err = fmt.Errorf("quic version mismatch between bootstrap peers and ccq bootstrap peers")
			return
		}
	}

	if bootstrapIsV1 {
		for _, la := range components.ListeningAddressesPatterns {
			if strings.Contains(la, "quic") && !strings.Contains(la, "quic-v1") {
				err = fmt.Errorf("bootstrapPeers has been updated to quic-v1, but components.ListeningAddressesPatterns has not: %s", la)
				return
			}
		}
		logger.Info("bootstrap peers parameter is already using quic-v1, cut over is not necessary", zap.String("bootstrapPeers", bootstrapPeers), zap.String("component", "p2pco"))
		delay = 0
		return
	}

	cutOverTime, err := time.Parse(cutOverFmtStr, coTimeStr)
	if err != nil {
		logger.Error("failed to parse p2p cut over time", zap.String("cutOverTimeStr", coTimeStr), zap.Error(err), zap.String("component", "p2pco"))
		err = fmt.Errorf("failed to parse cut over time: %w", err)
		return
	}

	if cutOverTime.Before(now) {
		// We should already be using the new quic-v1.
		newBootstrapPeers = strings.ReplaceAll(bootstrapPeers, "quic", "quic-v1")
		newCcqBootstrapPeers = strings.ReplaceAll(ccqBootstrapPeers, "quic", "quic-v1")

		for idx, la := range components.ListeningAddressesPatterns {
			components.ListeningAddressesPatterns[idx] = strings.ReplaceAll(la, "quic", "quic-v1")
		}

		logger.Info("cut over time has passed, using new quic-v1",
			zap.String("cutOverTime", cutOverTime.Format(cutOverFmtStr)),
			zap.String("now", now.Format(cutOverFmtStr)),
			zap.String("oldBootstrapPeers", bootstrapPeers),
			zap.String("newBootstrapPeers", newBootstrapPeers),
			zap.String("oldCcqBootstrapPeers", ccqBootstrapPeers),
			zap.String("newCcqBootstrapPeers", newCcqBootstrapPeers),
			zap.Any("newComponents", components),
			zap.String("component", "p2pco"))

		delay = 0
		return
	}

	// If we get here, we need to wait for the cutover and then force a restart.
	delay = cutOverTime.Sub(now)
	logger.Info("still waiting for cut over time",
		zap.Stringer("cutOverTime", cutOverTime),
		zap.String("now", now.Format(cutOverFmtStr)),
		zap.Stringer("delay", delay),
		zap.String("component", "p2pco"))

	return
}

func validateQuic(bootstrapPeers string) (bool, error) {
	if !strings.Contains(bootstrapPeers, "quic") {
		return false, fmt.Errorf(`unexpected format, does not contain "quic"`)
	}

	if !strings.Contains(bootstrapPeers, "quic-v1") {
		return false, nil
	}

	// If it contains any references to "quic-v1", make sure it doesn't have any references to "quic".
	// Do this by removing all the references to "quic-v1" and seeing if there are any remaining references to "quic".
	if strings.Contains(strings.ReplaceAll(bootstrapPeers, "quic-v1", ""), "quic") {
		return false, fmt.Errorf(`contains a mix of "quic" and "quic-v1"`)
	}

	return true, nil
}
