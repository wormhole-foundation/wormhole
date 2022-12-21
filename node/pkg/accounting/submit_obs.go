package accounting

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
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
		// The key that uniquely identifies the Observation.
		Key TransferKey `json:"key"`

		// The nonce for the transfer.
		Nonce uint32 `json:"nonce"`

		// The serialized tokenbridge payload.
		Payload string `json:"payload"`

		// The hash of the transaction on the emitter chain in which the transfer was performed.
		TxHash string `json:"tx_hash"`
	}

	TransferKey struct {
		// The chain id of the chain on which this transfer originated.
		EmitterChain uint16 `json:"emitter_chain"`

		// The address on the emitter chain that created this transfer.
		EmitterAddress string `json:"emitter_address"`

		// The sequence number of the transfer.
		Sequence uint64 `json:"sequence"`
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
	acct.logger.Info("acct: debug: in submitObservationToContract", zap.String("msgID", msg.MessageIDString()))
	txResp, err := SubmitObservationToContract(acct.ctx, acct.logger, acct.gk, gsIndex, acct.wormchainConn, acct.contract, msg)
	if err != nil {
		acct.logger.Error("acct: failed to submit observation request", zap.String("msgId", msg.MessageIDString()), zap.Error(err))
		submitFailures.Inc()
		return
	}

	alreadyCommitted, err := CheckSubmitObservationResult(txResp)
	if err != nil {
		acct.logger.Error("acct: failed to submit observation request", zap.String("msgId", msg.MessageIDString()), zap.Error(err))
		submitFailures.Inc()
		return
	}

	if alreadyCommitted {
		acct.mutex.Lock()
		defer acct.mutex.Unlock()
		pk := pendingKey{emitterChainId: msg.EmitterChain, txHash: msg.TxHash}
		pe, exists := acct.pendingTransfers[pk]
		if exists {
			acct.logger.Info("acct: transfer has already been committed, publishing it", zap.String("msgId", msg.MessageIDString()))
			acct.publishTransfer(pe)
			transfersApproved.Inc()
		} else {
			acct.logger.Info("acct: debug: transfer has already been committed, and it is no longer in our map", zap.String("msgId", msg.MessageIDString()))
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
	obs := []Observation{
		Observation{
			Key: TransferKey{
				EmitterChain:   uint16(msg.EmitterChain),
				EmitterAddress: base64.StdEncoding.EncodeToString(msg.EmitterAddress.Bytes()),
				Sequence:       msg.Sequence,
			},
			Nonce:   msg.Nonce,
			TxHash:  strings.Trim(string(msg.TxHash.String()), `0x`),
			Payload: base64.StdEncoding.EncodeToString(msg.Payload),
		},
	}

	bytes, err := json.Marshal(obs)
	if err != nil {
		err = fmt.Errorf("acct: failed to marshal accounting observation request: %w", err)
		panic(err)
	}

	b64String := base64.StdEncoding.EncodeToString(bytes)

	digest := vaa.SigningMsg(bytes)

	SignatureBytes, err := ethCrypto.Sign(digest.Bytes(), gk)
	if err != nil {
		err = fmt.Errorf("acct: failed to sign accounting Observation request: %w", err)
		panic(err)
	}

	sig := SignatureType{Index: 0, Signature: SignatureBytes}

	msgData := SubmitObservationsMsg{
		Params: SubmitObservationsParams{
			Observations:     b64String,
			GuardianSetIndex: gsIndex,
			Signature:        sig,
		},
	}

	msgBytes, err := json.Marshal(msgData)
	if err != nil {
		err = fmt.Errorf("acct: failed to marshal accounting observation request: %w", err)
		panic(err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.PublicKey(),
		Contract: contract,
		Msg:      msgBytes,
		Funds:    sdktypes.Coins{},
	}

	logger.Info("acct: debug: in SubmitObservationToContract, sending broadcast")
	txResp, err := wormchainConn.SignAndBroadcastTx(ctx, &subMsg)
	if err != nil {
		logger.Error("acct: SubmitObservationToContract failed to send broadcast", zap.Error(err))
	} else {
		out, err := wormchainConn.BroadcastTxResponseToString(txResp)
		if err != nil {
			logger.Error("acct: SubmitObservationToContract failed to parse broadcast response", zap.Error(err))
		} else {
			logger.Info("acct: debug: in SubmitObservationToContract, done sending broadcast", zap.String("resp", out))
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
		if strings.Contains(txResp.TxResponse.RawLog, "cannot submit duplicate signatures for the same observation") {
			return false, nil
		}

		return false, fmt.Errorf(txResp.TxResponse.RawLog)
	}

	if strings.Contains(txResp.TxResponse.RawLog, "failed to execute message") {
		return false, fmt.Errorf(txResp.TxResponse.RawLog)
	}

	return false, nil
}
