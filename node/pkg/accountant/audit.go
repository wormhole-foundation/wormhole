// This code audits the set of pending transfers against the state reported by the smart contract. It is called from the processor every minute,
// but the audit is performed less frequently. The audit occurs in two phases that operate off of a temporary map of all pending transfers known to this guardian.
//
// The first phase involves querying the smart contract for any observations that it thinks are missing for this guardian. The audit processes everything in the
// returned results and does one of the following:
// - If the observation is in our temporary map, we resubmit an observation to the contract and delete it from our temporary map.
// - If the observation is not in the temporary map, we request a reobservation from the local watcher.
//
// The second phase consists of requesting the status from the contract for everything that is still in the temporary map. For each returned item, we do the following:
// - If the contract indicates that the transfer has been committed, we validate the digest, then publish it and delete it from the map.
// - If the contract indicates that the transfer is pending, we continue to wait for it to be committed.
// - If the contract indicates any other status (most likely meaning it does not know about it), we resubmit an observation to the contract
//
// Note that any time we are considering resubmitting an observation to the contract, we first check the "submit pending" flag. If that is set, we do not
// submit the observation to the contract, but continue to wait for it to work its way through the queue.

package accountant

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"

	ethCommon "github.com/ethereum/go-ethereum/common"

	"go.uber.org/zap"
)

const (
	// auditInterval indicates how often the audit runs (given that it is invoked by the processor once per minute)
	auditInterval = 5 * time.Minute

	// maxSubmitPendingTime indicates how long a transfer can be in the submit pending state before the audit starts complaining about it.
	maxSubmitPendingTime = 30 * time.Minute
)

type (
	// MissingObservationsResponse is the result returned from the "missing_observations" query.
	MissingObservationsResponse struct {
		Missing []MissingObservation
	}

	MissingObservation struct {
		ChainId uint16 `json:"chain_id"`
		TxHash  []byte `json:"tx_hash"`
	}

	// BatchTransferStatusResponse is the result returned by the "batch_transfer_status" query.
	BatchTransferStatusResponse struct {
		Details []TransferDetails `json:"details"`
	}

	TransferDetails struct {
		Key    TransferKey
		Status TransferStatus
	}

	TransferStatus struct {
		Committed *TransferStatusCommitted `json:"committed"`
		Pending   *TransferStatusPending   `json:"pending"`
	}

	TransferStatusCommitted struct {
		Data   TransferData `json:"data"`
		Digest []byte       `json:"digest"`
	}

	TransferData struct {
		Amount         *cosmossdk.Int `json:"amount"`
		TokenChain     uint16         `json:"token_chain"`
		TokenAddress   vaa.Address    `json:"token_address"`
		RecipientChain uint16         `json:"recipient_chain"`
	}

	TransferStatusPending struct {
	}
)

// makeAuditKey creates an audit map key from a missing observation.
func (mo *MissingObservation) makeAuditKey() string {
	return fmt.Sprintf("%d-%s", mo.ChainId, hex.EncodeToString(mo.TxHash[:]))
}

// makeAuditKey creates an audit map key from a pending observation entry.
func (pe *pendingEntry) makeAuditKey() string {
	return fmt.Sprintf("%d-%s", pe.msg.EmitterChain, pe.msg.TxHash.String())
}

// AuditPendingTransfers is the entry point for the audit of the pending transfer map. It determines if it has been long enough since the last audit.
// If so, it creates a temporary map of all pending transfers and invokes the main audit function as a go routine.
func (acct *Accountant) AuditPendingTransfers() {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	if time.Since(acct.lastAuditTime) < auditInterval {
		acct.logger.Debug("acctaudit: in AuditPendingTransfers, not time to run yet", zap.Stringer("lastAuditTime", acct.lastAuditTime))
		return
	}

	tmpMap := make(map[string]*pendingEntry)
	for _, pe := range acct.pendingTransfers {
		if (pe.submitPending) && (time.Since(pe.updTime) > maxSubmitPendingTime) {
			auditErrors.Inc()
			acct.logger.Error("acctaudit: transfer has been in the submit pending state for too long", zap.Stringer("lastUpdateTime", pe.updTime))
		}
		acct.logger.Debug("acctaudit: will audit pending transfer", zap.String("msgId", pe.msgId), zap.Stringer("lastUpdateTime", pe.updTime))
		tmpMap[pe.makeAuditKey()] = pe
	}

	acct.logger.Debug("acctaudit: in AuditPendingTransfers: starting audit", zap.Int("numPending", len(tmpMap)))
	acct.lastAuditTime = time.Now()
	go acct.performAudit(tmpMap)
	acct.logger.Debug("acctaudit: leaving AuditPendingTransfers")
}

// performAudit audits the temporary map against the smart contract. It is meant to be run in a go routine. It takes a temporary map of all pending transfers
// and validates that against what is reported by the smart contract. For more details, please see the prologue of this file.
func (acct *Accountant) performAudit(tmpMap map[string]*pendingEntry) {
	acct.logger.Debug("acctaudit: entering performAudit")
	missingObservations, err := acct.queryMissingObservations()
	if err != nil {
		acct.logger.Error("acctaudit: unable to perform audit, failed to query missing observations", zap.Error(err))
		for _, pe := range tmpMap {
			acct.logger.Error("acctaudit: unsure of status of pending transfer due to query error", zap.String("msgId", pe.msgId))
		}
		return
	}

	if len(missingObservations) != 0 {
		for _, mo := range missingObservations {
			key := mo.makeAuditKey()
			pe, exists := tmpMap[key]
			if exists {
				if !pe.submitPending {
					auditErrors.Inc()
					acct.logger.Error("acctaudit: contract reported pending observation as missing, resubmitting it", zap.String("msgID", pe.msgId))
					acct.submitObservation(pe)
				} else {
					acct.logger.Info("acctaudit: contract reported pending observation as missing but it is queued up to be submitted, skipping it", zap.String("msgID", pe.msgId))
				}

				delete(tmpMap, key)
			} else {
				acct.handleMissingObservation(mo)
			}
		}
	}

	if len(tmpMap) != 0 {
		var keys []TransferKey
		var pendingTransfers []*pendingEntry
		for _, pe := range tmpMap {
			keys = append(keys, TransferKey{EmitterChain: uint16(pe.msg.EmitterChain), EmitterAddress: pe.msg.EmitterAddress, Sequence: pe.msg.Sequence})
			pendingTransfers = append(pendingTransfers, pe)
		}

		transferDetails, err := acct.queryBatchTransferStatus(keys)
		if err != nil {
			acct.logger.Error("acctaudit: unable to finish audit, failed to query for transfer statuses", zap.Error(err))
			for _, pe := range tmpMap {
				acct.logger.Error("acctaudit: unsure of status of pending transfer due to query error", zap.String("msgId", pe.msgId))
			}
			return
		}

		for _, pe := range pendingTransfers {
			item, exists := transferDetails[pe.msgId]
			if !exists {
				if !pe.submitPending {
					auditErrors.Inc()
					acct.logger.Error("acctaudit: query did not return status for transfer, this should not happen, resubmitting it", zap.String("msgId", pe.msgId))
					acct.submitObservation(pe)
				} else {
					acct.logger.Debug("acctaudit: query did not return status for transfer we have not submitted yet, ignoring it", zap.String("msgId", pe.msgId))
				}

				continue
			}

			if item.Status.Committed != nil {
				digest := hex.EncodeToString(item.Status.Committed.Digest)
				if pe.digest == digest {
					acct.logger.Info("acctaudit: audit determined that transfer has been committed, publishing it", zap.String("msgId", pe.msgId))
					acct.handleCommittedTransfer(pe.msgId)
				} else {
					digestMismatches.Inc()
					acct.logger.Error("acctaudit: audit detected a digest mismatch, dropping transfer", zap.String("msgId", pe.msgId), zap.String("ourDigest", pe.digest), zap.String("reportedDigest", digest))
					acct.deletePendingTransfer(pe.msgId)
				}
			} else if item.Status.Pending != nil {
				acct.logger.Debug("acctaudit: contract says transfer is still pending", zap.String("msgId", pe.msgId))
			} else if !pe.submitPending {
				auditErrors.Inc()
				acct.logger.Error("acctaudit: contract does not know about pending transfer, resubmitting it", zap.String("msgId", pe.msgId))
				acct.submitObservation(pe)
			}
		}
	}

	acct.logger.Debug("acctaudit: exiting performAudit")
}

// handleMissingObservation submits a reobservation request if appropriate.
func (acct *Accountant) handleMissingObservation(mo MissingObservation) {
	// It's possible we received this transfer after we built the temporary map. If so, we don't want to do a reobservation.
	if acct.transferNowExists(mo) {
		acct.logger.Debug("acctaudit: contract reported unknown observation as missing but it is now in our pending map, ignoring it", zap.Uint16("chainId", mo.ChainId), zap.String("txHash", hex.EncodeToString(mo.TxHash)))
		return
	}

	acct.logger.Debug("acctaudit: contract reported unknown observation as missing, requesting reobservation", zap.Uint16("chainId", mo.ChainId), zap.String("txHash", hex.EncodeToString(mo.TxHash)))
	msg := &gossipv1.ObservationRequest{ChainId: uint32(mo.ChainId), TxHash: mo.TxHash}
	acct.obsvReqWriteC <- msg
}

// transferNowExists checks to see if a missed observation exists in the pending transfer map. It grabs the lock.
func (acct *Accountant) transferNowExists(mo MissingObservation) bool {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	chanId := vaa.ChainID(mo.ChainId)
	txHash := ethCommon.BytesToHash(mo.TxHash)
	for _, pe := range acct.pendingTransfers {
		if (pe.msg.EmitterChain == chanId) && (pe.msg.TxHash == txHash) {
			return true
		}
	}

	return false
}

// queryMissingObservations queries the contract for the set of observations it thinks are missing for this guardian.
func (acct *Accountant) queryMissingObservations() ([]MissingObservation, error) {
	gs := acct.gst.Get()
	if gs == nil {
		return nil, fmt.Errorf("failed to get guardian set")
	}

	guardianIndex, found := gs.KeyIndex(acct.guardianAddr)
	if !found {
		return nil, fmt.Errorf("failed to get guardian index")
	}

	query := fmt.Sprintf(`{"missing_observations":{"guardian_set": %d, "index": %d}}`, gs.Index, guardianIndex)
	acct.logger.Debug("acctaudit: submitting missing_observations query", zap.String("query", query))
	resp, err := acct.wormchainConn.SubmitQuery(acct.ctx, acct.contract, []byte(query))
	if err != nil {
		return nil, fmt.Errorf("missing_observations query failed: %w", err)
	}

	var ret MissingObservationsResponse
	if err := json.Unmarshal(resp.Data, &ret); err != nil {
		return nil, fmt.Errorf("failed to parse missing_observations response: %w", err)
	}

	acct.logger.Debug("acctaudit: missing_observations query response", zap.Int("numEntries", len(ret.Missing)), zap.String("result", string(resp.Data)))
	return ret.Missing, nil
}

// queryBatchTransferStatus queries the status of the specified transfers and returns a map keyed by transfer key (as a string) to the status.
func (acct *Accountant) queryBatchTransferStatus(keys []TransferKey) (map[string]TransferDetails, error) {
	bytes, err := json.Marshal(keys)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keys: %w", err)
	}

	query := fmt.Sprintf(`{"batch_transfer_status":%s}`, string(bytes))
	acct.logger.Debug("acctaudit: submitting batch_transfer_status query", zap.String("query", query))
	resp, err := acct.wormchainConn.SubmitQuery(acct.ctx, acct.contract, []byte(query))
	if err != nil {
		return nil, fmt.Errorf("batch_transfer_status query failed: %w", err)
	}

	var response BatchTransferStatusResponse
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	ret := make(map[string]TransferDetails)
	for _, item := range response.Details {
		ret[item.Key.String()] = item
	}

	acct.logger.Debug("acctaudit: batch_transfer_status query response", zap.Int("numEntries", len(ret)), zap.String("result", string(resp.Data)))
	return ret, nil
}
