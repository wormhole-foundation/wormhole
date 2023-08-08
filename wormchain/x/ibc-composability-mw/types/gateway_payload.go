package types

import (
	"encoding/json"
	"fmt"
)

type GatewayIbcTokenBridgePayload struct {
	GatewayIbcTokenBridgePayloadObj GatewayIbcTokenBridgePayloadObj `json:"gateway_ibc_token_bridge_payload"`
}

type GatewayIbcTokenBridgePayloadObj struct {
	Transfer            GatewayTransfer            `json:"gateway_transfer,omitempty"`
	TransferWithPayload GatewayTransferWithPayload `json:"gateway_transfer_with_payload,omitempty"`
}

type GatewayTransfer struct {
	Chain     uint16 `json:"chain,omitempty"`
	Recipient []byte `json:"recipient,omitempty"`
	Fee       string `json:"fee,omitempty"`
	Nonce     uint32 `json:"nonce,omitempty"`
}

type GatewayTransferWithPayload struct {
	Chain    uint16 `json:"chain,omitempty"`
	Contract []byte `json:"contract,omitempty"`
	Payload  []byte `json:"payload,omitempty"`
	Nonce    uint32 `json:"nonce,omitempty"`
}

type ParsedPayload struct {
	NoPayload bool
	ChainId   uint16
	Recipient []byte
	Fee       string
	Nonce     uint32
	Payload   []byte
}

func VerifyAndParseGatewayPayload(memo string) (ParsedPayload, error) {
	var parsedPayload ParsedPayload

	gatewayIbcTokenBridgePayload := GatewayIbcTokenBridgePayload{}
	err := json.Unmarshal([]byte(memo), &gatewayIbcTokenBridgePayload)
	if err != nil {
		return parsedPayload, fmt.Errorf("ibc-composability-mw: error parsing gateway ibc token bridge payload, %s", err)
	}

	if gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Transfer.Recipient != nil {
		parsedPayload.NoPayload = true
		parsedPayload.ChainId = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Transfer.Chain
		parsedPayload.Recipient = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Transfer.Recipient
		parsedPayload.Fee = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Transfer.Fee
		parsedPayload.Nonce = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Transfer.Nonce
	} else if gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Contract != nil {
		parsedPayload.NoPayload = false
		parsedPayload.ChainId = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Chain
		parsedPayload.Recipient = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Contract
		parsedPayload.Nonce = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Nonce
		parsedPayload.Payload = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Payload
	} else {
		return parsedPayload, fmt.Errorf("ibc-composability-mw: error parsing gateway ibc token bridge payload")
	}

	return parsedPayload, nil
}
