package cosmwasm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckTxCodeLCD(t *testing.T) {
	cases := []struct {
		name    string
		json    string
		wantErr error
	}{
		{"success", `{"tx_response":{"txhash":"AB","code":0,"events":[]}}`, nil},
		{"failed", `{"tx_response":{"txhash":"AB","code":5,"events":[]}}`, ErrTxFailed},
		{"missing code is success", `{"tx_response":{"txhash":"AB","events":[]}}`, nil},
		{"missing tx_response", `{}`, nil},
		{"string zero is rejected", `{"tx_response":{"code":"0"}}`, ErrTxFailed},
		{"string non-zero is rejected", `{"tx_response":{"code":"abc"}}`, ErrTxFailed},
		{"null is rejected", `{"tx_response":{"code":null}}`, ErrTxFailed},
		{"bool is rejected", `{"tx_response":{"code":false}}`, ErrTxFailed},
		{"fractional is rejected", `{"tx_response":{"code":0.5}}`, ErrTxFailed},
		{"negative is rejected", `{"tx_response":{"code":-1}}`, ErrTxFailed},
		{"large code is rejected", `{"tx_response":{"code":4294967295}}`, ErrTxFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckTxCode(tc.json, LCDTxCodePath)
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestCheckTxCodeWebsocket(t *testing.T) {
	wrap := func(result string) string {
		return `{"result":{"data":{"value":{"TxResult":{"height":"1","result":` + result + `}}}}}`
	}

	cases := []struct {
		name    string
		json    string
		wantErr error
	}{
		{"success with code omitted", wrap(`{"events":[]}`), nil},
		{"success with explicit zero", wrap(`{"code":0,"events":[]}`), nil},
		{"failed", wrap(`{"code":5,"events":[]}`), ErrTxFailed},
		{"string code is rejected", wrap(`{"code":"0","events":[]}`), ErrTxFailed},
		{"null is rejected", wrap(`{"code":null,"events":[]}`), ErrTxFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckTxCode(tc.json, WebsocketTxCodePath)
			if tc.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}
