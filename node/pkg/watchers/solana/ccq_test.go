package solana

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetrySlopIsValid(t *testing.T) {
	assert.Less(t, CCQ_RETRY_SLOP, query.RetryInterval)
}

func TestCcqIsMinContextSlotErrorSuccess(t *testing.T) {
	myErr := &jsonrpc.RPCError{
		Code:    -32016,
		Message: "Minimum context slot has not been reached",
		Data: map[string]interface{}{
			"contextSlot": json.Number("13526"),
		},
	}

	isMinContext, currentSlot, err := ccqIsMinContextSlotError(error(myErr))
	require.NoError(t, err)
	require.True(t, isMinContext)
	assert.Equal(t, uint64(13526), currentSlot)
}

func TestCcqIsMinContextSlotErrorSomeOtherError(t *testing.T) {
	myErr := fmt.Errorf("Some other error")
	isMinContext, _, err := ccqIsMinContextSlotError(error(myErr))
	require.NoError(t, err)
	require.False(t, isMinContext)
}

func TestCcqIsMinContextSlotErrorSomeOtherRPCError(t *testing.T) {
	myErr := &jsonrpc.RPCError{
		Code:    -32000,
		Message: "Some other RPC error",
		Data: map[string]interface{}{
			"contextSlot": json.Number("13526"),
		},
	}

	isMinContext, _, err := ccqIsMinContextSlotError(error(myErr))
	require.NoError(t, err)
	require.False(t, isMinContext)
}

func TestCcqIsMinContextSlotErrorNoData(t *testing.T) {
	myErr := &jsonrpc.RPCError{
		Code:    -32016,
		Message: "Minimum context slot has not been reached",
	}

	_, _, err := ccqIsMinContextSlotError(error(myErr))
	assert.EqualError(t, err, `failed to extract data from min context slot error`)
}

func TestCcqIsMinContextSlotErrorContextSlotMissing(t *testing.T) {
	myErr := &jsonrpc.RPCError{
		Code:    -32016,
		Message: "Minimum context slot has not been reached",
		Data: map[string]interface{}{
			"someOtherField": json.Number("13526"),
		},
	}

	_, _, err := ccqIsMinContextSlotError(error(myErr))
	assert.EqualError(t, err, `min context slot error does not contain "contextSlot"`)
}

func TestCcqIsMinContextSlotErrorContextSlotIsNotAJsonNumber(t *testing.T) {
	myErr := &jsonrpc.RPCError{
		Code:    -32016,
		Message: "Minimum context slot has not been reached",
		Data: map[string]interface{}{
			"contextSlot": "13526",
		},
	}

	_, _, err := ccqIsMinContextSlotError(error(myErr))
	assert.EqualError(t, err, `min context slot error "contextSlot" is not json.Number`)
}

func TestCcqIsMinContextSlotErrorContextSlotIsNotUint64(t *testing.T) {
	myErr := &jsonrpc.RPCError{
		Code:    -32016,
		Message: "Minimum context slot has not been reached",
		Data: map[string]interface{}{
			"contextSlot": json.Number("HelloWorld"),
		},
	}

	_, _, err := ccqIsMinContextSlotError(error(myErr))
	assert.True(t, strings.Contains(err.Error(), `min context slot error "contextSlot" is not uint64`))
}
