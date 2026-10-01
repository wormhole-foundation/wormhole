package accountant

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"go.uber.org/zap"
)

const (
	DefaultSubmitObservationBatchSize = 100             // Maximum observations per batch, also subject to maxSubmitObservationsMsgSize
	maxSubmitObservationsMsgSize      = 64 * 1024       // Maximum size of the marshaled submit_observations message (the wasm contract input is limited to 64KB)
	batchTimeout                      = 2 * time.Second // Time to collect observations before submitting
)

// baseWorker is the entry point for the base accountant worker.
func (acct *Accountant) baseWorker(ctx context.Context) error {
	return acct.worker(ctx, false)
}

// nttWorker is the entry point for the NTT accountant worker.
func (acct *Accountant) nttWorker(ctx context.Context) error {
	return acct.worker(ctx, true)
}

// worker listens for observation requests from the accountant and submits them to the smart contract.
func (acct *Accountant) worker(ctx context.Context, isNTT bool) error {
	subChan := acct.subChan
	wormchainConn := acct.wormchainConn
	contract := acct.contract
	prefix := SubmitObservationPrefix
	tag := "accountant"
	if isNTT {
		subChan = acct.nttSubChan
		wormchainConn = acct.nttWormchainConn
		contract = acct.nttContract
		prefix = NttSubmitObservationPrefix
		tag = "ntt-accountant"
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := acct.handleBatch(ctx, subChan, wormchainConn, contract, prefix, tag); err != nil {
				return err
			}
		}
	}
}

// handleBatch reads a batch of events from the channel, either until a timeout occurs or the batch is full,
// and submits them to the smart contract.
func (acct *Accountant) handleBatch(ctx context.Context, subChan chan *common.MessagePublication, wormchainConn AccountantWormchainConn, contract string, prefix []byte, tag string) error {
	ctx, cancel := context.WithTimeout(ctx, batchTimeout)
	defer cancel()

	msgs, err := common.ReadFromChannelWithTimeout[*common.MessagePublication](ctx, subChan, acct.submitObservationBatchSize)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("failed to read messages from channel for %s: %w", tag, err)
	}

	if len(msgs) != 0 {
		msgs = acct.removeCompleted(msgs)
	}

	if len(msgs) == 0 {
		return nil
	}

	gs := acct.gst.Get()
	if gs == nil {
		return fmt.Errorf("failed to get guardian set for %s", tag)
	}

	guardianIndex, found := gs.KeyIndex(acct.guardianAddr)
	if !found {
		return fmt.Errorf("failed to get guardian index for %s", tag)
	}

	if guardianIndex > math.MaxUint32 {
		return fmt.Errorf("guardian index greater than max uint32 %v", guardianIndex)
	}

	batches, oversized := packObservationBatches(msgs, acct.submitObservationBatchSize, maxSubmitObservationsMsgSize)
	if len(batches) > 1 {
		acct.logger.Info(fmt.Sprintf("split observations for %s into multiple batches to stay within the transaction size limit", tag), zap.Int("numMsgs", len(msgs)), zap.Int("numBatches", len(batches)))
		batchSizeSplits.Inc()
	}
	for _, msg := range oversized {
		// The message is submitted anyway (alone, so it can't take any other observations down with it) in case our size accounting is too conservative.
		acct.logger.Error(fmt.Sprintf("observation exceeds the transaction size limit for %s even in a batch by itself", tag), zap.String("msgId", msg.MessageIDString()), zap.Int("payloadLen", len(msg.Payload)))
		oversizedObservations.Inc()
	}

	for _, batch := range batches {
		acct.submitObservationsToContract(batch, gs.Index, uint32(guardianIndex), wormchainConn, contract, prefix, tag) // #nosec G115 -- This is checked above
	}
	transfersSubmitted.Add(float64(len(msgs)))
	return nil
}

// packObservationBatches partitions the messages into batches of at most maxBatchCount messages whose marshaled
// submit_observations messages each stay within maxMsgSize. Each message is placed in the first batch with enough room for it,
// so a large message that does not fit in the current batch is deferred to a later batch rather than failing the messages
// around it. A message too large to fit even in a batch by itself is returned in oversized as well as being placed in its own
// batch.
func packObservationBatches(msgs []*common.MessagePublication, maxBatchCount int, maxMsgSize int) (batches [][]*common.MessagePublication, oversized []*common.MessagePublication) {
	// batchSizes[i] is the size the marshaled observations array for batches[i] would have, including the enclosing brackets.
	var batchSizes []int
	for _, msg := range msgs {
		obsSize := marshaledObservationSize(msg)
		placed := false
		for idx := range batches {
			// Adding an observation to a batch grows the observations array by the observation plus a separating comma.
			if len(batches[idx]) < maxBatchCount && submitObservationsMsgSize(batchSizes[idx]+obsSize+jsonCommaSize) <= maxMsgSize {
				batches[idx] = append(batches[idx], msg)
				batchSizes[idx] += obsSize + jsonCommaSize
				placed = true
				break
			}
		}
		if !placed {
			if submitObservationsMsgSize(obsSize+jsonBracketsSize) > maxMsgSize {
				oversized = append(oversized, msg)
			}
			batches = append(batches, []*common.MessagePublication{msg})
			batchSizes = append(batchSizes, obsSize+jsonBracketsSize)
		}
	}
	return batches, oversized
}

// marshaledObservationSize returns the number of bytes the observation for the message occupies in the marshaled observations array.
func marshaledObservationSize(msg *common.MessagePublication) int {
	bytes, err := json.Marshal(makeObservation(msg))
	if err != nil {
		// Marshaling an Observation cannot fail, since none of its field types can produce a marshaling error.
		panic(fmt.Sprintf("failed to marshal observation: %v", err))
	}
	return len(bytes)
}

// Sizes of the JSON punctuation accounted for when computing how large a marshaled submit_observations message will be.
const (
	jsonQuotesSize   = len(`""`) // The quotes around a JSON string, such as the base64-encoded observations
	jsonBracketsSize = len(`[]`) // The brackets around a JSON array, such as the marshaled observations
	jsonCommaSize    = len(`,`)  // The comma between JSON array elements
)

// submitObservationsMsgSize returns the size of the marshaled submit_observations message for an observations array of
// obsArraySize bytes, assuming worst-case sizes for the other fields.
func submitObservationsMsgSize(obsArraySize int) int {
	// The marshaled observations array appears in the message as a quoted base64 string.
	return submitObservationsMsgOverhead + jsonQuotesSize + base64.StdEncoding.EncodedLen(obsArraySize)
}

// submitObservationsMsgOverhead is the number of bytes in a marshaled submit_observations message excluding the base64-encoded
// observations string, computed with worst-case values for the other fields.
var submitObservationsMsgOverhead = func() int {
	sig := make(SignatureBytes, 65) //nolint:mnd // Length of an ECDSA (r, s, v) signature
	for idx := range sig {
		sig[idx] = math.MaxUint8
	}
	bytes, err := json.Marshal(SubmitObservationsMsg{
		Params: SubmitObservationsParams{
			Observations:     []byte{},
			GuardianSetIndex: math.MaxUint32,
			Signature:        SignatureType{Index: math.MaxUint32, Signature: sig},
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to marshal submit observations message: %v", err))
	}
	return len(bytes) - jsonQuotesSize // Marshaling the empty observations produces just the quotes of an empty string.
}()

// removeCompleted drops any messages that are no longer in the pending transfer map. This is to handle the case where the contract reports
// that a transfer is committed while it is in the channel. There is no point in submitting the observation once the transfer is committed.
func (acct *Accountant) removeCompleted(msgs []*common.MessagePublication) []*common.MessagePublication {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()

	out := make([]*common.MessagePublication, 0, len(msgs))
	for _, msg := range msgs {
		if _, exists := acct.pendingTransfers[msg.MessageIDString()]; exists {
			out = append(out, msg)
		}
	}

	return out
}

type (
	TransferKey struct {
		EmitterChain   uint16      `json:"emitter_chain"`
		EmitterAddress vaa.Address `json:"emitter_address"`
		Sequence       uint64      `json:"sequence"`
	}

	SubmitObservationsMsg struct {
		Params SubmitObservationsParams `json:"submit_observations"`
	}

	SubmitObservationsParams struct {
		// A serialized `Vec<Observation>`. Multiple observations can be submitted together to reduce  transaction overhead.
		Observations []byte `json:"observations"`

		// The index of the guardian set used to sign the observations.
		GuardianSetIndex uint32 `json:"guardian_set_index"`

		// A signature for `observations`.
		Signature SignatureType `json:"signature"`
	}

	SignatureType struct {
		Index     uint32         `json:"index"`
		Signature SignatureBytes `json:"signature"`
	}

	SignatureBytes []uint8

	Observation struct {
		// The hash of the transaction on the emitter chain in which the transfer was performed.
		TxHash []byte `json:"tx_hash"`

		// Seconds since UNIX epoch.
		Timestamp uint32 `json:"timestamp"`

		// The nonce for the transfer.
		Nonce uint32 `json:"nonce"`

		// The source chain from which this observation was created.
		EmitterChain uint16 `json:"emitter_chain"`

		// The address on the source chain that emitted this message.
		EmitterAddress vaa.Address `json:"emitter_address"`

		// The sequence number of this observation.
		Sequence uint64 `json:"sequence"`

		// The consistency level requested by the emitter.
		ConsistencyLevel uint8 `json:"consistency_level"`

		// The serialized tokenbridge payload.
		Payload []byte `json:"payload"`
	}

	// These are used to parse the response data
	ObservationResponses []ObservationResponse

	ObservationResponse struct {
		Key    TransferKey
		Status ObservationResponseStatus
	}

	ObservationResponseStatus struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
)

var SubmitObservationPrefix = []byte("acct_sub_obsfig_000000000000000000|")
var NttSubmitObservationPrefix = []byte("ntt_acct_sub_obsfig_00000000000000|")

// makeObservation converts a message publication into the observation that is submitted to the smart contract.
func makeObservation(msg *common.MessagePublication) Observation {
	return Observation{
		TxHash:           msg.TxID,
		Timestamp:        uint32(msg.Timestamp.Unix()), // #nosec G115 -- This conversion is safe until year 2106
		Nonce:            msg.Nonce,
		EmitterChain:     uint16(msg.EmitterChain),
		EmitterAddress:   msg.EmitterAddress,
		Sequence:         msg.Sequence,
		ConsistencyLevel: msg.ConsistencyLevel,
		Payload:          msg.Payload,
	}
}

func (k TransferKey) String() string {
	return fmt.Sprintf("%v/%v/%v", k.EmitterChain, hex.EncodeToString(k.EmitterAddress[:]), k.Sequence)
}

//nolint:unparam // error is always nil but is required to satisfy the custom JSON marshal interface.
func (sb SignatureBytes) MarshalJSON() ([]byte, error) {
	var result string
	if sb == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", sb)), ",")
	}
	return []byte(result), nil
}

// submitObservationsToContract makes a call to the smart contract to submit a batch of observation requests.
// It should be called from a go routine because it can block.
func (acct *Accountant) submitObservationsToContract(msgs []*common.MessagePublication, gsIndex uint32, guardianIndex uint32, wormchainConn AccountantWormchainConn, contract string, prefix []byte, tag string) {
	txResp, err := SubmitObservationsToContract(acct.ctx, acct.logger, acct.guardianSigner, gsIndex, guardianIndex, wormchainConn, contract, prefix, msgs)
	if err != nil {
		// This means the whole batch failed. They will all get retried the next audit cycle.
		acct.logger.Error(fmt.Sprintf("failed to submit any observations in batch to %s", tag), zap.Int("numMsgs", len(msgs)), zap.Error(err))
		for idx, msg := range msgs {
			acct.logger.Error(fmt.Sprintf("failed to submit observation to %s", tag), zap.Int("idx", idx), zap.String("msgId", msg.MessageIDString()))
		}

		submitFailures.Add(float64(len(msgs)))
		acct.clearSubmitPendingFlags(msgs)
		return
	}

	responses, err := GetObservationResponses(txResp)
	if err != nil {
		// This means the whole batch failed. They will all get retried the next audit cycle.
		acct.logger.Error(fmt.Sprintf("failed to get responses from batch from %s", tag), zap.Error(err), zap.String("txResp", wormchainConn.BroadcastTxResponseToString(txResp)))
		for idx, msg := range msgs {
			acct.logger.Error(fmt.Sprintf("need to retry observation to %s", tag), zap.Int("idx", idx), zap.String("msgId", msg.MessageIDString()))
		}

		submitFailures.Add(float64(len(msgs)))
		acct.clearSubmitPendingFlags(msgs)
		return
	}

	if len(responses) != len(msgs) {
		// This means the whole batch failed. They will all get retried the next audit cycle.
		acct.logger.Error(fmt.Sprintf("number of responses from %s does not match number of messages", tag), zap.Int("numMsgs", len(msgs)), zap.Int("numResp", len(responses)), zap.Error(err))
		for idx, msg := range msgs {
			acct.logger.Error(fmt.Sprintf("need to retry observation to %s", tag), zap.Int("idx", idx), zap.String("msgId", msg.MessageIDString()))
		}

		submitFailures.Add(float64(len(msgs)))
		acct.clearSubmitPendingFlags(msgs)
		return
	}

	for _, msg := range msgs {
		msgId := msg.MessageIDString()

		status, exists := responses[msgId]
		if !exists {
			// This will get retried next audit interval.
			acct.logger.Error(fmt.Sprintf("did not receive an observation response from %s for message", tag), zap.String("msgId", msgId))
			submitFailures.Inc()
			continue
		}

		switch status.Type {
		case "pending":
			acct.logger.Info(fmt.Sprintf("transfer is pending on %s", tag), zap.String("msgId", msgId))
		case "committed":
			acct.handleCommittedTransfer(msgId)
		case "error":
			submitFailures.Inc()
			acct.handleTransferError(msgId, status.Data, "transfer failed")
		default:
			// This will get retried next audit interval.
			acct.logger.Error(fmt.Sprintf("unexpected status response from %s on observation", tag), zap.String("msgId", msgId), zap.String("status", status.Type), zap.String("text", status.Data))
			submitFailures.Inc()
		}
	}

	acct.clearSubmitPendingFlags(msgs)
}

// handleCommittedTransfer updates the pending map and publishes a committed transfer. It grabs the lock.
func (acct *Accountant) handleCommittedTransfer(msgId string) {
	acct.pendingTransfersLock.Lock()
	defer acct.pendingTransfersLock.Unlock()
	pe, exists := acct.pendingTransfers[msgId]
	if exists {
		acct.logger.Info("transfer has been committed, publishing it", zap.String("msgId", msgId))
		acct.publishTransferAlreadyLocked(pe)
		transfersApproved.Inc()
	} else {
		acct.logger.Debug("transfer has been committed but it is no longer in our map", zap.String("msgId", msgId))
	}
}

// handleTransferError is called when a transfer fails, either from a submit or an event notification. It handles insufficient balance error. It grabs the lock.
func (acct *Accountant) handleTransferError(msgId string, errText string, logText string) {
	if strings.Contains(errText, "insufficient balance") {
		balanceErrors.Inc()
		acct.logger.Error("insufficient balance error detected, dropping transfer", zap.String("msgId", msgId), zap.String("text", errText))
		acct.deletePendingTransfer(msgId)
	} else {
		// This will get retried next audit interval.
		acct.logger.Error(logText, zap.String("msgId", msgId), zap.String("text", errText))
	}
}

// SubmitObservationsToContract is a free function to make a call to the smart contract to submit an observation request.
// If the submit fails or the result contains an error, it will return the error. If an error is returned, the caller is
// expected to use GetFailedIndexInBatch() to see which observation in the batch failed.
func SubmitObservationsToContract(
	ctx context.Context,
	logger *zap.Logger,
	guardianSigner guardiansigner.GuardianSigner,
	gsIndex uint32,
	guardianIndex uint32,
	wormchainConn AccountantWormchainConn,
	contract string,
	prefix []byte,
	msgs []*common.MessagePublication,
) (*sdktx.BroadcastTxResponse, error) {
	obs := make([]Observation, len(msgs))
	for idx, msg := range msgs {
		obs[idx] = makeObservation(msg)

		logger.Debug("in SubmitObservationsToContract, encoding observation",
			zap.String("contract", contract),
			zap.Int("idx", idx),
			zap.String("txHash", msg.TxIDString()), zap.String("encTxHash", hex.EncodeToString(obs[idx].TxHash[:])),
			zap.Stringer("timeStamp", msg.Timestamp), zap.Uint32("encTimestamp", obs[idx].Timestamp),
			zap.Uint32("nonce", msg.Nonce), zap.Uint32("encNonce", obs[idx].Nonce),
			zap.Stringer("emitterChain", msg.EmitterChain), zap.Uint16("encEmitterChain", obs[idx].EmitterChain),
			zap.Stringer("emitterAddress", msg.EmitterAddress), zap.String("encEmitterAddress", hex.EncodeToString(obs[idx].EmitterAddress[:])),
			zap.Uint64("squence", msg.Sequence), zap.Uint64("encSequence", obs[idx].Sequence),
			zap.Uint8("consistencyLevel", msg.ConsistencyLevel), zap.Uint8("encConsistencyLevel", obs[idx].ConsistencyLevel),
			zap.String("payload", hex.EncodeToString(msg.Payload)), zap.String("encPayload", hex.EncodeToString(obs[idx].Payload)),
		)
	}

	bytes, err := json.Marshal(obs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal accountant observation request: %w", err)
	}

	digest, err := vaa.MessageSigningDigest(prefix, bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign accountant Observation request: %w", err)
	}

	sigBytes, err := guardianSigner.Sign(ctx, digest.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to sign accountant Observation request: %w", err)
	}

	sig := SignatureType{Index: guardianIndex, Signature: sigBytes}

	msgData := SubmitObservationsMsg{
		Params: SubmitObservationsParams{
			Observations:     bytes,
			GuardianSetIndex: gsIndex,
			Signature:        sig,
		},
	}

	msgBytes, err := json.Marshal(msgData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal accountant observation request: %w", err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.SenderAddress(),
		Contract: contract,
		Msg:      msgBytes,
		Funds:    sdktypes.Coins{},
	}

	logger.Debug("in SubmitObservationsToContract, sending broadcast",
		zap.String("contract", contract),
		zap.String("sender", wormchainConn.SenderAddress()),
		zap.Int("numObs", len(obs)),
		zap.String("observations", string(bytes)),
		zap.Uint32("gsIndex", gsIndex), zap.Uint32("guardianIndex", guardianIndex),
	)

	start := time.Now()
	txResp, err := wormchainConn.SignAndBroadcastTx(ctx, &subMsg)
	if err != nil {
		return txResp, fmt.Errorf("failed to send broadcast: %w", err)
	}

	if txResp == nil {
		return txResp, fmt.Errorf("sent broadcast but returned txResp is nil")
	}

	if txResp.TxResponse == nil {
		return txResp, fmt.Errorf("sent broadcast but returned txResp.TxResponse is nil")
	}

	if txResp.TxResponse.RawLog == "" {
		return txResp, fmt.Errorf("sent broadcast but raw_log is not set, unable to analyze the result")
	}

	if strings.Contains(txResp.TxResponse.RawLog, "out of gas") {
		return txResp, fmt.Errorf("out of gas: %s", txResp.TxResponse.RawLog)
	}

	if strings.Contains(txResp.TxResponse.RawLog, "failed to execute message") {
		return txResp, fmt.Errorf("failed to submit observations: %s", txResp.TxResponse.RawLog)
	}

	logger.Info("done sending broadcast",
		zap.String("contract", contract),
		zap.Int("numObs", len(obs)),
		zap.Int64("gasUsed", txResp.TxResponse.GasUsed),
		zap.Stringer("elapsedTime", time.Since(start)),
	)

	logger.Debug("in SubmitObservationsToContract, done sending broadcast",
		zap.String("contract", contract),
		zap.String("resp", wormchainConn.BroadcastTxResponseToString(txResp)),
	)
	return txResp, nil
}

// GetObservationResponses is a free function that extracts the observation responses from a transaction response.
// It assumes the transaction response is valid (SubmitObservationsToContract() did not return an error).
func GetObservationResponses(txResp *sdktx.BroadcastTxResponse) (map[string]ObservationResponseStatus, error) {
	data, err := hex.DecodeString(txResp.TxResponse.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	var msg sdktypes.TxMsgData
	if unmarshalErr := msg.Unmarshal(data); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", unmarshalErr)
	}

	if len(msg.Data) == 0 {
		return nil, fmt.Errorf("data field is empty")
	}

	var execContractResp wasmdtypes.MsgExecuteContractResponse
	if unmarshalErr := execContractResp.Unmarshal(msg.Data[0].Data); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal ExecuteContractResponse: %w", unmarshalErr)
	}

	var responses ObservationResponses
	err = json.Unmarshal(execContractResp.Data, &responses)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal responses: %w", err)
	}

	out := make(map[string]ObservationResponseStatus)
	for _, resp := range responses {
		out[resp.Key.String()] = resp.Status
	}

	return out, nil
}
