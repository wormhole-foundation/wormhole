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
		for _, xfer := range xfers {
			if startTime.Before(xfer.Timestamp) {
				gov.reloadTransfer(xfer, now, startTime)
			} else {
				if err := gov.db.DeleteTransfer(xfer); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (gov *ChainGovernor) reloadPendingTransfer(msg *common.MessagePublication, now time.Time) {
	ce, exists := gov.chains[msg.EmitterChain]
	if !exists {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported chain, dropping it",
			zap.String("MsgID", msg.MessageIDString()),
			zap.Stringer("TxHash", msg.TxHash),
			zap.Stringer("Timestamp", msg.Timestamp),
			zap.Uint32("Nonce", msg.Nonce),
			zap.Uint64("Sequence", msg.Sequence),
			zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
			zap.Stringer("EmitterChain", msg.EmitterChain),
			zap.Stringer("EmitterAddress", msg.EmitterAddress),
		)
		return
	}

	if msg.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("cgov: reloaded pending transfer for unsupported emitter address, dropping it",
			zap.String("MsgID", msg.MessageIDString()),
			zap.Stringer("TxHash", msg.TxHash),
			zap.Stringer("Timestamp", msg.Timestamp),
			zap.Uint32("Nonce", msg.Nonce),
			zap.Uint64("Sequence", msg.Sequence),
			zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
			zap.Stringer("EmitterChain", msg.EmitterChain),
			zap.Stringer("EmitterAddress", msg.EmitterAddress),
		)
		return
	}

	payload, err := vaa.DecodeTransferPayloadHdr(msg.Payload)
	if err != nil {
		gov.logger.Error("cgov: failed to parse payload for reloaded pending transfer, dropping it",
			zap.String("MsgID", msg.MessageIDString()),
			zap.Stringer("TxHash", msg.TxHash),
			zap.Stringer("Timestamp", msg.Timestamp),
			zap.Uint32("Nonce", msg.Nonce),
			zap.Uint64("Sequence", msg.Sequence),
			zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
			zap.Stringer("EmitterChain", msg.EmitterChain),
			zap.Stringer("EmitterAddress", msg.EmitterAddress),
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
			zap.String("MsgID", msg.MessageIDString()),
			zap.Stringer("TxHash", msg.TxHash),
			zap.Stringer("Timestamp", msg.Timestamp),
			zap.Uint32("Nonce", msg.Nonce),
			zap.Uint64("Sequence", msg.Sequence),
			zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
			zap.Stringer("EmitterChain", msg.EmitterChain),
			zap.Stringer("EmitterAddress", msg.EmitterAddress),
			zap.Stringer("tokenChain", payload.OriginChain),
			zap.Stringer("tokenAddress", payload.OriginAddress),
		)
		return
	}

	gov.logger.Info("cgov: reloaded pending transfer",
		zap.String("MsgID", msg.MessageIDString()),
		zap.Stringer("TxHash", msg.TxHash),
		zap.Stringer("Timestamp", msg.Timestamp),
		zap.Uint32("Nonce", msg.Nonce),
		zap.Uint64("Sequence", msg.Sequence),
		zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
		zap.Stringer("EmitterChain", msg.EmitterChain),
		zap.Stringer("EmitterAddress", msg.EmitterAddress),
		zap.Stringer("Amount", payload.Amount),
	)

	ce.pending = append(ce.pending, pendingEntry{timeStamp: now, token: token, amount: payload.Amount, msg: msg})
}

func (gov *ChainGovernor) reloadTransfer(xfer *db.Transfer, now time.Time, startTime time.Time) {
	ce, exists := gov.chains[xfer.EmitterChain]
	if !exists {
		gov.logger.Error("cgov: reloaded transfer for unsupported chain, dropping it",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("EmitterChain", xfer.EmitterChain),
			zap.Stringer("EmitterAddress", xfer.EmitterAddress),
			zap.String("MsgID", xfer.MsgID),
		)
		return
	}

	if xfer.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("cgov: reloaded transfer for unsupported emitter address, dropping it",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
		)
		return
	}

	tk := tokenKey{chain: xfer.OriginChain, addr: xfer.OriginAddress}
	_, exists = gov.tokens[tk]
	if !exists {
		gov.logger.Error("cgov: reloaded transfer for unsupported token, dropping it",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
		)
		return
	}

	gov.logger.Info("cgov: reloaded transfer",
		zap.Stringer("Timestamp", xfer.Timestamp),
		zap.Uint64("Value", xfer.Value),
		zap.Stringer("OriginChain", xfer.OriginChain),
		zap.Stringer("OriginAddress", xfer.OriginAddress),
		zap.String("MsgID", xfer.MsgID),
	)

	ce.transfers = append(ce.transfers, *xfer)
}
