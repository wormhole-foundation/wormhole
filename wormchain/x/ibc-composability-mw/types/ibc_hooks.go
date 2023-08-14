package types

import (
	"encoding/json"
)

// IBC hooks formatted payload for IBC translator's GatewayTransfer
type IbcTranslatorGatewayTransfer struct {
	Payload IbcTranslatorGatewayTransferObj `json:"wasm"`
}

type IbcTranslatorGatewayTransferObj struct {
	Contract string                          `json:"contract"`
	Msg      IbcTranslatorGatewayTransferMsg `json:"msg"`
}

type IbcTranslatorGatewayTransferMsg struct {
	Msg GatewayTransfer `json:"gateway_convert_and_transfer"`
}

// IBC hooks formatted payload for IBC translator's GatewayTransferWithPayload
type IbcTranslatorGatewayTransferWithPayload struct {
	Payload IbcTranslatorGatewayTransferWithPayloadObj `json:"wasm"`
}

type IbcTranslatorGatewayTransferWithPayloadObj struct {
	Contract string                                     `json:"contract"`
	Msg      IbcTranslatorGatewayTransferWithPayloadMsg `json:"msg"`
}

type IbcTranslatorGatewayTransferWithPayloadMsg struct {
	Msg GatewayTransferWithPayload `json:"gateway_convert_and_transfer_with_payload"`
}

func FormatIbcHooksMemo(parsedPayload ParsedPayload, ibcTranslatorContract string) (string, error) {
	var ibcHooksMemo string
	if parsedPayload.NoPayload {
		transfer := IbcTranslatorGatewayTransfer{
			Payload: IbcTranslatorGatewayTransferObj{
				Contract: ibcTranslatorContract,
				Msg: IbcTranslatorGatewayTransferMsg{
					Msg: GatewayTransfer{
						Chain:     parsedPayload.ChainId,
						Recipient: parsedPayload.Recipient,
						Fee:       parsedPayload.Fee,
						Nonce:     parsedPayload.Nonce,
					},
				},
			},
		}
		simpleBz, err := json.Marshal(&transfer)
		if err != nil {
			return "", err
		}
		ibcHooksMemo = string(simpleBz)
	} else {
		transferWithPayload := IbcTranslatorGatewayTransferWithPayload{
			Payload: IbcTranslatorGatewayTransferWithPayloadObj{
				Contract: ibcTranslatorContract,
				Msg: IbcTranslatorGatewayTransferWithPayloadMsg{
					Msg: GatewayTransferWithPayload{
						Chain:    parsedPayload.ChainId,
						Contract: parsedPayload.Recipient,
						Payload:  parsedPayload.Payload,
						Nonce:    parsedPayload.Nonce,
					},
				},
			},
		}
		ccBz, err := json.Marshal(&transferWithPayload)
		if err != nil {
			return "", err
		}
		ibcHooksMemo = string(ccBz)
	}
	return ibcHooksMemo, nil
}
