package types

import (
	"encoding/json"
)

type IbcTranslatorIbcHooksSimple struct {
	Payload IbcTranslatorIbcHooksPayloadSimple `json:"wasm"`
}

type IbcTranslatorIbcHooksPayloadSimple struct {
	Contract string                     `json:"contract"`
	Msg      IbcTranslatorExecuteSimple `json:"msg"`
}

type IbcTranslatorExecuteSimple struct {
	Msg Simple `json:"simple_convert_and_transfer"`
}

type Simple struct {
	Chain     uint16 `json:"chain"`
	Recipient []byte `json:"recipient"`
	Fee       string `json:"fee"`
	Nonce     uint32 `json:"nonce"`
}

type IbcTranslatorIbcHooksContractControlled struct {
	Payload IbcTranslatorIbcHooksPayloadContractControlled `json:"wasm"`
}

type IbcTranslatorIbcHooksPayloadContractControlled struct {
	Contract string                                 `json:"contract"`
	Msg      IbcTranslatorExecuteContractControlled `json:"msg"`
}

type IbcTranslatorExecuteContractControlled struct {
	Msg ContractControlled `json:"contract_controlled_convert_and_transfer"`
}

type ContractControlled struct {
	Chain    uint16 `json:"chain"`
	Contract []byte `json:"contract"`
	Payload  []byte `json:"payload"`
	Nonce    uint32 `json:"nonce"`
}

func FormatIbcHooksMemo(parsedPayload ParsedPayload, middlewareContract string) (string, error) {
	// If exists, create PFM memo
	var ibcHooksMemo string
	if parsedPayload.IsSimple {
		simple := IbcTranslatorIbcHooksSimple{
			Payload: IbcTranslatorIbcHooksPayloadSimple{
				Contract: middlewareContract,
				Msg: IbcTranslatorExecuteSimple{
					Msg: Simple{
						Chain:     parsedPayload.ChainId,
						Recipient: parsedPayload.Recipient,
						Fee:       parsedPayload.Fee,
						Nonce:     parsedPayload.Nonce,
					},
				},
			},
		}
		simpleBz, err := json.Marshal(&simple)
		if err != nil {
			return "", err
		}
		ibcHooksMemo = string(simpleBz)
	} else {
		cc := IbcTranslatorIbcHooksContractControlled{
			Payload: IbcTranslatorIbcHooksPayloadContractControlled{
				Contract: middlewareContract,
				Msg: IbcTranslatorExecuteContractControlled{
					Msg: ContractControlled{
						Chain:    parsedPayload.ChainId,
						Contract: parsedPayload.Recipient,
						Payload:  parsedPayload.Payload,
						Nonce:    parsedPayload.Nonce,
					},
				},
			},
		}
		ccBz, err := json.Marshal(&cc)
		if err != nil {
			return "", err
		}
		ibcHooksMemo = string(ccBz)
	}
	return ibcHooksMemo, nil
}
