// This code audits the set of pending transfers against the state reported by the smart contract. It has a runnable that is started when the accountant initializes.
// It uses a ticker to periodically run the audit. The audit occurs in two phases that operate off of a temporary map of all pending transfers known to this guardian.
//
// The first phase involves querying the smart contract for any observations that it thinks are missing for this guardian. The audit processes everything in the
// returned results and does one of the following:
// - If the observation is in our temporary map, we resubmit an observation to the contract and delete it from our temporary map.
// - If the observation is not in the temporary map, we request a reobservation from the local watcher.
//
// The second phase consists of requesting the status from the contract for everything that is still in the temporary map. For each returned item, we do the following:
// - If the contract indicates that the transfer has been committed, we validate the digest, then publish the transfer and delete it from the map.
// - If the contract indicates that the transfer is pending, we continue to wait for it to be committed.
// - If the contract indicates any other status (most likely meaning it does not know about it), we resubmit an observation to the contract.
//
// Note that any time we are considering resubmitting an observation to the contract, we first check the "submit pending" flag. If that is set, we do not
// submit the observation to the contract, but continue to wait for it to work its way through the queue.

package accountant

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"

	"go.uber.org/zap"
)

const (
	// auditInterval indicates how often the audit runs.
	// Make this bigger than the reobservation window (11 minutes).
	auditInterval = 15 * time.Minute

	// maxSubmitPendingTime indicates how long a transfer can be in the submit pending state before the audit starts complaining about it.
	maxSubmitPendingTime = 30 * time.Minute

	// maxPendingsPerQuery is the maximum number of pending transfers to submit in a single batch_transfer_status query to avoid gas errors.
	maxPendingsPerQuery = 500
)

type (
	// MissingObservationsResponse is the result returned from the "missing_observations" query.
	MissingObservationsResponse struct {
		Missing []MissingObservation `json:"missing"`
	}

	// MissingObservation is what is returned for a single missing observation.
	MissingObservation struct {
		ChainId uint16 `json:"chain_id"`
		TxHash  []byte `json:"tx_hash"`
	}

	// BatchTransferStatusResponse contains the details returned by the "batch_transfer_status" query.
	BatchTransferStatusResponse struct {
		Details []TransferDetails `json:"details"`
	}

	// TransferDetails contains the details returned for a single transfer.
	TransferDetails struct {
		Key    TransferKey     `json:"key"`
		Status *TransferStatus `json:"status"`
	}

	// TransferStatus contains the status returned for a transfer.
	TransferStatus struct {
		Committed *TransferStatusCommitted `json:"committed"`
		Pending   *[]TransferStatusPending `json:"pending"`
	}

	// TransferStatusCommitted contains the data returned for a committed transfer.
	TransferStatusCommitted struct {
		Data   TransferData `json:"data"`
		Digest []byte       `json:"digest"`
	}

	// TransferData contains the detailed data returned for a committed transfer.
	TransferData struct {
		Amount         *cosmossdk.Int `json:"amount"`
		TokenChain     uint16         `json:"token_chain"`
		TokenAddress   vaa.Address    `json:"token_address"`
		RecipientChain uint16         `json:"recipient_chain"`
	}

	// TransferStatusPending contains the data returned for a committed transfer.
	TransferStatusPending struct {
		Digest           []byte `json:"digest"`
		TxHash           []byte `json:"tx_hash"`
		Signatures       string `json:"signatures"`
		GuardianSetIndex uint32 `json:"guardian_set_index"`
		EmitterChain     uint16 `json:"emitter_chain"`
	}
)

func (mo MissingObservation) String() string {
	return fmt.Sprintf("%d-%s", mo.ChainId, hex.EncodeToString(mo.TxHash))
}

// makeAuditKey creates an audit map key from a missing observation.
func (mo *MissingObservation) makeAuditKey() string {
	return fmt.Sprintf("%d-%s", mo.ChainId, strings.TrimPrefix(hex.EncodeToString(mo.TxHash[:]), "0x"))
}

// makeAuditKey creates an audit map key from a pending observation entry.
func (pe *pendingEntry) makeAuditKey() string {
	return fmt.Sprintf("%d-%s", pe.msg.EmitterChain, strings.TrimPrefix(pe.msg.TxHash.String(), "0x"))
}

// audit is the runnable that executes the audit each interval.
func (acct *Accountant) audit(ctx context.Context) error {
	ticker := time.NewTicker(auditInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			acct.runAudit()
		}
	}
}

// runAudit is the entry point for the audit of the pending transfer map. It creates a temporary map of all pending transfers and invokes the main audit function.
func (acct *Accountant) runAudit() {
	tmpMap := acct.createAuditMap(false)
	acct.logger.Debug("in AuditPendingTransfers: starting base audit", zap.Int("numPending", len(tmpMap)))
	acct.performAudit(tmpMap, acct.wormchainConn, acct.contract)
	acct.logger.Debug("in AuditPendingTransfers: finished base audit")

	tmpMap = acct.createAuditMap(true)
	acct.logger.Debug("in AuditPendingTransfers: starting ntt audit", zap.Int("numPending", len(tmpMap)))
	acct.performAudit(tmpMap, acct.nttWormchainConn, acct.nttContract)
	acct.logger.Debug("in AuditPendingTransfers: finished ntt audit")
}

// createAuditMap creates a temporary map of all pending transfers. It grabs the pending transfer lock.
func (acct *Accountant) createAuditMap(isNTT bool) map[string]*pendingEntry {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	tmpMap := make(map[string]*pendingEntry)
	for _, pe := range acct.pendingTransfers {
		if pe.isNTT == isNTT {
			if pe.hasBeenPendingForTooLong() {
				auditErrors.Inc()
				acct.logger.Error("transfer has been in the submit pending state for too long", zap.Stringer("lastUpdateTime", pe.updTime()))
			}
			key := pe.makeAuditKey()
			acct.logger.Debug("will audit pending transfer", zap.String("msgId", pe.msgId), zap.String("moKey", key), zap.Bool("submitPending", pe.submitPending()), zap.Stringer("lastUpdateTime", pe.updTime()))
			tmpMap[key] = pe
		}
	}

	return tmpMap
}

// hasBeenPendingForTooLong determines if a transfer has been in the "submit pending" state for too long.
func (pe *pendingEntry) hasBeenPendingForTooLong() bool {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()
	return pe.state.submitPending && time.Since(pe.state.updTime) > maxSubmitPendingTime
}

// performAudit audits the temporary map against the smart contract. It is meant to be run in a go routine. It takes a temporary map of all pending transfers
// and validates that against what is reported by the smart contract. For more details, please see the prologue of this file.
func (acct *Accountant) performAudit(tmpMap map[string]*pendingEntry, wormchainConn AccountantWormchainConn, contract string) {
	acct.logger.Debug("entering performAudit", zap.String("contract", contract))
	missingObservations, err := acct.queryMissingObservations(wormchainConn, contract)
	if err != nil {
		acct.logger.Error("unable to perform audit, failed to query missing observations", zap.Error(err))
		for _, pe := range tmpMap {
			acct.logger.Error("unsure of status of pending transfer due to query error", zap.String("msgId", pe.msgId))
		}
		return
	}

	for _, mo := range missingObservations {
		key := mo.makeAuditKey()
		pe, exists := tmpMap[key]
		if exists {
			if acct.submitObservation(pe) {
				auditErrors.Inc()
				acct.logger.Error("contract reported pending observation as missing, resubmitted it", zap.String("msgID", pe.msgId))
			} else {
				acct.logger.Info("contract reported pending observation as missing but it is queued up to be submitted, skipping it", zap.String("msgID", pe.msgId))
			}

			delete(tmpMap, key)
		} else {
			acct.handleMissingObservation(mo)
		}
	}

	if len(tmpMap) != 0 {
		var keys []TransferKey
		var pendingTransfers []*pendingEntry
		for _, pe := range tmpMap {
			keys = append(keys, TransferKey{EmitterChain: uint16(pe.msg.EmitterChain), EmitterAddress: pe.msg.EmitterAddress, Sequence: pe.msg.Sequence})
			pendingTransfers = append(pendingTransfers, pe)
		}

		transferDetails, err := acct.queryBatchTransferStatus(keys, wormchainConn, contract)
		if err != nil {
			acct.logger.Error("unable to finish audit, failed to query for transfer statuses", zap.Error(err))
			for _, pe := range tmpMap {
				acct.logger.Error("unsure of status of pending transfer due to query error", zap.String("msgId", pe.msgId))
			}
			return
		}

		for _, pe := range pendingTransfers {
			status, exists := transferDetails[pe.msgId]
			if !exists {
				if acct.submitObservation(pe) {
					auditErrors.Inc()
					acct.logger.Error("query did not return status for transfer, this should not happen, resubmitted it", zap.String("msgId", pe.msgId))
				} else {
					acct.logger.Info("query did not return status for transfer we have not submitted yet, ignoring it", zap.String("msgId", pe.msgId))
				}

				continue
			}

			if status == nil {
				// This is the case when the contract does not know about a transfer. Resubmit it.
				if acct.submitObservation(pe) {
					auditErrors.Inc()
					acct.logger.Error("contract does not know about pending transfer, resubmitted it", zap.String("msgId", pe.msgId))
				}
			} else if status.Committed != nil {
				digest := hex.EncodeToString(status.Committed.Digest)
				if pe.digest == digest {
					acct.logger.Warn("audit determined that transfer has been committed, publishing it", zap.String("msgId", pe.msgId))
					acct.handleCommittedTransfer(pe.msgId)
				} else {
					digestMismatches.Inc()
					acct.logger.Error("audit detected a digest mismatch, dropping transfer", zap.String("msgId", pe.msgId), zap.String("ourDigest", pe.digest), zap.String("reportedDigest", digest))
					acct.deletePendingTransfer(pe.msgId)
				}
			} else if status.Pending != nil {
				acct.logger.Debug("contract says transfer is still pending", zap.String("msgId", pe.msgId))
			} else {
				// This is the case when the contract does not know about a transfer. Resubmit it.
				if acct.submitObservation(pe) {
					auditErrors.Inc()
					bytes, err := json.Marshal(*status)
					if err != nil {
						acct.logger.Error("unknown status returned for pending transfer, resubmitted it", zap.String("msgId", pe.msgId), zap.Error(err))
					} else {
						acct.logger.Error("unknown status returned for pending transfer, resubmitted it", zap.String("msgId", pe.msgId), zap.String("status", string(bytes)))
					}
				}
			}
		}
	}

	acct.logger.Debug("exiting performAudit")
}

// handleMissingObservation submits a local reobservation request. It relies on the reobservation code to throttle requests.
func (acct *Accountant) handleMissingObservation(mo MissingObservation) {
	acct.logger.Warn("contract reported unknown observation as missing, requesting local reobservation", zap.Stringer("moKey", mo))
	msg := &gossipv1.ObservationRequest{ChainId: uint32(mo.ChainId), TxHash: mo.TxHash}

	select {
	case acct.obsvReqWriteC <- msg:
		acct.logger.Debug("submitted local reobservation", zap.Stringer("moKey", mo))
	default:
		acct.logger.Error("unable to submit local reobservation because the channel is full, will try next interval", zap.Stringer("moKey", mo))
	}
}

// queryMissingObservations queries the contract for the set of observations it thinks are missing for this guardian.
func (acct *Accountant) queryMissingObservations(wormchainConn AccountantWormchainConn, contract string) ([]MissingObservation, error) {
	gs := acct.gst.Get()
	if gs == nil {
		return nil, fmt.Errorf("failed to get guardian set")
	}

	guardianIndex, found := gs.KeyIndex(acct.guardianAddr)
	if !found {
		return nil, fmt.Errorf("failed to get guardian index")
	}

	query := fmt.Sprintf(`{"missing_observations":{"guardian_set": %d, "index": %d}}`, gs.Index, guardianIndex)
	acct.logger.Debug("submitting missing_observations query", zap.String("query", query))
	respBytes, err := wormchainConn.SubmitQuery(acct.ctx, contract, []byte(query))
	if err != nil {
		return nil, fmt.Errorf("missing_observations query failed: %w, %s", err, query)
	}

	var ret MissingObservationsResponse
	if err := json.Unmarshal(respBytes, &ret); err != nil {
		return nil, fmt.Errorf("failed to parse missing_observations response: %w, resp: %s", err, string(respBytes))
	}

	acct.logger.Debug("missing_observations query response", zap.Int("numEntries", len(ret.Missing)), zap.String("result", string(respBytes)))
	return ret.Missing, nil
}

// queryConn allows us to mock the SubmitQuery call.
type queryConn interface {
	SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error)
}

// queryBatchTransferStatus queries the status of the specified transfers and returns a map keyed by transfer key (as a string) to the status.
func (acct *Accountant) queryBatchTransferStatus(keys []TransferKey, wormchainConn AccountantWormchainConn, contract string) (map[string]*TransferStatus, error) {
	return queryBatchTransferStatusWithConn(acct.ctx, acct.logger, wormchainConn, contract, keys)
}

// queryBatchTransferStatus is a free function that queries the status of the specified transfers and returns a map keyed by transfer key (as a string)
// to the status. If there are too many keys to be queried, it breaks them up into smaller chunks (based on the maxPendingsPerQuery constant).
func queryBatchTransferStatusWithConn(
	ctx context.Context,
	logger *zap.Logger,
	qc queryConn,
	contract string,
	keys []TransferKey,
) (map[string]*TransferStatus, error) {
	if len(keys) <= maxPendingsPerQuery {
		return queryBatchTransferStatusForChunk(ctx, logger, qc, contract, keys)
	}

	// Break the large batch into smaller chunks. Found this logic here: https://freshman.tech/snippets/go/split-slice-into-chunks/
	ret := make(map[string]*TransferStatus)
	for i := 0; i < len(keys); i += maxPendingsPerQuery {
		end := i + maxPendingsPerQuery

		// Necessary check to avoid slicing beyond slice capacity.
		if end > len(keys) {
			end = len(keys)
		}

		chunkRet, err := queryBatchTransferStatusForChunk(ctx, logger, qc, contract, keys[i:end])
		if err != nil {
			return nil, err
		}

		for key, item := range chunkRet {
			ret[key] = item
		}
	}

	return ret, nil
}

// queryBatchTransferStatus is a free function that queries the status of a chunk of transfers and returns a map keyed by transfer key (as a string) to the status.
func queryBatchTransferStatusForChunk(
	ctx context.Context,
	logger *zap.Logger,
	qc queryConn,
	contract string,
	keys []TransferKey,
) (map[string]*TransferStatus, error) {
	bytes, err := json.Marshal(keys)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keys: %w", err)
	}

	query := fmt.Sprintf(`{"batch_transfer_status":%s}`, string(bytes))
	logger.Debug("submitting batch_transfer_status query", zap.String("query", query))
	respBytes, err := qc.SubmitQuery(ctx, contract, []byte(query))
	if err != nil {
		return nil, fmt.Errorf("batch_transfer_status query failed: %w, %s", err, query)
	}

	var response BatchTransferStatusResponse
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, resp: %s", err, string(respBytes))
	}

	ret := make(map[string]*TransferStatus)
	for _, item := range response.Details {
		ret[item.Key.String()] = item.Status
	}

	logger.Debug("batch_transfer_status query response", zap.Int("numEntries", len(ret)), zap.String("result", string(respBytes)))
	return ret, nil
}
