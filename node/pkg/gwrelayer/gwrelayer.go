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
	"github.com/wormhole-foundation/wormhole/sdk"
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
type (
	GatewayRelayer struct {
		ctx                         context.Context
		logger                      *zap.Logger
		ibcTranslatorAddress        string
		wormchainConn               GatewayRelayerWormchainConn
		env                         common.Environment
		subChan                     chan *VaaToPublish
		tokenBridges                tokenBridgeMap
		tokenBridgeAddress          string
		ibcTranslatorPayloadAddress vaa.Address
	}

	tokenBridgeMap map[tokenBridgeKey]struct{}

	tokenBridgeKey struct {
		emitterChainId vaa.ChainID
		emitterAddr    vaa.Address
	}

	VaaToPublish struct {
		V               *vaa.VAA
		ContractAddress string
		VType           VaaType
	}

	VaaType uint8
)

const (
	IbcTranslator VaaType = iota
	TokenBridge
)

// subChanSize is the capacity of the submit channel used to publish VAAs.
const subChanSize = 50

var (
	vaasSubmittedToIbcTranslator = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_vaas_submitted_to_ibc_translator",
			Help: "Total number of VAAs submitted to the ibc translator contract by the gateway relayer",
		})

	vaasSubmittedToTokenBridge = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_vaas_submitted_to_token_bridge",
			Help: "Total number of VAAs submitted to the token bridge contract by the gateway relayer",
		})

	channelFullErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_channel_full_errors",
			Help: "Total number of VAAs dropped because the gateway relayer channel was full",
		})
	submitErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gwrelayer_submit_errors",
			Help: "Total number of errors encountered while submitting VAAs",
		})
)

// NewGatewayRelayer creates a new instance of the GatewayRelayer object.
func NewGatewayRelayer(
	ctx context.Context,
	logger *zap.Logger,
	ibcTranslatorAddress string,
	wormchainConn GatewayRelayerWormchainConn,
	env common.Environment,
) *GatewayRelayer {
	if ibcTranslatorAddress == "" {
		return nil // This is not an error, it just means the feature is not enabled.
	}

	return &GatewayRelayer{
		ctx:                  ctx,
		logger:               logger.With(zap.String("component", "gwrelayer")),
		ibcTranslatorAddress: ibcTranslatorAddress,
		wormchainConn:        wormchainConn,
		env:                  env,
		subChan:              make(chan *VaaToPublish, subChanSize),
		// tokenBridgeAddress and tokenBridges are initialized in Start().
	}
}

// Start initializes the gateway relayer and starts the worker runnable that submits VAAs to the contract.
func (gwr *GatewayRelayer) Start(ctx context.Context) error {
	var err error
	gwr.ibcTranslatorPayloadAddress, err = convertBech32AddressToWormhole(gwr.ibcTranslatorAddress)
	if err != nil {
		return err
	}

	gwr.tokenBridges, gwr.tokenBridgeAddress, err = buildTokenBridgeMap(gwr.logger, gwr.env)
	if err != nil {
		return fmt.Errorf("failed to build token bridge map: %w", err)
	}
	if gwr.tokenBridgeAddress == "" {
		return fmt.Errorf("failed to look up token bridge address for gateway")
	}

	gwr.logger.Info("starting gateway relayer",
		zap.String("ibcTranslatorAddress", gwr.ibcTranslatorAddress),
		zap.String("ibcTranslatorPayloadAddress", hex.EncodeToString(gwr.ibcTranslatorPayloadAddress.Bytes())),
		zap.String("tokenBridgeAddress", gwr.tokenBridgeAddress),
	)

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

// buildTokenBridgeMap builds a set of all token bridge emitter chain / emitter addresses.
func buildTokenBridgeMap(logger *zap.Logger, env common.Environment) (tokenBridgeMap, string, error) {
	emitterMap := sdk.KnownTokenbridgeEmitters
	if env == common.TestNet {
		emitterMap = sdk.KnownTestnetTokenbridgeEmitters
	} else if env == common.UnsafeDevNet || env == common.GoTest || env == common.AccountantMock {
		emitterMap = sdk.KnownDevnetTokenbridgeEmitters
	}

	// Build the map of token bridges to be monitored.
	tokenBridges := make(tokenBridgeMap)
	tokenBridgeAddress := ""
	for chainId, emitterAddrBytes := range emitterMap {
		if chainId == vaa.ChainIDWormchain {
			var err error
			tokenBridgeAddress, err = sdktypes.Bech32ifyAddressBytes("wormhole", emitterAddrBytes)
			if err != nil {
				return nil, "", fmt.Errorf(`failed to convert gateway emitter address "%s" to bech32: %v`, hex.EncodeToString(emitterAddrBytes), err)
			}

			// We don't want to forward stuff that originated on Gateway, so don't add it to the map.
			continue
		}
		emitterAddr, err := vaa.BytesToAddress(emitterAddrBytes)
		if err != nil {
			return nil, "", fmt.Errorf("failed to convert emitter address for chain: %v", chainId)
		}

		tbk := tokenBridgeKey{emitterChainId: chainId, emitterAddr: emitterAddr}
		_, exists := tokenBridges[tbk]
		if exists {
			return nil, "", fmt.Errorf("detected duplicate token bridge for chain: %v", chainId)
		}

		tokenBridges[tbk] = struct{}{}
		logger.Info("will monitor token bridge:", zap.Stringer("emitterChainId", tbk.emitterChainId), zap.Stringer("emitterAddr", tbk.emitterAddr))
	}

	return tokenBridges, tokenBridgeAddress, nil
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
	var v2p VaaToPublish
	if shouldPub, err := shouldPublishToIbcTranslator(v.Payload, vaa.ChainIDWormchain, gwr.ibcTranslatorPayloadAddress); err != nil {
		gwr.logger.Error("failed to check if vaa should be published", zap.String("msgId", v.MessageID()), zap.Error(err))
		return
	} else if shouldPub {
		v2p.VType = IbcTranslator
		v2p.ContractAddress = gwr.ibcTranslatorAddress
	} else if shouldPub = shouldPublishToTokenBridge(gwr.tokenBridges, v); shouldPub {
		v2p.VType = TokenBridge
		v2p.ContractAddress = gwr.tokenBridgeAddress
	} else {
		gwr.logger.Debug("not relaying vaa", zap.String("msgId", v.MessageID()))
		return
	}

	v2p.V = v

	select {
	case gwr.subChan <- &v2p:
		gwr.logger.Debug("submitted vaa to channel", zap.String("msgId", v.MessageID()), zap.String("contract", v2p.ContractAddress), zap.Uint8("vaaType", uint8(v2p.VType)))
	default:
		channelFullErrors.Inc()
		gwr.logger.Error("unable to submit vaa because the channel is full, dropping it", zap.String("msgId", v.MessageID()), zap.String("contract", v2p.ContractAddress), zap.Uint8("vaaType", uint8(v2p.VType)))
	}
}

// shouldPublishToIbcTranslator returns true if a message should be forwarded to the contract on wormchain, false if not.
func shouldPublishToIbcTranslator(payload []byte, targetChain vaa.ChainID, targetAddress vaa.Address) (bool, error) {
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

// shouldPublishToTokenBridge returns true if a message should be forwarded to the token bridge, false if not.
func shouldPublishToTokenBridge(tokenBridges tokenBridgeMap, v *vaa.VAA) bool {
	if _, exists := tokenBridges[tokenBridgeKey{emitterChainId: v.EmitterChain, emitterAddr: v.EmitterAddress}]; !exists {
		return false
	}

	if len(v.Payload) == 0 {
		return false
	}

	// We only forward attestations (type two).
	if v.Payload[0] != 2 {
		return false
	}

	return true
}

// worker listens for VAAs and submits them to the smart contract.
func (gwr *GatewayRelayer) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case v2p := <-gwr.subChan:
			if err := gwr.submitVAAToContract(v2p); err != nil {
				gwr.logger.Error("failed to submit vaa to contract", zap.String("msgId", v2p.V.MessageID()), zap.String("contract", v2p.ContractAddress), zap.Uint8("vaaType", uint8(v2p.VType)), zap.Error(err))
				// TODO: For now we don't want to restart because this will happen if the VAA has already been submitted by another guardian.
				//return fmt.Errorf("failed to submit vaa to contract: %w", err)
			}
		}
	}
}

// submitVAAToContract submits a VAA to the smart contract on wormchain.
func (gwr *GatewayRelayer) submitVAAToContract(v2p *VaaToPublish) error {
	_, err := SubmitVAAToContract(gwr.ctx, gwr.logger, gwr.wormchainConn, v2p)
	if err != nil {
		submitErrors.Inc()
		return err
	}

	if v2p.VType == IbcTranslator {
		vaasSubmittedToIbcTranslator.Inc()
	} else {
		vaasSubmittedToTokenBridge.Inc()
	}
	return nil
}

type (
	// completeTransferAndConvertMsg is used to submit a VAA to the IBC translator contract.
	completeTransferAndConvertMsg struct {
		Params completeTransferAndConvertParams `json:"complete_transfer_and_convert"`
	}

	completeTransferAndConvertParams struct {
		VAA []byte `json:"vaa"`
	}

	// submitVAA is used to submit a VAA to the token bridge contract.
	submitVAA struct {
		Params submitVAAParams `json:"submit_vaa"`
	}

	submitVAAParams struct {
		Data []byte `json:"data"`
	}
)

// SubmitVAAToContract submits a VAA to the smart contract on wormchain.
func SubmitVAAToContract(
	ctx context.Context,
	logger *zap.Logger,
	wormchainConn GatewayRelayerWormchainConn,
	v2p *VaaToPublish,
) (*sdktx.BroadcastTxResponse, error) {
	logger.Info("submitting VAA to contract", zap.String("message_id", v2p.V.MessageID()), zap.String("contract", v2p.ContractAddress), zap.Uint8("vaaType", uint8(v2p.VType)))

	vaaBytes, err := v2p.V.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vaa: %w", err)
	}

	var msgBytes []byte

	if v2p.VType == IbcTranslator {
		msgData := completeTransferAndConvertMsg{
			Params: completeTransferAndConvertParams{
				VAA: vaaBytes,
			},
		}
		msgBytes, err = json.Marshal(msgData)
	} else if v2p.VType == TokenBridge {
		msgData := submitVAA{
			Params: submitVAAParams{
				Data: vaaBytes,
			},
		}
		msgBytes, err = json.Marshal(msgData)
	} else {
		return nil, fmt.Errorf("invalid vtype: %d", uint8(v2p.VType))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	subMsg := wasmdtypes.MsgExecuteContract{
		Sender:   wormchainConn.SenderAddress(),
		Contract: v2p.ContractAddress,
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

	if strings.Contains(txResp.TxResponse.RawLog, "failed") && !strings.Contains(txResp.TxResponse.RawLog, "VaaAlreadyExecuted") {
		return txResp, fmt.Errorf("submit failed: %s", txResp.TxResponse.RawLog)
	}

	logger.Info("done sending broadcast",
		zap.String("msgId", v2p.V.MessageID()),
		zap.String("contract", v2p.ContractAddress),
		zap.Uint8("vaaType", uint8(v2p.VType)),
		zap.Int64("gasUsed", txResp.TxResponse.GasUsed),
		zap.Stringer("elapsedTime", time.Since(start)),
		zap.String("txHash", txResp.TxResponse.TxHash),
	)

	logger.Debug("in SubmitVAAToContract, done sending broadcast", zap.String("resp", wormchainConn.BroadcastTxResponseToString(txResp)))

	return txResp, nil
}
