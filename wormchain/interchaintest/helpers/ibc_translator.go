package helpers

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type IbcTranslatorInstantiateMsg struct {
	TokenBridgeContract string `json:"token_bridge_contract"`
	CoreContract string `json:"wormhole_contract"`
}

func IbcTranslatorContractInstantiateMsg(t *testing.T, tbContract string, coreContract string) string {
	msg := IbcTranslatorInstantiateMsg{
		TokenBridgeContract: tbContract,
		CoreContract: coreContract,
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
	module := vaa.LeftPadBytes("IbcTranslator", 32)
	payload.Write(module.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, uint8(1))
	vaa.MustWrite(payload, binary.BigEndian, uint16(0))
	channelPadded := vaa.LeftPadBytes(allowlistChannel, 64)
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

	tokenAddrPadded := vaa.LeftPadBytes(tokenAddr, 32)
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

	tokenAddrPadded := vaa.LeftPadBytes(tokenAddr, 32)
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

type GatewayIbcTokenBridgePayloadSimple struct {
	Simple Simple `json:"simple"`
}

type Simple struct {
	Chain uint16 `json:"chain"`
	Recipient []byte `json:"recipient"`
	Fee string `json:"fee"`
	Nonce uint32 `json:"nonce"`
}

func CreateGatewayIbcTokenBridgePayloadSimple(t *testing.T, chainID uint16, recipient string, fee uint64, nonce uint32) []byte {
	msg := GatewayIbcTokenBridgePayloadSimple{
		Simple: Simple{
			Chain: chainID,
			Recipient: []byte(recipient),
			Fee: fmt.Sprint(fee),
			Nonce: nonce,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

type GatewayIbcTokenBridgePayloadContractControlled struct {
	ContractControlled ContractControlled `json:"contract_controlled"`
}

type ContractControlled struct {
	Chain uint16 `json:"chain"`
	Contract []byte `json:"contract"`
	Payload []byte `json:"payload"`
	Nonce uint32 `json:"nonce"`
}

func CreateGatewayIbcTokenBridgePayloadContract(t *testing.T, chainID uint16, contract string, payload []byte, nonce uint32) []byte {
	msg := GatewayIbcTokenBridgePayloadContractControlled{
		ContractControlled: ContractControlled{
			Chain: chainID,
			Contract: []byte(contract),
			Payload: payload,
			Nonce: nonce,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

type IbcTranslatorExecuteSimple struct {
	Msg Simple `json:"simple_convert_and_transfer"`
}

// Temporary method for test the contract interface before the middleware is available
func CreateIbcTranslatorExecuteSimple(t *testing.T, chainID uint16, recipient string, fee uint64, nonce uint32) string {
	msg :=  IbcTranslatorExecuteSimple {
		Msg: Simple{
			Chain: chainID,
			Recipient: []byte(recipient),
			Fee: fmt.Sprint(fee),
			Nonce: nonce,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

type IbcTranslatorExecuteContractControlled struct {
	Msg ContractControlled `json:"contract_controlled_convert_and_transfer"`
}

// Temporary method for testing the contract interface before the middleware is available
func CreateIbcTranslatorExecuteContractControlled(t *testing.T, chainID uint16, contract string, payload []byte, nonce uint32) string {
	msg := IbcTranslatorExecuteContractControlled{
		Msg: ContractControlled{
			Chain: chainID,
			Contract: []byte(contract),
			Payload: payload,
			Nonce: nonce,
		},
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
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