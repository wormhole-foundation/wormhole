package accounting

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/wormconn"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"go.uber.org/zap"
)

func (acct *Accounting) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-acct.subChan:
			gs := acct.gst.Get()
			if gs == nil {
				acct.logger.Error("acct: unable to send observation request: failed to look up guardian set", zap.String("msgID", msg.MessageIDString()))
				continue
			}

			go acct.submitObservationToContract(msg, gs.Index)
			transfersSubmitted.Inc()
		}
	}
}

type (
	SubmitObservationsMsg struct {
		Params SubmitObservationsParams `json:"submit_observations"`
	}

	SubmitObservationsParams struct {
		// A serialized `Vec<Observation>`. Multiple observations can be submitted together to reduce  transaction overhead.
		Observations string `json:"observations"`

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
		TxHash string `json:"tx_hash"`

		// Seconds since UNIX epoch.
		Timestamp uint32 `json:"timestamp"`

		// The nonce for the transfer.
		Nonce uint32 `json:"nonce"`

		// The source chain from which this observation was created.
		EmitterChain uint16 `json:"emitter_chain"`

		// The address on the source chain that emitted this message.
		EmitterAddress [32]byte `json:"emitter_address"`

		// The sequence number of this observation.
		Sequence uint64 `json:"sequence"`

		// The consistency level requested by the emitter.
		ConsistencyLevel uint8 `json:"consistency_level"`

		// The serialized tokenbridge payload.
		Payload string `json:"payload"`
	}
)

func (sb SignatureBytes) MarshalJSON() ([]byte, error) {
	var result string
	if sb == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", sb)), ",")
	}
	return []byte(result), nil
}

// submitObservationToContract makes a call to the smart contract to submit an observation request.
// It should be called from a go routine because it can block.
func (acct *Accounting) submitObservationToContract(msg *common.MessagePublication, gsIndex uint32) {
	msgId := msg.MessageIDString()
	acct.logger.Debug("acct: in submitObservationToContract", zap.String("msgID", msgId))
	txResp, err := SubmitObservationToContract(acct.ctx, acct.logger, acct.gk, gsIndex, acct.wormchainConn, acct.contract, msg)
	if err != nil {
		acct.logger.Error("acct: failed to submit observation request", zap.String("msgId", msgId), zap.Error(err))
		submitFailures.Inc()
		return
	}

	alreadyCommitted, err := CheckSubmitObservationResult(txResp)
	if err != nil {
		submitFailures.Inc()
		if strings.Contains(err.Error(), "insufficient balance") {
			balanceErrors.Inc()
			acct.logger.Error("acct: insufficient balance error detected, dropping transfer", zap.String("msgId", msgId), zap.Error(err))
			acct.pendingTransfersLock.Lock()
			defer acct.pendingTransfersLock.Unlock()
			acct.deletePendingTransfer(msgId)
		} else {
			acct.logger.Error("acct: failed to submit observation request", zap.String("msgId", msgId), zap.Error(err))
		}
		return
	}

	if alreadyCommitted {
		acct.pendingTransfersLock.Lock()
		defer acct.pendingTransfersLock.Unlock()
		pe, exists := acct.pendingTransfers[msgId]
		if exists {
			acct.logger.Info("acct: transfer has already been committed, publishing it", zap.String("msgId", msgId))
			acct.publishTransfer(pe)
			transfersApproved.Inc()
		} else {
			acct.logger.Debug("acct: transfer has already been committed, and it is no longer in our map", zap.String("msgId", msgId))
		}
	}
}

// SubmitObservationToContract is a free function to make a call to the smart contract to submit an observation request.
func SubmitObservationToContract(
	ctx context.Context,
	logger *zap.Logger,
	gk *ecdsa.PrivateKey,
	gsIndex uint32,
	wormchainConn *wormconn.ClientConn,
	contract string,
	msg *common.MessagePublication,
) (*sdktx.BroadcastTxResponse, error) {
	txHashStr := msg.TxHash.String()
	if strings.Index(txHashStr, "0x") == 0 {
		txHashStr = txHashStr[2:]
	}
	txHashBytes, err := hex.DecodeString(txHashStr)
	if err != nil {
		return nil, fmt.Errorf("acct: failed to decode tx_hash '%s': %w", msg.TxHash.String(), err)
	}
	obs := []Observation{
		Observation{
			TxHash:           base64.StdEncoding.EncodeToString(txHashBytes),
			Timestamp:        uint32(msg.Timestamp.Unix()),
			Nonce:            msg.Nonce,
			EmitterChain:     uint16(msg.EmitterChain),
			EmitterAddress:   msg.EmitterAddress,
			Sequence:         msg.Sequence,
			ConsistencyLevel: msg.ConsistencyLevel,
			Payload:          base64.StdEncoding.EncodeToString(msg.Payload),
		},
	}

	bytes, err := json.Marshal(obs)
	if err != nil {
		return nil, fmt.Errorf("acct: failed to marshal accounting observation request: %w", err)
	}

	b64String := base64.StdEncoding.EncodeToString(bytes)

	digest := vaa.SigningMsg(bytes)

	sigBytes, err := ethCrypto.Sign(digest.Bytes(), gk)
	if err != nil {
		return nil, fmt.Errorf("acct: failed to sign accounting Observation request: %w", err)
	}

	sig := SignatureType{Index: 0, Signature: sigBytes}

	msgData := SubmitObservationsMsg{
		Params: SubmitObservationsParams{
			Observations:     b64String,
			GuardianSetIndex: gsIndex,
			Signature:        sig,
		},
	}

	msgBytes, err := json.Marshal(msgData)
	if err != nil {
		return nil, fmt.Errorf("acct: failed to marshal accounting observation request: %w", err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.PublicKey(),
		Contract: contract,
		Msg:      msgBytes,
		Funds:    sdktypes.Coins{},
	}

	logger.Debug("acct: in SubmitObservationToContract, sending broadcast",
		zap.String("txHash", txHashStr), zap.String("encTxHash", obs[0].TxHash),
		zap.Stringer("timeStamp", msg.Timestamp), zap.Uint32("encTimestamp", obs[0].Timestamp),
		zap.Uint32("nonce", msg.Nonce), zap.Uint32("encNonce", obs[0].Nonce),
		zap.Stringer("emitterChain", msg.EmitterChain), zap.Uint16("encEmitterChain", obs[0].EmitterChain),
		zap.Stringer("emitterAddress", msg.EmitterAddress), zap.String("encEmitterAddress", hex.EncodeToString(obs[0].EmitterAddress[:])),
		zap.Uint64("squence", msg.Sequence), zap.Uint64("encSequence", obs[0].Sequence),
		zap.Uint8("consistencyLevel", msg.ConsistencyLevel), zap.Uint8("encConsistencyLevel", obs[0].ConsistencyLevel),
		zap.String("payload", hex.EncodeToString(msg.Payload)), zap.String("encPayload", obs[0].Payload),
		zap.String("b64String", b64String),
	)

	txResp, err := wormchainConn.SignAndBroadcastTx(ctx, &subMsg)
	if err != nil {
		logger.Error("acct: SubmitObservationToContract failed to send broadcast", zap.Error(err))
	} else {
		if txResp.TxResponse == nil {
			return txResp, fmt.Errorf("txResp.TxResponse is nil")
		}
		if strings.Contains(txResp.TxResponse.RawLog, "out of gas") {
			return txResp, fmt.Errorf("out of gas: %s", txResp.TxResponse.RawLog)
		}

		out, err := wormchainConn.BroadcastTxResponseToString(txResp)
		if err != nil {
			logger.Error("acct: SubmitObservationToContract failed to parse broadcast response", zap.Error(err))
		} else {
			logger.Debug("acct: in SubmitObservationToContract, done sending broadcast", zap.String("resp", out))
		}
	}
	return txResp, err
}

// CheckSubmitObservationResult() is a free function that returns true if the observation has already been committed
// or false if we need to wait for the commit. An error is returned when appropriate.
func CheckSubmitObservationResult(txResp *sdktx.BroadcastTxResponse) (bool, error) {
	if txResp == nil {
		return false, fmt.Errorf("txResp is nil")
	}
	if txResp.TxResponse == nil {
		return false, fmt.Errorf("txResp does not contain a TxResponse")
	}
	if txResp.TxResponse.RawLog == "" {
		return false, fmt.Errorf("RawLog is not set")
	}
	if strings.Contains(txResp.TxResponse.RawLog, "execute wasm contract failed") {
		if strings.Contains(txResp.TxResponse.RawLog, "already committed") {
			return true, nil

		}

		// TODO Need to test this, requires multiple guardians.
		if strings.Contains(txResp.TxResponse.RawLog, "duplicate signature") {
			return false, nil
		}

		return false, fmt.Errorf(txResp.TxResponse.RawLog)
	}

	if strings.Contains(txResp.TxResponse.RawLog, "failed to execute message") {
		return false, fmt.Errorf(txResp.TxResponse.RawLog)
	}

	return false, nil
}
