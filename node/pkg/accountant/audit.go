// This code audits the set of pending transfers against the state reported by the smart contract. It has a runnable that is started when the accountant initializes.
// It uses a ticker to periodically run the audit. The audit occurs in two phases that operate off of a temporary map of all pending transfers known to this guardian.
//
// The first phase involves querying the smart contract for all pending transfers using the "all_pending_transfers" query. For each pending transfer returned:
// - If this guardian has already signed (based on the signatures bitmask), it is skipped.
// - If the transfer is in our local pending map, we resubmit our observation to the contract.
// - If the transfer is not in our local map, we request a reobservation from the local watcher.
//
// The second phase handles transfers that are in our local map but were NOT found in the contract's pending list. For each such transfer, we query
// the contract for its status using "batch_transfer_status":
// - If the contract indicates the transfer has been committed, we validate the digest and publish the transfer.
// - If the contract indicates the transfer is pending, we continue to wait.
// - If the contract indicates any other status (or doesn't know about it), we resubmit our observation.
//
// Note that any time we are considering resubmitting an observation to the contract, we first check the "submit pending" flag. If that is set, we do not
// submit the observation to the contract, but continue to wait for it to work its way through the queue.

package accountant

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
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

	// allPendingTransfersPageSize is the page size for all_pending_transfers query.
	allPendingTransfersPageSize = 500
)

type (
	// MissingObservation represents a single observation that the contract expects but we haven't submitted.
	// Used when requesting reobservation from the local watcher.
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

	// AllPendingTransfersResponse is the result from "all_pending_transfers" query.
	AllPendingTransfersResponse struct {
		Pending []PendingTransfer `json:"pending"`
	}

	// PendingTransfer represents a single pending transfer from the contract.
	PendingTransfer struct {
		Key  TransferKey           `json:"key"`
		Data []PendingTransferData `json:"data"`
	}

	// PendingTransferData contains observation data for a pending transfer.
	PendingTransferData struct {
		Digest           []byte `json:"digest"`
		TxHash           []byte `json:"tx_hash"`
		Signatures       string `json:"signatures"` // u128 as decimal string
		GuardianSetIndex uint32 `json:"guardian_set_index"`
		EmitterChain     uint16 `json:"emitter_chain"`
	}
)

func (mo MissingObservation) String() string {
	return fmt.Sprintf("%d-%s", mo.ChainId, hex.EncodeToString(mo.TxHash))
}

// makeAuditKey creates an audit map key from a pending observation entry.
func (pe *pendingEntry) makeAuditKey() string {
	return fmt.Sprintf("%d-%s", pe.msg.EmitterChain, strings.TrimPrefix(pe.msg.TxIDString(), "0x"))
}

// hasGuardianSigned checks if a guardian has signed based on the signatures bitmask.
// The signatures field is a u128 represented as a decimal string where each bit
// corresponds to a guardian index.
func hasGuardianSigned(signatures string, guardianIndex int) bool {
	sigInt, ok := new(big.Int).SetString(signatures, 10) //nolint:mnd // Base 10 because the signatures are encoded as a decimal string
	if !ok {
		return false // Assume not signed if we can't parse
	}
	return sigInt.Bit(guardianIndex) == 1
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
			acct.runAudit(ctx)
		}
	}
}

// runAudit is the entry point for the audit of the pending transfer map. It creates a temporary map of all pending transfers and invokes the main audit function.
func (acct *Accountant) runAudit(ctx context.Context) {
	if acct.baseEnabled() {
		knownPendingTransferMap := acct.createAuditMap(false)
		acct.logger.Debug("in AuditPendingTransfers: starting base audit", zap.Int("numPending", numPendingEntries(knownPendingTransferMap)))
		acct.performAudit(ctx, knownPendingTransferMap, acct.wormchainConn, acct.contract)
		acct.logger.Debug("in AuditPendingTransfers: finished base audit")
	}

	if acct.nttEnabled() {
		knownPendingNttTransferMap := acct.createAuditMap(true)
		acct.logger.Debug("in AuditPendingTransfers: starting ntt audit", zap.Int("numPending", numPendingEntries(knownPendingNttTransferMap)))
		acct.performAudit(ctx, knownPendingNttTransferMap, acct.nttWormchainConn, acct.nttContract)
		acct.logger.Debug("in AuditPendingTransfers: finished ntt audit")
	}
}

// numPendingEntries returns the total number of non-nil pending entries across all keys
// in the audit map. Nil entries are skipped so a slice of nils does not inflate the count.
func numPendingEntries(tmpMap map[string][]*pendingEntry) int {
	n := 0
	for _, entries := range tmpMap {
		for _, pe := range entries {
			if pe != nil {
				n++
			}
		}
	}
	return n
}

// createAuditMap creates a temporary map of all pending transfers. It grabs the pending transfer lock.
// The map is keyed by chain-txHash (the audit key). Multiple transfers may share the same key if they
// originate from the same transaction, so each key maps to a slice of pending entries.
func (acct *Accountant) createAuditMap(isNTT bool) map[string][]*pendingEntry {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	knownPendingTransferMap := make(map[string][]*pendingEntry)
	for _, pe := range acct.pendingTransfers {
		// Skip over nil entries
		if pe == nil {
			continue
		}
		if pe.isNTT == isNTT {
			if pe.hasBeenPendingForTooLong() {
				auditErrors.Inc()
				acct.logger.Error("transfer has been in the submit pending state for too long", zap.Stringer("lastUpdateTime", pe.updTime()))
			}
			key := pe.makeAuditKey()
			acct.logger.Debug("will audit pending transfer", zap.String("msgId", pe.msgId), zap.String("moKey", key), zap.Bool("submitPending", pe.submitPending()), zap.Stringer("lastUpdateTime", pe.updTime()))
			knownPendingTransferMap[key] = append(knownPendingTransferMap[key], pe)
		}
	}

	return knownPendingTransferMap
}

// hasBeenPendingForTooLong determines if a transfer has been in the "submit pending" state for too long.
func (pe *pendingEntry) hasBeenPendingForTooLong() bool {
	pe.stateLock.Lock()
	defer pe.stateLock.Unlock()
	return pe.state.submitPending && time.Since(pe.state.updTime) > maxSubmitPendingTime
}

// performAudit audits the temporary map against the smart contract. It is meant to be run in a go routine. It takes a temporary map of all pending transfers
// and validates that against what is reported by the smart contract. For more details, please see the prologue of this file.
func (acct *Accountant) performAudit(ctx context.Context, knownPendingTransferMap map[string][]*pendingEntry, wormchainConn AccountantWormchainConn, contract string) {
	acct.logger.Debug("entering performAudit", zap.String("contract", contract))

	gs := acct.gst.Get()
	if gs == nil {
		acct.logger.Error("unable to perform audit, failed to get guardian set")
		return
	}

	guardianIndex, found := gs.KeyIndex(acct.guardianAddr)
	if !found {
		acct.logger.Error("unable to perform audit, failed to get guardian index")
		return
	}

	// Query all pending transfers with pagination
	pendingTransfers, err := acct.queryAllPendingTransfers(wormchainConn, contract)
	if err != nil {
		acct.logger.Error("unable to perform audit, failed to query pending transfers", zap.Error(err))
		for _, entries := range knownPendingTransferMap {
			for _, pe := range entries {
				// We already do a nil check in `createAuditMap`, but we'll do the same here
				if pe == nil {
					continue
				}
				acct.logger.Error("unsure of status of pending transfer due to query error", zap.String("msgId", pe.msgId))
			}
		}
		return
	}

	acct.logger.Info("audit queried pending transfers",
		zap.Int("totalPending", len(pendingTransfers)),
		zap.String("contract", contract))

	// SECURITY: knownPendingTransferMap contains only transfers that are verified through the normal message processing pipeline. Transfers returned by
	// the contract's all_pending_transfers query are untrusted external data. We must ONLY resubmit
	// observations for transfers present in knownPendingTransferMap. For transfers the contract
	// reports that we do NOT have locally, we request a reobservation from the watcher, which
	// re-verifies the transaction on-chain before it enters the signing pipeline. Never construct
	// a signable observation directly from contract response data.
	for _, pt := range pendingTransfers {
		for _, data := range pt.Data {
			// Skip if we've already signed this
			if hasGuardianSigned(data.Signatures, guardianIndex) {
				continue
			}

			// We haven't signed - build key to check our local map
			key := fmt.Sprintf("%d-%s", data.EmitterChain,
				strings.TrimPrefix(hex.EncodeToString(data.TxHash), "0x"))

			if entries, exists := knownPendingTransferMap[key]; exists {
				for _, pe := range entries {
					// We already do a nil check in `createAuditMap`, but we'll do the same here
					if pe == nil {
						continue
					}

					// We have it locally but haven't submitted successfully - resubmit
					if acct.submitObservation(ctx, pe, true) {
						auditErrors.Inc()
						acct.logger.Error("contract reported we have not signed a pending transfer, resubmitting", zap.String("msgId", pe.msgId))
					} else {
						acct.logger.Info("contract reported we have not signed a pending transfer but it is already pending submission, skipping", zap.String("msgId", pe.msgId))
					}
				}
				delete(knownPendingTransferMap, key)
			} else {
				// We don't have it locally - request reobservation
				acct.handleMissingObservation(MissingObservation{
					ChainId: data.EmitterChain,
					TxHash:  data.TxHash,
				})
			}
		}
	}

	if len(knownPendingTransferMap) == 0 {
		acct.logger.Debug("exiting performAudit with no known pending transfers left")
		return
	}

	// Anything still in knownPendingTransferMap is something WE have but the CONTRACT doesn't know about as pending.
	// It could be committed (need to publish) or unknown (need to resubmit).
	// Query the status to find out.
	var keys []TransferKey
	var localTransfers []*pendingEntry
	for _, entries := range knownPendingTransferMap {
		for _, pe := range entries {
			// We already do a nil check in `createAuditMap`, but we'll do the same here
			if pe == nil {
				continue
			}

			keys = append(keys, TransferKey{EmitterChain: uint16(pe.msg.EmitterChain), EmitterAddress: pe.msg.EmitterAddress, Sequence: pe.msg.Sequence})
			localTransfers = append(localTransfers, pe)
		}
	}
	transferDetails, err := acct.queryBatchTransferStatus(keys, wormchainConn, contract)
	if err != nil {
		acct.logger.Error("unable to finish audit, failed to query for transfer statuses", zap.Error(err))
		for _, pe := range localTransfers {
			// We already do a nil check in `createAuditMap`, but we'll do the same here
			if pe == nil {
				continue
			}
			acct.logger.Error("unsure of status of pending transfer due to query error", zap.String("msgId", pe.msgId))
		}
		return
	}

	for _, pe := range localTransfers {
		// There should be no nil entries, but we'll skip to be safe
		if pe == nil {
			continue
		}
		status, exists := transferDetails[pe.msgId]
		if !exists {
			if acct.submitObservation(ctx, pe, true) {
				auditErrors.Inc()
				acct.logger.Error("query did not return status for transfer, this should not happen, resubmitted it", zap.String("msgId", pe.msgId))
			} else {
				acct.logger.Info("query did not return status for transfer we have not submitted yet, ignoring it", zap.String("msgId", pe.msgId))
			}

			continue
		}

		if status == nil {
			// This is the case when the contract does not know about a transfer. Resubmit it.
			if acct.submitObservation(ctx, pe, true) {
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
			if acct.submitObservation(ctx, pe, true) {
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

	acct.logger.Debug("exiting performAudit")
}

// handleMissingObservation submits a local reobservation request. It relies on the reobservation code to throttle requests.
func (acct *Accountant) handleMissingObservation(mo MissingObservation) {
	acct.logger.Warn("contract reported unknown observation as missing, requesting local reobservation", zap.Stringer("moKey", mo))
	msg := &gossipv1.ObservationRequest{ChainId: uint32(mo.ChainId), TxHash: mo.TxHash, Timestamp: time.Now().UnixNano()}

	select {
	case acct.obsvReqWriteC <- msg:
		acct.logger.Debug("submitted local reobservation", zap.Stringer("moKey", mo))
	default:
		acct.logger.Error("unable to submit local reobservation because the channel is full, will try next interval", zap.Stringer("moKey", mo))
	}
}

// queryConn allows us to mock the SubmitQuery call.
type queryConn interface {
	SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error)
}

// queryAllPendingTransfers paginates through all pending transfers from the contract.
func (acct *Accountant) queryAllPendingTransfers(
	wormchainConn AccountantWormchainConn,
	contract string,
) ([]PendingTransfer, error) {
	return queryAllPendingTransfersWithConn(acct.ctx, acct.logger, wormchainConn, contract, allPendingTransfersPageSize)
}

// queryAllPendingTransfersWithConn is a free function that paginates through all pending transfers.
// It accepts a queryConn interface to allow mocking in tests.
func queryAllPendingTransfersWithConn(
	ctx context.Context,
	logger *zap.Logger,
	qc queryConn,
	contract string,
	pageSize int,
) ([]PendingTransfer, error) {
	var allPending []PendingTransfer
	var startAfter *TransferKey

	for {
		page, err := queryAllPendingTransfersPage(ctx, logger, qc, contract, startAfter, pageSize)
		if err != nil {
			return nil, err
		}

		allPending = append(allPending, page...)

		if len(page) < pageSize {
			// Last page
			break
		}

		// Set cursor for next page
		lastKey := page[len(page)-1].Key
		startAfter = &lastKey
	}

	return allPending, nil
}

// queryAllPendingTransfersPage queries a single page of pending transfers.
func queryAllPendingTransfersPage(
	ctx context.Context,
	logger *zap.Logger,
	qc queryConn,
	contract string,
	startAfter *TransferKey,
	limit int,
) ([]PendingTransfer, error) {
	var query string
	if startAfter == nil {
		query = fmt.Sprintf(`{"all_pending_transfers":{"limit":%d}}`, limit)
	} else {
		startAfterBytes, err := json.Marshal(startAfter)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal start_after: %w", err)
		}
		query = fmt.Sprintf(`{"all_pending_transfers":{"start_after":%s,"limit":%d}}`, string(startAfterBytes), limit)
	}

	logger.Debug("submitting all_pending_transfers query", zap.String("query", query))
	respBytes, err := qc.SubmitQuery(ctx, contract, []byte(query))
	if err != nil {
		return nil, fmt.Errorf("all_pending_transfers query failed: %w, %s", err, query)
	}

	var resp AllPendingTransfersResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse all_pending_transfers response: %w, resp: %s", err, string(respBytes))
	}

	logger.Debug("all_pending_transfers query response", zap.Int("numEntries", len(resp.Pending)))
	return resp.Pending, nil
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
