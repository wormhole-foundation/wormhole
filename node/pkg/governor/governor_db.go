// This file contains the code to load transfers and pending messages from the database.

package governor

import (
	"sort"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"go.uber.org/zap"
)

func (gov *ChainGovernor) loadFromDB() error {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()
	return gov.loadFromDBAlreadyLocked()
}

// Loads transfers and pending data from the database and modifies the corresponding fields in the ChainGovernor.
// These fields are slices transfers or pendingTransfers and will be sorted by their Timestamp property.
// Modifies the state of the database as a side-effect: 'transfers' that are older than 24 hours are deleted.
func (gov *ChainGovernor) loadFromDBAlreadyLocked() error {
	xfers, pending, err := gov.db.GetChainGovernorData(gov.logger)
	if err != nil {
		gov.logger.Error("failed to reload transactions from db", zap.Error(err))
		return err
	}

	now := time.Now()
	if len(pending) != 0 {
		sort.SliceStable(pending, func(i, j int) bool {
			return pending[i].Msg.Timestamp.Before(pending[j].Msg.Timestamp)
		})

		for _, p := range pending {
			gov.reloadPendingTransfer(p)
		}
	}

	if len(xfers) != 0 {
		sort.SliceStable(xfers, func(i, j int) bool {
			return xfers[i].Timestamp.Before(xfers[j].Timestamp)
		})

		startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
		for _, xfer := range xfers {
			if startTime.Before(xfer.Timestamp) {
				if err := gov.reloadTransfer(xfer); err != nil {
					return err
				}
			} else {
				if err := gov.db.DeleteTransfer(xfer); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (gov *ChainGovernor) reloadPendingTransfer(pending *db.PendingTransfer) {
	msg := &pending.Msg
	ce, exists := gov.chains[msg.EmitterChain]
	if !exists {
		gov.logger.Error("reloaded pending transfer for unsupported chain, dropping it",
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
		gov.logger.Error("reloaded pending transfer for unsupported emitter address, dropping it",
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
		gov.logger.Error("failed to parse payload for reloaded pending transfer, dropping it",
			zap.String("MsgID", msg.MessageIDString()),
			zap.Stringer("TxHash", msg.TxHash),
			zap.Stringer("Timestamp", msg.Timestamp),
			zap.Uint32("Nonce", msg.Nonce),
			zap.Uint64("Sequence", msg.Sequence),
			zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
			zap.Stringer("EmitterChain", msg.EmitterChain),
			zap.Stringer("EmitterAddress", msg.EmitterAddress),
			zap.Error(err),
		)
		return
	}

	tk := tokenKey{chain: payload.OriginChain, addr: payload.OriginAddress}
	token, exists := gov.tokens[tk]
	if !exists {
		gov.logger.Error("reloaded pending transfer for unsupported token, dropping it",
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

	hash := gov.HashFromMsg(msg)

	if _, alreadyExists := gov.msgsSeen[hash]; alreadyExists {
		gov.logger.Error("not reloading pending transfer because it is a duplicate",
			zap.String("MsgID", msg.MessageIDString()),
			zap.Stringer("TxHash", msg.TxHash),
			zap.Stringer("Timestamp", msg.Timestamp),
			zap.Uint32("Nonce", msg.Nonce),
			zap.Uint64("Sequence", msg.Sequence),
			zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
			zap.Stringer("EmitterChain", msg.EmitterChain),
			zap.Stringer("EmitterAddress", msg.EmitterAddress),
			zap.Stringer("Amount", payload.Amount),
			zap.String("Hash", hash),
		)
		return
	}

	gov.logger.Info("reloaded pending transfer",
		zap.String("MsgID", msg.MessageIDString()),
		zap.Stringer("TxHash", msg.TxHash),
		zap.Stringer("Timestamp", msg.Timestamp),
		zap.Uint32("Nonce", msg.Nonce),
		zap.Uint64("Sequence", msg.Sequence),
		zap.Uint8("ConsistencyLevel", msg.ConsistencyLevel),
		zap.Stringer("EmitterChain", msg.EmitterChain),
		zap.Stringer("EmitterAddress", msg.EmitterAddress),
		zap.Stringer("Amount", payload.Amount),
		zap.String("Hash", hash),
	)

	// Note: no flow cancel added here. We only want to add an inverse, flow-cancel transfer when the transfer is
	// released from the pending queue, not when it's added.
	ce.pending = append(ce.pending, &pendingEntry{token: token, amount: payload.Amount, hash: hash, dbData: *pending})
	gov.msgsSeen[hash] = transferEnqueued
}

// Processes a db.Transfer and validates that it should be loaded into `gov`.
// Modifies `gov` as a side-effect: when valid transfer is loaded, the properties 'transfers' and 'msgsSeen' are
// updated with information about the loaded transfer. In the case of a loading a transfer of a flow-canceling asset,
// both chain entries (emitter and target) will be updated.
func (gov *ChainGovernor) reloadTransfer(xfer *db.Transfer) error {
	ce, exists := gov.chains[xfer.EmitterChain]
	if !exists {
		gov.logger.Error("reloaded transfer for unsupported chain, dropping it",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("EmitterChain", xfer.EmitterChain),
			zap.Stringer("EmitterAddress", xfer.EmitterAddress),
			zap.String("MsgID", xfer.MsgID),
		)
		return nil
	}

	if xfer.EmitterAddress != ce.emitterAddr {
		gov.logger.Error("reloaded transfer for unsupported emitter address, dropping it",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
		)
		return nil
	}

	tk := tokenKey{chain: xfer.OriginChain, addr: xfer.OriginAddress}
	_, exists = gov.tokens[tk]
	if !exists {
		gov.logger.Error("reloaded transfer for unsupported token, dropping it",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
		)
		return nil
	}

	if _, alreadyExists := gov.msgsSeen[xfer.Hash]; alreadyExists {
		gov.logger.Info("not reloading transfer because it is a duplicate",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
			zap.String("Hash", xfer.Hash),
		)
		return nil
	}

	if xfer.Hash != "" {
		gov.logger.Info("reloaded transfer",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
			zap.String("Hash", xfer.Hash),
		)

		gov.msgsSeen[xfer.Hash] = transferComplete
	} else {
		gov.logger.Error("reloaded transfer that does not have a hash, will not be able to detect a duplicate",
			zap.Stringer("Timestamp", xfer.Timestamp),
			zap.Uint64("Value", xfer.Value),
			zap.Stringer("OriginChain", xfer.OriginChain),
			zap.Stringer("OriginAddress", xfer.OriginAddress),
			zap.String("MsgID", xfer.MsgID),
		)
	}

	transfer, err := newTransferFromDbTransfer(xfer)
	if err != nil {
		return err
	}
	ce.transfers = append(ce.transfers, transfer)

	// If the transfer does not flow cancel, we're done now. Transfers about the bigTransactionSize never flow cancel.
	if ce.isBigTransfer(xfer.Value) {
		return nil
	}

	// Reload flow-cancel transfers for the TargetChain. This is important when node restarts so that a corresponding,
	// inverse transfer is added to the TargetChain. This is already done during the `ProcessMsgForTime` loop but
	// that function does not capture flow-cancelling when the node is restarted.
	tokenEntry := gov.tokens[tk]
	if tokenEntry != nil {
		// Mandatory check to ensure that the token should be able to reduce the Governor limit.
		if tokenEntry.flowCancels {
			if destinationChainEntry, ok := gov.chains[xfer.TargetChain]; ok {
				if err := destinationChainEntry.addFlowCancelTransferFromDbTransfer(xfer); err != nil {
					return  err
				}
			} else {
				gov.logger.Warn("tried to cancel flow but chain entry for target chain does not exist",
					zap.String("msgID", xfer.MsgID),
					zap.Stringer("token chain", xfer.OriginChain),
					zap.Stringer("token address", xfer.OriginAddress),
					zap.Stringer("target chain", xfer.TargetChain),
				)
			}
		}
	}
	return nil
}
