package cosmwasm

import (
	"errors"
	"fmt"

	"github.com/tidwall/gjson"
)

// Defense-in-depth: explicitly check the transaction status code before processing events.
//
// In the Cosmos SDK, failed transactions (code != 0) should not contain wasm events because
// message execution is rolled back. On failure, DeliverTx only returns ante-handler events
// (e.g. tx.fee), never message-level events like wasm. See the failed-tx branch in DeliverTx,
// which returns only anteEvents:
// https://github.com/cosmos/cosmos-sdk/blob/v0.47.0/baseapp/abci.go#L401-L404
//
// We check explicitly rather than relying on that invariant, consistent with the approach
// taken by the EVM watcher.
//
// A missing code is treated as success. The CometBFT websocket serializes ABCI results with
// `json:"code,omitempty"`, so a successful transaction there has no code field at all. The
// LCD normally emits "code": 0, but we apply the same rule to both sources so they behave
// identically. A code that is present must be a numeric zero.

// ErrTxFailed is returned when a transaction's status code indicates failure or is malformed.
var ErrTxFailed = errors.New("transaction failed")

// LCDTxCodePath is the location of the status code in an LCD GetTx response.
const LCDTxCodePath = "tx_response.code"

// WebsocketTxCodePath is the location of the status code in a CometBFT websocket Tx event.
const WebsocketTxCodePath = "result.data.value.TxResult.result.code"

// CheckTxCode verifies that the status code at path in txJSON describes a successful
// transaction. Use LCDTxCodePath or WebsocketTxCodePath for path.
func CheckTxCode(txJSON string, path string) error {
	code := gjson.Get(txJSON, path)
	if !code.Exists() {
		return nil
	}
	// The uint32 code is encoded as a JSON number. Anything else is malformed, and gjson would
	// coerce it to 0, so reject it rather than treat it as success.
	if code.Type != gjson.Number {
		return fmt.Errorf("%w: non-numeric code %q", ErrTxFailed, code.Raw)
	}
	if code.Num != 0 {
		return fmt.Errorf("%w: code %s", ErrTxFailed, code.Raw)
	}
	return nil
}
