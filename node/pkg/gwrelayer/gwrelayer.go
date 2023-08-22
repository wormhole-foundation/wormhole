// The GatewayRelayer manages the interface to the ibcTranslator smart contract on wormchain. It is called when a signed VAA with quorum gets published.
// It forwards all payload three VAAs destined for the ibcTranslator contract on wormchain to that contract.

package gwrelayer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"go.uber.org/zap"
)

type (
	GatewayRelayerWormchainConn interface {
		Close()
		SenderAddress() string
		SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error)
		SignAndBroadcastTx(ctx context.Context, msg sdktypes.Msg) (*sdktx.BroadcastTxResponse, error)
		BroadcastTxResponseToString(txResp *sdktx.BroadcastTxResponse) string
	}
)

// GatewayRelayer is the object that manages the interface to the wormchain accountant smart contract.
type GatewayRelayer struct {
	ctx             context.Context
	logger          *zap.Logger
	contractAddress string
	wormchainConn   GatewayRelayerWormchainConn
	env             common.Environment
	subChan         chan *vaa.VAA
	targetAddress   vaa.Address
}

// subChanSize is the capacity of the submit channel used to publish VAAs.
const subChanSize = 50

var (
	vaasSubmitted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_vaas_submitted",
			Help: "Total number of VAAs submitted to the gateway relayer",
		})

	channelFullErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_channel_full_errors",
			Help: "Total number of VAAs dropped because the gateway relayer channel was full",
		})
	submitErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_submit_errors",
			Help: "Total number of errors encountered while submitting VAAs to the gateway relayer",
		})
)

// NewGatewayRelayer creates a new instance of the GatewayRelayer object.
func NewGatewayRelayer(
	ctx context.Context,
	logger *zap.Logger,
	contractAddress string,
	wormchainConn GatewayRelayerWormchainConn,
	env common.Environment,
) *GatewayRelayer {
	if contractAddress == "" {
		return nil // This is not an error, it just means the feature is not enabled.
	}
	return &GatewayRelayer{
		ctx:             ctx,
		logger:          logger.With(zap.String("component", "gwrelayer")),
		contractAddress: contractAddress,
		wormchainConn:   wormchainConn,
		env:             env,
		subChan:         make(chan *vaa.VAA, subChanSize),
	}
}

// Start initializes the gateway relayer and starts the worker runnable that submits VAAs to the contract.
func (gwr *GatewayRelayer) Start(ctx context.Context) error {
	var err error
	gwr.targetAddress, err = convertBech32AddressToWormhole(gwr.contractAddress)
	if err != nil {
		return err
	}

	gwr.logger.Info("starting gateway relayer", zap.String("contract", gwr.contractAddress), zap.String("targetAddress", hex.EncodeToString(gwr.targetAddress.Bytes())))

	// Start the watcher to listen to transfer events from the smart contract.
	if gwr.env == common.GoTest {
		// We're not in a runnable context, so we can't use supervisor.
		go func() {
			_ = gwr.worker(ctx)
		}()
	} else {
		if err := supervisor.Run(ctx, "gwrworker", common.WrapWithScissors(gwr.worker, "gwrworker")); err != nil {
			return fmt.Errorf("failed to start submit vaa worker: %w", err)
		}
	}

	return nil
}

// Close closes the connection to the smart contract.
func (gwr *GatewayRelayer) Close() {
	if gwr.wormchainConn != nil {
		gwr.wormchainConn.Close()
		gwr.wormchainConn = nil
	}
}

// convertBech32AddressToWormhole converts a bech32 address to a wormhole address.
func convertBech32AddressToWormhole(contractAddress string) (vaa.Address, error) {
	addrBytes, err := sdktypes.GetFromBech32(contractAddress, "wormhole")
	if err != nil {
		return vaa.Address{}, fmt.Errorf(`failed to decode target address "%s": %w`, contractAddress, err)
	}
	return vaa.Address(addrBytes), nil
}

// SubmitVAA checks to see if the VAA should be submitted to the smart contract, and if so, writes it to the channel for publishing.
func (gwr *GatewayRelayer) SubmitVAA(v *vaa.VAA) {
	if shouldPub, err := shouldPublish(v.Payload, vaa.ChainIDWormchain, gwr.targetAddress); err != nil {
		gwr.logger.Error("failed to check if vaa should be published", zap.String("msgId", v.MessageID()), zap.Error(err))
		return
	} else if !shouldPub {
		return
	}

	select {
	case gwr.subChan <- v:
		gwr.logger.Debug("submitted vaa to channel", zap.String("msgId", v.MessageID()))
	default:
		channelFullErrors.Inc()
		gwr.logger.Error("unable to submit vaa because the channel is full, dropping it", zap.String("msgId", v.MessageID()))
	}
}

// shouldPublish returns true if a message should be forwarded to the contract on wormchain, false if not.
func shouldPublish(payload []byte, targetChain vaa.ChainID, targetAddress vaa.Address) (bool, error) {
	if len(payload) == 0 {
		return false, nil
	}

	if payload[0] != 3 {
		return false, nil
	}

	hdr, err := vaa.DecodeTransferPayloadHdr(payload)
	if err != nil {
		return false, fmt.Errorf("failed to decode payload: %w", err)
	}

	if hdr.TargetChain != targetChain || hdr.TargetAddress != targetAddress {
		return false, nil
	}

	return true, nil
}

// worker listens for VAAs and submits them to the smart contract.
func (gwr *GatewayRelayer) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case v := <-gwr.subChan:
			if err := gwr.submitVAAToContract(v); err != nil {
				gwr.logger.Error("failed to submit vaa to contract", zap.String("msgId", v.MessageID()), zap.Error(err))
				// TODO: For now we don't want to restart because this will happen if the VAA has already been submitted by another guardian.
				//return fmt.Errorf("failed to submit vaa to contract: %w", err)
			}
		}
	}
}

// submitVAAToContract submits a VAA to the smart contract on wormchain.
func (gwr *GatewayRelayer) submitVAAToContract(v *vaa.VAA) error {
	_, err := SubmitVAAToContract(gwr.ctx, gwr.logger, gwr.wormchainConn, gwr.contractAddress, v)
	if err != nil {
		submitErrors.Inc()
		return err
	}
	// TODO: Need to check txResp for "VAA already submitted", which should not be an error.
	vaasSubmitted.Inc()
	return nil
}

type (
	completeTransferAndConvertMsg struct {
		Params completeTransferAndConvertParams `json:"complete_transfer_and_convert"`
	}

	completeTransferAndConvertParams struct {
		VAA []byte `json:"vaa"`
	}
)

// SubmitVAAToContract submits a VAA to the smart contract on wormchain.
func SubmitVAAToContract(
	ctx context.Context,
	logger *zap.Logger,
	wormchainConn GatewayRelayerWormchainConn,
	contract string,
	v *vaa.VAA,
) (*sdktx.BroadcastTxResponse, error) {
	logger.Info("submitting VAA to contract", zap.String("message_id", v.MessageID()))

	vaaBytes, err := v.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vaa: %w", err)
	}

	msgData := completeTransferAndConvertMsg{
		Params: completeTransferAndConvertParams{
			VAA: vaaBytes,
		},
	}

	msgBytes, err := json.Marshal(msgData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.SenderAddress(),
		Contract: contract,
		Msg:      msgBytes,
		Funds:    sdktypes.Coins{},
	}

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

	if strings.Contains(txResp.TxResponse.RawLog, "failed") {
		return txResp, fmt.Errorf("submit failed: %s", txResp.TxResponse.RawLog)
	}

	logger.Info("done sending broadcast", zap.String("msgId", v.MessageID()), zap.Int64("gasUsed", txResp.TxResponse.GasUsed), zap.Stringer("elapsedTime", time.Since(start)))
	logger.Debug("in SubmitVAAToContract, done sending broadcast", zap.String("resp", wormchainConn.BroadcastTxResponseToString(txResp)))

	return txResp, nil
}
