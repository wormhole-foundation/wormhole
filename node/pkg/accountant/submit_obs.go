package accountant

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"go.uber.org/zap"
)

const batchSize = 10 // TODO: Arbitrary limit. What makes sense?

func (acct *Accountant) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-acct.subChan:
			// TODO: Is there a better way to read at most X items from a channel without blocking?

			// Sleep for a bit until either there are no more new observations or the batch is full.
			numToRead := len(acct.subChan)
			for numToRead < batchSize-1 {
				time.Sleep(100 * time.Millisecond) // TODO: Arbitrary delay. What makes sense?
				newNumToRead := len(acct.subChan)
				if newNumToRead == numToRead {
					break
				}

				numToRead = newNumToRead
			}

			// Read up to the batch size.
			if numToRead >= batchSize {
				numToRead = batchSize - 1
			}

			msgs := make([]*common.MessagePublication, numToRead+1)
			msgs[0] = msg
			acct.logger.Debug("acct: submitting message to contract", zap.String("msgID", msg.MessageIDString()))

			for i := 0; i < numToRead; i++ {
				msgs[i+1] = <-acct.subChan
				acct.logger.Debug("acct: submitting message to contract", zap.String("msgID", msg.MessageIDString()))
			}

			gs := acct.gst.Get()
			if gs == nil {
				acct.logger.Error("acct: unable to send observation request: failed to look up guardian set", zap.String("msgID", msg.MessageIDString()))
				continue
			}

			guardianIndex, found := gs.KeyIndex(acct.guardianAddr)
			if !found {
				acct.logger.Error("acct: unable to send observation request: failed to look up guardian index",
					zap.String("msgID", msg.MessageIDString()), zap.Stringer("guardianAddr", acct.guardianAddr))
				continue
			}

			acct.submitObservationToContract(msgs, gs.Index, uint32(guardianIndex))
			transfersSubmitted.Add(float64(len(msgs)))
		}
	}
}

type (
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
		Key    ObservationKey
		Status ObservationResponseStatus
	}

	ObservationKey struct {
		EmitterChain   uint16      `json:"emitter_chain"`
		EmitterAddress vaa.Address `json:"emitter_address"`
		Sequence       uint64      `json:"sequence"`
	}

	ObservationResponseStatus struct {
		Type string `json:"type"`
		Data string `json:"data"`
	}
)

func (k ObservationKey) String() string {
	return fmt.Sprintf("%v/%v/%v", k.EmitterChain, hex.EncodeToString(k.EmitterAddress[:]), k.Sequence)
}

func (sb SignatureBytes) MarshalJSON() ([]byte, error) {
	var result string
	if sb == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", sb)), ",")
	}
	return []byte(result), nil
}

// submitObservationToContract makes a call to the smart contract to submit a batch of observation requests.
// It should be called from a go routine because it can block.
func (acct *Accountant) submitObservationToContract(msgs []*common.MessagePublication, gsIndex uint32, guardianIndex uint32) {
	txResp, err := SubmitObservationsToContract(acct.ctx, acct.logger, acct.gk, gsIndex, guardianIndex, acct.wormchainConn, acct.contract, msgs)
	if err != nil {
		// This means the whole batch failed. They will all get retried the next audit cycle.
		acct.logger.Error("acct: failed to submit any observations in batch", zap.Int("numMsgs", len(msgs)), zap.Error(err))
		for idx, msg := range msgs {
			acct.logger.Error("acct: failed to submit observation", zap.Int("idx", idx), zap.String("msgId", msg.MessageIDString()))
		}

		submitFailures.Add(float64(len(msgs)))
		return
	}

	responses, err := GetObservationResponses(txResp, len(msgs))
	if err != nil {
		// This means the whole batch failed. They will all get retried the next audit cycle.
		acct.logger.Error("acct: failed to get responses from batch", zap.Int("numMsgs", len(msgs)), zap.Error(err))
		for idx, msg := range msgs {
			acct.logger.Error("acct: need to retry observation", zap.Int("idx", idx), zap.String("msgId", msg.MessageIDString()))
		}

		submitFailures.Add(float64(len(msgs)))
		return
	}

	for idx, resp := range responses {
		// Verify that the responses are in the same order as the observations.
		msgId := msgs[idx].MessageIDString()
		if resp.Key.String() != msgId {
			// This will get retried next audit interval.
			acct.logger.Error("acct: unexpected msgId in observation response", zap.Int("idx", idx), zap.String("expected", msgId), zap.String("actual", resp.Key.String()))
			submitFailures.Inc()
			continue
		}

		switch resp.Status.Type {
		case "pending":
			acct.logger.Info("acct: transfer is pending", zap.String("msgId", msgId))
		case "committed":
			acct.pendingTransfersLock.Lock()
			defer acct.pendingTransfersLock.Unlock()
			pe, exists := acct.pendingTransfers[msgId]
			if exists {
				acct.logger.Info("acct: transfer has already been committed, publishing it", zap.String("msgId", msgId))
				acct.publishTransfer(pe)
				transfersApproved.Inc()
			} else {
				acct.logger.Debug("acct: transfer has already been committed but it is no longer in our map", zap.String("msgId", msgId))
			}
		case "error":
			submitFailures.Inc()
			if strings.Contains(resp.Status.Data, "insufficient balance") {
				balanceErrors.Inc()
				acct.logger.Error("acct: insufficient balance error detected, dropping transfer", zap.String("msgId", msgId), zap.String("text", resp.Status.Data))
				acct.pendingTransfersLock.Lock()
				defer acct.pendingTransfersLock.Unlock()
				acct.deletePendingTransfer(msgId)
			} else {
				// This will get retried next audit interval.
				acct.logger.Error("acct: failed to submit observation", zap.String("msgId", msgId), zap.String("text", resp.Status.Data))
			}
		default:
			// This will get retried next audit interval.
			acct.logger.Error("acct: unexpected status response on observation", zap.String("msgId", msgId), zap.String("status", resp.Status.Type), zap.String("text", resp.Status.Data))
			submitFailures.Inc()
		}
	}
}

// SubmitObservationsToContract is a free function to make a call to the smart contract to submit an observation request.
// If the submit fails or the result contains an error, it will return the error. If an error is returned, the caller is
// expected to use GetFailedIndexInBatch() to see which observation in the batch failed.
func SubmitObservationsToContract(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	gsIndex uint32,
	guardianIndex uint32,
	wormchainConn *wormconn.ClientConn,
	contract string,
	msgs []*common.MessagePublication,
) (*sdktx.BroadcastTxResponse, error) {
	obs := make([]Observation, len(msgs))
	for idx, msg := range msgs {
		obs[idx] = Observation{
			TxHash:           msg.TxHash.Bytes(),
			Timestamp:        uint32(msg.Timestamp.Unix()),
			Nonce:            msg.Nonce,
			EmitterChain:     uint16(msg.EmitterChain),
			EmitterAddress:   msg.EmitterAddress,
			Sequence:         msg.Sequence,
			ConsistencyLevel: msg.ConsistencyLevel,
			Payload:          msg.Payload,
		}

		logger.Debug("acct: in SubmitObservationsToContract, encoding observation",
			zap.Int("idx", idx),
			zap.String("txHash", msg.TxHash.String()), zap.String("encTxHash", hex.EncodeToString(obs[idx].TxHash[:])),
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
		return nil, fmt.Errorf("acct: failed to marshal accountant observation request: %w", err)
	}

	digest := vaa.SigningMsg(bytes)

	sigBytes, err := ethCrypto.Sign(digest.Bytes(), gk)
	if err != nil {
		return nil, fmt.Errorf("acct: failed to sign accountant Observation request: %w", err)
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
		return nil, fmt.Errorf("acct: failed to marshal accountant observation request: %w", err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.SenderAddress(),
		Contract: contract,
		Msg:      msgBytes,
		Funds:    sdktypes.Coins{},
	}

	logger.Debug("acct: in SubmitObservationsToContract, sending broadcast",
		zap.Int("numObs", len(obs)),
		zap.String("observations", string(bytes)),
		zap.Uint32("gsIndex", gsIndex), zap.Uint32("guardianIndex", guardianIndex),
	)

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

	logger.Debug("acct: in SubmitObservationsToContract, done sending broadcast", zap.String("resp", wormchainConn.BroadcastTxResponseToString(txResp)))
	return txResp, nil
}

// GetObservationResponses is a free function that extracts the observation responses from a transaction response.
// It assumes the transaction response is valid (SubmitObservationsToContract() did not return an error).
func GetObservationResponses(txResp *sdktx.BroadcastTxResponse, numExpected int) (ObservationResponses, error) {
	data, err := hex.DecodeString(txResp.TxResponse.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	var msg sdktypes.TxMsgData
	if err := msg.Unmarshal([]byte(data)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	var execContractResp wasmdtypes.MsgExecuteContractResponse
	if err := execContractResp.Unmarshal(msg.Data[0].Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ExecuteContractResponse: %w", err)
	}

	var responses ObservationResponses
	err = json.Unmarshal(execContractResp.Data, &responses)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal responses: %w", err)
	}

	if len(responses) != numExpected {
		return nil, fmt.Errorf("unexpected number of responses, expected %d, actual %d", numExpected, len(responses))
	}

	return responses, nil
}
