package helpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type IbcHooks struct {
	Payload IbcHooksPayload `json:"wasm"`
}

type IbcHooksPayload struct {
	Contract string          `json:"contract"`
	Msg      IbcHooksExecute `json:"msg"`
}

type IbcHooksExecute struct {
	Forward IbcHooksForward `json:"forward_tokens"`
}

type IbcHooksForward struct {
	Recipient string `json:"recipient"`
}

func CreateIbcHooksMsg(t *testing.T, contract string, recipient string) []byte {
	msg := IbcHooks{
		Payload: IbcHooksPayload{
			Contract: contract,
			Msg: IbcHooksExecute{
				Forward: IbcHooksForward{
					Recipient: recipient,
				},
			},
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}
