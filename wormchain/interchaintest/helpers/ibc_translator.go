package helpers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func SubmitAllowlistInstantiateContract(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName string,
	cfg ibc.ChainConfig,
	contractBech32Addr string,
	codeIdStr string,
	guardians *guardians.ValSet,
) {
	node := chain.FullNodes[0]
	codeId, err := strconv.ParseUint(codeIdStr, 10, 64)
	require.NoError(t, err)

	contractAddr := [32]byte{}
	copy(contractAddr[:], MustAccAddressFromBech32(contractBech32Addr, cfg.Bech32Prefix).Bytes())
	payload := vaa.BodyWormchainWasmAllowlistInstantiate{
		ContractAddr: contractAddr,
		CodeId:       codeId,
	}
	payloadBz, err := payload.Serialize(vaa.ActionAddWasmInstantiateAllowlist)
	require.NoError(t, err)
	v := generateVaa(0, guardians, vaa.GovernanceChain, vaa.GovernanceEmitter, payloadBz)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	// add-wasm-instantiate-allowlist [bech32 contract addr] [codeId] [vaa-hex]
	_, err = node.ExecTx(ctx, keyName, "wormhole", "add-wasm-instantiate-allowlist", contractBech32Addr, codeIdStr, vHex, "--gas", "auto")
	require.NoError(t, err)
}

type IbcTranslatorInstantiateMsg struct {
	TokenBridgeContract string `json:"token_bridge_contract"`
}

func IbcTranslatorContractInstantiateMsg(t *testing.T, tbContract string) string {
	msg := IbcTranslatorInstantiateMsg{
		TokenBridgeContract: tbContract,
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

type IbcTranslatorSubmitUpdateChainToChannelMap struct {
	SubmitUpdateChainToChannelMap SubmitUpdateChainToChannelMap `json:"submit_update_chain_to_channel_map"`
}

type SubmitUpdateChainToChannelMap struct {
	Vaa []byte `json:"vaa"`
}

func SubmitUpdateChainToChannelMapMsg(t *testing.T, allowlistChainID uint16, allowlistChannel string, guardians *guardians.ValSet) string {
	payload := new(bytes.Buffer)
	module, err := vaa.LeftPadBytes("IbcTranslator", 32)
	require.NoError(t, err)
	payload.Write(module.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, uint8(1))
	vaa.MustWrite(payload, binary.BigEndian, uint16(0))
	channelPadded, err := vaa.LeftPadBytes(allowlistChannel, 64)
	require.NoError(t, err)
	payload.Write(channelPadded.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, allowlistChainID)

	v := generateVaa(0, guardians, vaa.GovernanceChain, vaa.GovernanceEmitter, payload.Bytes())
	vBz, err := v.Marshal()
	require.NoError(t, err)

	msg := IbcTranslatorSubmitUpdateChainToChannelMap{
		SubmitUpdateChainToChannelMap: SubmitUpdateChainToChannelMap{
			Vaa: vBz,
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

type IbcTranslatorCompleteTransferAndConvert struct {
	CompleteTransferAndConvert CompleteTransferAndConvert `json:"complete_transfer_and_convert"`
}

type CompleteTransferAndConvert struct {
	Vaa []byte `json:"vaa"`
}

// TODO: replace amount's uint64 with big int or equivalent
func CreatePayload1(amount uint64, tokenAddr string, tokenChain uint16, recipient []byte, recipientChain uint16, fee uint64) []byte {
	payload := new(bytes.Buffer)
	vaa.MustWrite(payload, binary.BigEndian, uint8(1)) // Payload 1: Transfer
	payload.Write(make([]byte, 24))
	vaa.MustWrite(payload, binary.BigEndian, amount)

	tokenAddrPadded, err := vaa.LeftPadBytes(tokenAddr, 32)
	if err != nil {
		panic(err)
	}
	payload.Write(tokenAddrPadded.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, tokenChain)

	payload.Write(recipient)
	vaa.MustWrite(payload, binary.BigEndian, recipientChain)

	payload.Write(make([]byte, 24))
	vaa.MustWrite(payload, binary.BigEndian, fee)

	return payload.Bytes()
}

// TODO: replace amount's uint64 with big int or equivalent
func CreatePayload3(cfg ibc.ChainConfig, amount uint64, tokenAddr string, tokenChain uint16, recipient string, recipientChain uint16, from []byte, contractPayload []byte) []byte {
	payload := new(bytes.Buffer)
	vaa.MustWrite(payload, binary.BigEndian, uint8(3)) // Payload 3: TransferWithPayload
	payload.Write(make([]byte, 24))
	vaa.MustWrite(payload, binary.BigEndian, amount)

	tokenAddrPadded, err := vaa.LeftPadBytes(tokenAddr, 32)
	if err != nil {
		panic(err)
	}
	payload.Write(tokenAddrPadded.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, tokenChain)

	recipientAddr := MustAccAddressFromBech32(recipient, cfg.Bech32Prefix)
	payload.Write(recipientAddr.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, recipientChain)

	payload.Write(from)

	payload.Write(contractPayload)

	return payload.Bytes()
}

func IbcTranslatorCompleteTransferAndConvertMsg(t *testing.T, emitterChainID uint16, emitterAddr string, payload []byte, guardians *guardians.ValSet) string {
	emitterBz := [32]byte{}
	eIndex := 32
	for i := len(emitterAddr); i > 0; i-- {
		emitterBz[eIndex-1] = emitterAddr[i-1]
		eIndex--
	}
	v := generateVaa(0, guardians, vaa.ChainID(emitterChainID), vaa.Address(emitterBz), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	msg := IbcTranslatorCompleteTransferAndConvert{
		CompleteTransferAndConvert: CompleteTransferAndConvert{
			Vaa: vBz,
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

type GatewayIbcTokenBridgePayloadTransfer struct {
	GatewayTransfer GatewayTransfer `json:"gateway_transfer"`
}

type GatewayTransfer struct {
	Chain     uint16 `json:"chain"`
	Recipient []byte `json:"recipient"`
	Fee       string `json:"fee"`
	Nonce     uint32 `json:"nonce"`
}

func CreateGatewayIbcTokenBridgePayloadTransfer(t *testing.T, chainID uint16, recipient string, fee uint64, nonce uint32) []byte {
	msg := GatewayIbcTokenBridgePayloadTransfer{
		GatewayTransfer: GatewayTransfer{
			Chain:     chainID,
			Recipient: []byte(recipient),
			Fee:       fmt.Sprint(fee),
			Nonce:     nonce,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

type GatewayIbcTokenBridgePayloadTransferWithPayload struct {
	GatewayTransferWithPayload GatewayTransferWithPayload `json:"gateway_transfer_with_payload"`
}

type GatewayTransferWithPayload struct {
	Chain    uint16 `json:"chain"`
	Contract []byte `json:"contract"`
	Payload  []byte `json:"payload"`
	Nonce    uint32 `json:"nonce"`
}

func CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t *testing.T, chainID uint16, contract string, payload []byte, nonce uint32) []byte {
	msg := GatewayIbcTokenBridgePayloadTransferWithPayload{
		GatewayTransferWithPayload: GatewayTransferWithPayload{
			Chain:    chainID,
			Contract: []byte(contract),
			Payload:  payload,
			Nonce:    nonce,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

type IbcTranslatorQueryMsg struct {
	IbcChannel QueryIbcChannel `json:"ibc_channel"`
}

type QueryIbcChannel struct {
	ChainID uint16 `json:"chain_id"`
}

type IbcTranslatorQueryRspMsg struct {
	Data *IbcTranslatorQueryRspObj `json:"data"`
}

type IbcTranslatorQueryRspObj struct {
	Channel string `json:"channel,omitempty"`
}

type IbcComposabilityMwMemoGatewayTransfer struct {
	GatewayIbcTokenBridgePayloadTransfer GatewayIbcTokenBridgePayloadTransfer `json:"gateway_ibc_token_bridge_payload"`
}

func CreateIbcComposabilityMwMemoGatewayTransfer(t *testing.T, chainID uint16, recipient []byte, fee uint64, nonce uint32) string {
	msg := IbcComposabilityMwMemoGatewayTransfer{
		GatewayIbcTokenBridgePayloadTransfer{
			GatewayTransfer: GatewayTransfer{
				Chain:     chainID,
				Recipient: recipient,
				Fee:       fmt.Sprint(fee),
				Nonce:     nonce,
			},
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

type IbcComposabilityMwMemoGatewayTransferWithPayload struct {
	GatewayIbcTokenBridgePayloadTransferWithPayload GatewayIbcTokenBridgePayloadTransferWithPayload `json:"gateway_ibc_token_bridge_payload"`
}

func CreateIbcComposabilityMwMemoGatewayTransferWithPayload(t *testing.T, chainID uint16, externalContract []byte, payload []byte, nonce uint32) string {
	msg := IbcComposabilityMwMemoGatewayTransferWithPayload{
		GatewayIbcTokenBridgePayloadTransferWithPayload{
			GatewayTransferWithPayload: GatewayTransferWithPayload{
				Chain:    chainID,
				Contract: externalContract,
				Payload:  payload,
				Nonce:    nonce,
			},
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

func SubmitIbcReceiverUpdateChannelChainMsg(t *testing.T, allowlistChainID uint16, allowlistChannel string, guardians *guardians.ValSet) string {
	payload := new(bytes.Buffer)
	module, err := vaa.LeftPadBytes("WormchainCore", 32)
	require.NoError(t, err)
	payload.Write(module.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, uint8(1))
	vaa.MustWrite(payload, binary.BigEndian, uint16(0))
	channelPadded, err := vaa.LeftPadBytes(string(allowlistChannel), 64)
	require.NoError(t, err)
	payload.Write(channelPadded.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, allowlistChainID)

	var channelIdBytes [64]byte
	copy(channelIdBytes[:], channelPadded.Bytes())

	// v := generateVaa(0, guardians, vaa.GovernanceChain, vaa.GovernanceEmitter, payload.Bytes())
	// vBz, err := v.Marshal()
	// require.NoError(t, err)

	msg := vaa.BodyIbcUpdateChannelChain{
		TargetChainId: 3104,
		ChannelId:     channelIdBytes,
		ChainId:       vaa.ChainID(allowlistChainID),
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}
