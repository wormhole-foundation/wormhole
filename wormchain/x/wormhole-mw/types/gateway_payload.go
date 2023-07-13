package types

type GatewayIbcTokenBridgePayload struct {
	GatewayIbcTokenBridgePayloadObj GatewayIbcTokenBridgePayloadObj `json:"gateway_ibc_token_bridge_payload"`
}

type GatewayIbcTokenBridgePayloadObj struct {
	Simple SimplePayload `json:"simple,omitempty"`
	ContractControlled ContractControlledPayload `json:"contract_controlled,omitempty"`
}

type SimplePayload struct {
	Chain uint16 `json:"chain"`
	Recipient string `json:"recipient"`
	Fee string `json:"fee"`
	Nonce uint32 `json:"nonce"`
}

type ContractControlledPayload struct {
	Chain uint16 `json:"chain"`
	Contract string `json:"contract"`
	Payload []byte `json:"payload"`
	Nonce uint32 `json:"nonce"`
}