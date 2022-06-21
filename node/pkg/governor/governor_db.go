// This file contains the code to load transfers and pending messages from the database.

package governor

import (
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"go.uber.org/zap"
)

func (gov *ChainGovernor) loadFromDB() error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	return gov.loadFromDBAlreadyLocked()
}

func (gov *ChainGovernor) loadFromDBAlreadyLocked() error {
	xfers, pending, err := gov.db.GetChainGovernorData(gov.logger)
	if err != nil {
		gov.logger.Error("cgov: failed to reload transactions from db", zap.Error(err))
		return err
	}

	now := time.Now()
	if len(pending) != 0 {
		sort.SliceStable(pending, func(i, j int) bool {
			return pending[i].Timestamp.Before(pending[j].Timestamp)
		})

		for _, k := range pending {
			gov.reloadPendingTransfer(k, now)
		}
	}

	if len(xfers) != 0 {
		sort.SliceStable(xfers, func(i, j int) bool {
			return xfers[i].Timestamp.Before(xfers[j].Timestamp)
		})

		startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
		for _, t := range xfers {
			if startTime.Before(t.Timestamp) {
				gov.reloadTransfer(t, now, startTime)
			} else {
				if err := gov.db.DeleteTransfer(t); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (gov *ChainGovernor) reloadPendingTransfer(k *common.MessagePublication, now time.Time) {
	ce, exists := gov.chains[k.EmitterChain]
	if !exists {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported chain, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
		)
		return
	}

	if k.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported emitter address, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
		)
		return
	}

	payload, err := vaa.DecodeTransferPayloadHdr(k.Payload)
	if err != nil {
		gov.logger.Error("cgov: failed to parse payload for reloaded pending transfer, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
			zap.Stringer("tokenChain", payload.OriginChain),
			zap.Stringer("tokenAddress", payload.OriginAddress),
			zap.Error(err),
		)
		return
	}

	tk := tokenKey{chain: payload.OriginChain, addr: payload.OriginAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported token, dropping it",
			zap.String("MsgID", k.MessageIDString()),
			zap.Stringer("TxHash", k.TxHash),
			zap.Stringer("Timestamp", k.Timestamp),
			zap.Uint32("Nonce", k.Nonce),
			zap.Uint64("Sequence", k.Sequence),
			zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
			zap.Stringer("EmitterChain", k.EmitterChain),
			zap.Stringer("EmitterAddress", k.EmitterAddress),
			zap.Stringer("tokenChain", payload.OriginChain),
			zap.Stringer("tokenAddress", payload.OriginAddress),
		)
		return
	}

	gov.logger.Info("cgov: reloaded pending transfer",
		zap.String("MsgID", k.MessageIDString()),
		zap.Stringer("TxHash", k.TxHash),
		zap.Stringer("Timestamp", k.Timestamp),
		zap.Uint32("Nonce", k.Nonce),
		zap.Uint64("Sequence", k.Sequence),
		zap.Uint8("ConsistencyLevel", k.ConsistencyLevel),
		zap.Stringer("EmitterChain", k.EmitterChain),
		zap.Stringer("EmitterAddress", k.EmitterAddress),
		zap.Stringer("Amount", payload.Amount),
	)

	ce.pending = append(ce.pending, pendingEntry{timeStamp: now, token: token, amount: payload.Amount, msg: k})
}

func (gov *ChainGovernor) reloadTransfer(t *db.Transfer, now time.Time, startTime time.Time) {
	ce, exists := gov.chains[t.EmitterChain]
	if !exists {
		gov.logger.Error("cgov: reloaded transfer for unsupported chain, dropping it",
			zap.Stringer("Timestamp", t.Timestamp),
			zap.Uint64("Value", t.Value),
			zap.Stringer("EmitterChain", t.EmitterChain),
			zap.Stringer("EmitterAddress", t.EmitterAddress),
			zap.String("MsgID", t.MsgID),
		)
		return
	}

	if t.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("cgov: reloaded transfer for unsupported emitter address, dropping it",
			zap.Stringer("Timestamp", t.Timestamp),
			zap.Uint64("Value", t.Value),
			zap.Stringer("OriginChain", t.OriginChain),
			zap.Stringer("OriginAddress", t.OriginAddress),
			zap.String("MsgID", t.MsgID),
		)
		return
	}

	tk := tokenKey{chain: t.OriginChain, addr: t.OriginAddress}
	_, exists = gov.tokens[tk]
	if !exists {
		gov.logger.Error("cgov: reloaded transfer for unsupported token, dropping it",
			zap.Stringer("Timestamp", t.Timestamp),
			zap.Uint64("Value", t.Value),
			zap.Stringer("OriginChain", t.OriginChain),
			zap.Stringer("OriginAddress", t.OriginAddress),
			zap.String("MsgID", t.MsgID),
		)
		return
	}

	gov.logger.Info("cgov: reloaded transfer",
		zap.Stringer("Timestamp", t.Timestamp),
		zap.Uint64("Value", t.Value),
		zap.Stringer("OriginChain", t.OriginChain),
		zap.Stringer("OriginAddress", t.OriginAddress),
		zap.String("MsgID", t.MsgID),
	)

	ce.transfers = append(ce.transfers, *t)
}
