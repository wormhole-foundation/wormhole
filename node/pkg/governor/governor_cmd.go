// This file contains the code to process admin commands to the chain governor.
// These functions are called from the adminserver.
//
// The chain governor supports the following admin client commands:
//   - cgov-status - displays the status of the chain governor to the log file.
//   - cgov-drop-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and discards it.
//   - cgov-release-pending-vaa [VAA_ID] - removes the specified transfer from the pending list and publishes it, without regard to the threshold.
//
// The VAA_ID is of the form "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/3", which is "emitter chain / emitter address / sequence number".

package governor

import (
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	"go.uber.org/zap"
)

func (gov *ChainGovernor) Status() string {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	startTime := time.Now().Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
	for _, ce := range gov.chains {
		valueTrans := sumValue(ce.transfers, startTime)
		s := fmt.Sprintf("cgov: chain: %v, dailyLimit: %v, total: %v, numPending: %v", ce.emitterChainId, ce.dailyLimit, valueTrans, len(ce.pending))
		gov.logger.Info(s)
		if len(ce.pending) != 0 {
			for idx, pe := range ce.pending {
				value, _ := computeValue(pe.amount, pe.token)
				s := fmt.Sprintf("   cgov: chain: %v, pending[%v], value: %v, vaa: %v, time: %v", ce.emitterChainId, idx, value,
					pe.msg.MessageIDString(), pe.timeStamp.String())
				gov.logger.Info(s)
			}
		}
	}

	return "grep the log for \"cgov:\" for status"
}

func (gov *ChainGovernor) Reload() (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	if gov.db == nil {
		return "", fmt.Errorf("unable to reload because the database is not initialized")
	}

	for _, ce := range gov.chains {
		ce.transfers = nil
		ce.pending = nil
	}

	if err := gov.loadFromDBAlreadyLocked(); err != nil {
		gov.logger.Error("cgov: failed to load from the database", zap.Error(err))
		return "", err
	}

	return "chain governor has been reset and reloaded", nil
}

func (gov *ChainGovernor) DropPendingVAA(vaaId string) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for idx, pe := range ce.pending {
			if pe.msg.MessageIDString() == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("cgov: dropping pending vaa",
					zap.String("msgId", pe.msg.MessageIDString()),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.timeStamp),
				)
				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				return "vaa has been dropped from the pending list", nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

func (gov *ChainGovernor) ReleasePendingVAA(vaaId string) (string, error) {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, ce := range gov.chains {
		for idx, pe := range ce.pending {
			if pe.msg.MessageIDString() == vaaId {
				value, _ := computeValue(pe.amount, pe.token)
				gov.logger.Info("cgov: releasing pending vaa, should be published soon",
					zap.String("msgId", pe.msg.MessageIDString()),
					zap.Uint64("value", value),
					zap.Stringer("timeStamp", pe.timeStamp),
				)

				gov.msgsToPublish = append(gov.msgsToPublish, pe.msg)
				ce.pending = append(ce.pending[:idx], ce.pending[idx+1:]...)
				return "pending vaa has been released and will be published soon", nil
			}
		}
	}

	return "", fmt.Errorf("vaa not found in the pending list")
}

func sumValue(transfers []db.Transfer, startTime time.Time) uint64 {
	if len(transfers) == 0 {
		return 0
	}

	var sum uint64

	for _, t := range transfers {
		if !t.Timestamp.Before(startTime) {
			sum += t.Value
		}
	}

	return sum
}
