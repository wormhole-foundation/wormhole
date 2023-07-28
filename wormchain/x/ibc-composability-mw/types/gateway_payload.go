package types

import (
	"encoding/json"
	"fmt"
)

type GatewayIbcTokenBridgePayload struct {
	GatewayIbcTokenBridgePayloadObj GatewayIbcTokenBridgePayloadObj `json:"gateway_ibc_token_bridge_payload"`
}

type GatewayIbcTokenBridgePayloadObj struct {
	Simple             SimplePayload             `json:"simple,omitempty"`
	ContractControlled ContractControlledPayload `json:"contract_controlled,omitempty"`
}

type SimplePayload struct {
	Chain     uint16 `json:"chain,omitempty"`
	Recipient []byte `json:"recipient,omitempty"`
	Fee       string `json:"fee,omitempty"`
	Nonce     uint32 `json:"nonce,omitempty"`
}

type ContractControlledPayload struct {
	Chain    uint16 `json:"chain,omitempty"`
	Contract []byte `json:"contract,omitempty"`
	Payload  []byte `json:"payload,omitempty"`
	Nonce    uint32 `json:"nonce,omitempty"`
}

type ParsedPayload struct {
	IsSimple  bool
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

	if gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Simple.Recipient != nil {
		parsedPayload.IsSimple = true
		parsedPayload.ChainId = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Simple.Chain
		parsedPayload.Recipient = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Simple.Recipient
		parsedPayload.Fee = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Simple.Fee
		parsedPayload.Nonce = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.Simple.Nonce
	} else if gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.ContractControlled.Contract != nil {
		parsedPayload.IsSimple = false
		parsedPayload.ChainId = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.ContractControlled.Chain
		parsedPayload.Recipient = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.ContractControlled.Contract
		parsedPayload.Nonce = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.ContractControlled.Nonce
		parsedPayload.Payload = gatewayIbcTokenBridgePayload.GatewayIbcTokenBridgePayloadObj.ContractControlled.Payload
	} else {
		return parsedPayload, fmt.Errorf("ibc-composability-mw: error parsing gateway ibc token bridge payload")
	}

	return parsedPayload, nil
}
