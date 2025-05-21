package near

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type NearWormholePublishEvent struct {
	Standard    string `json:"standard"`
	Event       string `json:"event"`
	Data        string `json:"data"`
	Nonce       uint32 `json:"nonce"`
	Emitter     string `json:"emitter"`
	Seq         uint64 `json:"seq"`
	BlockHeight uint64 `json:"block"`
}

// processTx fetches a transaction's receipt_outcomes and looks for wormhole messages in it.
// we go through all receipt outcomes (result.receipts_outcome) and look for log emissions from the Wormhole core contract.
// sender_account_id is required to help determine which shard to query.
func (e *Watcher) processTx(logger *zap.Logger, ctx context.Context, job *transactionProcessingJob) error {
	logger.Debug("processTx", zap.String("log_msg_type", "info_process_tx"), zap.String("tx_hash", job.txHash))

	tx_receipts, err := e.nearAPI.GetTxStatus(ctx, job.txHash, job.senderAccountId)

	if err != nil {
		return err
	}

	receiptOutcomes := gjson.ParseBytes(tx_receipts).Get("result.receipts_outcome")

	if !receiptOutcomes.Exists() {
		// no outcomes means nothing to look at
		logger.Debug("processTx: No receipt outcomes", zap.String("tx_hash", job.txHash))
		return nil
	}

	for _, receiptOutcome := range receiptOutcomes.Array() {
		err = e.processOutcome(logger, ctx, job, receiptOutcome)
		if err != nil {
			logger.Debug("ProcessOutcome error: ", zap.Error(err))
			return err
		}
	}
	return nil
}

func (e *Watcher) processOutcome(logger *zap.Logger, ctx context.Context, job *transactionProcessingJob, receiptOutcome gjson.Result) error {
	outcome := receiptOutcome.Get("outcome")
	if !outcome.Exists() {
		logger.Warn("NEAR RPC malformed response: receipts_outcome.outcome does not exist", zap.String("error_type", "nearapi_inconsistent"), zap.String("json", receiptOutcome.Str))
		return errors.New("NEAR RPC malformed response")
	}

	executor_id := outcome.Get("executor_id")
	if !executor_id.Exists() {
		logger.Warn("NEAR RPC malformed response: receipts_outcome.outcome does not exist", zap.String("error_type", "nearapi_inconsistent"), zap.String("json", receiptOutcome.Str))
		return errors.New("NEAR RPC malformed response: receipts_outcome.outcome does not exist")
	}

	// SECURITY CRITICAL: Check that the outcome relates to the Wormhole core contract on NEAR.
	// according to near source documentation, executor_id is the id of the account on which the execution happens:
	// for transaction this is signer_id
	// for receipt this is receiver_id, i.e. the account on which the receipt has been applied
	if executor_id.String() == "" || executor_id.String() != e.wormholeAccount {
		return nil
	}

	logger.Debug("Found a Wormhole Transaction... Now checking if it's a valid log emission.", zap.String("tx_hash", job.txHash))

	outcomeBlockHash := receiptOutcome.Get("block_hash")
	if !outcomeBlockHash.Exists() {
		logger.Warn("NEAR RPC malformed response: receipts_outcome.block_hash does not exist", zap.String("error_type", "nearapi_inconsistent"), zap.String("json", receiptOutcome.Str))
		return errors.New("NEAR RPC malformed response: receipts_outcome.block_hash does not exist")
	}

	l := outcome.Get("logs")
	if !l.Exists() {
		logger.Warn("NEAR RPC malformed response: receipts_outcome.outcome.logs does not exist", zap.String("error_type", "nearapi_inconsistent"), zap.String("json", receiptOutcome.Str))
		return errors.New("NEAR RPC malformed response: receipts_outcome.outcome.logs does not exist")
	}

	// SECURITY CRITICAL: Check that block has been finalized.
	outcomeBlockHeader, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		// If it has not, we return an error such that this transaction can be put back into the queue.
		return fmt.Errorf("block %s not finalized yet", outcomeBlockHash.String())
	}

	successValue := outcome.Get("status.SuccessValue")
	if !successValue.Exists() || successValue.String() == "" {
		return errors.New("outcome.status.SuccessValue does not exist")
	}

	for _, log := range l.Array() {
		err := e.processWormholeLog(logger, ctx, job, outcomeBlockHeader, successValue.String(), log)
		if err != nil {
			// SECURITY defense-in-depth: If one of the logs is malformed, we skip processing the other logs for defense in depth
			return err
		}
	}
	return nil // SUCCESS
}

func (e *Watcher) processWormholeLog(logger *zap.Logger, _ context.Context, job *transactionProcessingJob, outcomeBlockHeader nearapi.BlockHeader, successValue string, log gjson.Result) error {
	event := log.String()

	// SECURITY CRITICAL: Ensure that we're reading a correct log message.
	// Unfortunately, NEAR does not yet support structured event emission like Ethereum.
	if !strings.HasPrefix(event, "EVENT_JSON:") {
		return nil
	}

	eventJsonStr := event[11:]

	logger.Info("event", zap.String("log_msg_type", "wormhole_event"), zap.String("event", eventJsonStr))

	// SECURITY: Wormhole is following NEP-297 (https://nomicon.io/Standards/EventsFormat)
	// First, check that we're looking at a "publish" event type from the "wormhole" standard.
	if !isWormholePublishEvent(logger, eventJsonStr) {
		return nil
	}

	// SECURITY: If we get this far, the checks below should be true, otherwise something has seriously gone wrong.

	var pubEvent NearWormholePublishEvent
	if err := json.Unmarshal([]byte(eventJsonStr), &pubEvent); err != nil {
		logger.Error("Wormhole publish event malformed", zap.String("error_type", "malformed_wormhole_event"), zap.String("json", eventJsonStr))
		return errors.New("Wormhole publish event malformed")
	}

	if pubEvent.Standard != "wormhole" || pubEvent.Event != "publish" || pubEvent.Emitter == "" || pubEvent.Seq <= 0 || pubEvent.BlockHeight == 0 {
		logger.Error("Wormhole publish event malformed", zap.String("error_type", "malformed_wormhole_event"), zap.String("json", eventJsonStr))
		return errors.New("Wormhole publish event malformed")
	}

	successValueUint64, err := successValueToUint64(successValue)

	// SECURITY defense-in-depth: check that outcome.status.SuccessValue should equal to the base64 encoded sequence number
	if err != nil || successValueUint64 == 0 || successValueUint64 != pubEvent.Seq {
		logger.Error(
			"SuccessValue does not match sequence number",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("SuccessValue", successValue),
			zap.Uint64("int(SuccessValue)", successValueUint64),
			zap.Uint64("log.seq", pubEvent.Seq),
		)
		return errors.New("Wormhole publish event.seq does not match SuccessValue")
	}

	// SECURITY: For defense-in-depth, check that the block height from the event matches the block height from the RPC node
	if pubEvent.BlockHeight != outcomeBlockHeader.Height {
		logger.Error(
			"Wormhole publish event.block does not equal receipt_outcome[x].block_height",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.Uint64("event.block", pubEvent.BlockHeight),
			zap.Uint64("receipt_outcome[x].block_height", outcomeBlockHeader.Height),
		)
		return errors.New("Wormhole publish event.block does not equal receipt_outcome[x].block_height")
	}

	// SECURITY: extract emitter address and ensure that it has the correct format
	emitter, err := hex.DecodeString(pubEvent.Emitter)
	if err != nil {
		return err
	}

	// emitter is sha256(account_name), so it should be 32 bytes long.
	if len(emitter) != 32 {
		logger.Error(
			"Wormhole publish event malformed",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("json", eventJsonStr),
			zap.String("field", "emitter"),
		)
		return errors.New("Wormhole publish event malformed")
	}

	// Assemble the Message Publication Event
	var a vaa.Address
	copy(a[:], emitter)

	txHashBytes, err := base58.Decode(job.txHash)
	if err != nil {
		return err
	}

	if len(txHashBytes) != 32 {
		logger.Error(
			"Transaction hash is not 32 bytes",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("txHash", job.txHash),
		)
		return errors.New("Transaction hash is not 32 bytes")
	}

	var txHashEthFormat = eth_common.BytesToHash(txHashBytes)

	pl, err := hex.DecodeString(pubEvent.Data)
	if err != nil {
		return err
	}

	if len(pl)*2 != len(pubEvent.Data) {
		logger.Error(
			"Wormhole publish event malformed",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("field", "data"),
			zap.String("data", pubEvent.Data),
		)
		return errors.New("Wormhole publish event malformed")
	}

	// SECURITY the timestamp of an observation is the timestamp of the block in which the wormhole core receipt has been finalized.
	ts := outcomeBlockHeader.Timestamp

	observation := &common.MessagePublication{
		TxID:             txHashEthFormat.Bytes(),
		Timestamp:        time.Unix(int64(ts), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            pubEvent.Nonce,
		Sequence:         pubEvent.Seq,
		EmitterChain:     vaa.ChainIDNear,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: 0,
		IsReobservation:  job.isReobservation,
	}

	if job.isReobservation {
		watchers.ReobservationsByChain.WithLabelValues("near", "std").Inc()
	}

	// tell everyone about it
	job.hasWormholeMsg = true

	e.eventChan <- EVENT_NEAR_MESSAGE_CONFIRMED

	logger.Info("message observed",
		zap.String("log_msg_type", "wormhole_event_success"),
		zap.Uint64("ts", ts),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint32("nonce", observation.Nonce),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress),
		zap.Binary("payload", observation.Payload),
		zap.Uint8("consistency_level", observation.ConsistencyLevel),
	)

	e.msgC <- observation

	return nil
}

// TODO test this code
func successValueToUint64(successValue string) (uint64, error) {
	successValueBytes, err := base64.StdEncoding.DecodeString(successValue)
	if err != nil {
		return 0, err
	}
	successValueUint64, err := strconv.ParseUint(string(successValueBytes), 10, 64)
	if err != nil {
		return 0, err
	}
	return successValueUint64, nil
}

func isWormholePublishEvent(logger *zap.Logger, eventJsonStr string) bool {
	if !gjson.Valid(eventJsonStr) {
		logger.Error(
			"event is invalid json",
			zap.String("error_type", "malformed_wormhole_event"),
			zap.String("log_msg_type", "tx_processing_error"),
			zap.String("json", eventJsonStr),
		)
		return false
	}

	eventJson := gjson.Parse(eventJsonStr)
	standard := eventJson.Get("standard")
	event_type := eventJson.Get("event")

	if standard.Exists() && standard.String() == "wormhole" && event_type.Exists() && event_type.String() == "publish" {
		return true
	}
	return false
}
