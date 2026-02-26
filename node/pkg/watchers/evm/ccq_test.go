package evm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCcqCreateBlockRequest(t *testing.T) {
	type test struct {
		input              string
		blockMethod        string
		callBlockArgAsJSON string
		errMsg             string
	}

	tests := []test{
		// Failure cases:
		{input: "", blockMethod: "", callBlockArgAsJSON: "", errMsg: "block id is required"},
		{input: "deadbeef", blockMethod: "", callBlockArgAsJSON: "", errMsg: "block id must start with 0x"},
		{input: "0xHelloWorld", blockMethod: "", callBlockArgAsJSON: "", errMsg: "block id is not valid hex"},

		// Success cases:
		{input: "0xb96d7a", blockMethod: "eth_getBlockByNumber", callBlockArgAsJSON: `"0xb96d7a"`, errMsg: ""},
		{input: "0xb96d7a4751d4ec70a6278a92d361e52821416bb6966aabeb596b81f92f4a6263", blockMethod: "eth_getBlockByHash", callBlockArgAsJSON: `{"blockHash":"0xb96d7a4751d4ec70a6278a92d361e52821416bb6966aabeb596b81f92f4a6263","requireCanonical":true}`, errMsg: ""},

		// Block hashes with leading zeros must not be misidentified as block numbers.
		{input: "0x0000000000000000000000000000000000000000000000000000000000000100", blockMethod: "eth_getBlockByHash", callBlockArgAsJSON: `{"blockHash":"0x0000000000000000000000000000000000000000000000000000000000000100","requireCanonical":true}`, errMsg: ""},
		// Block hash with trailing zeros must remain a hash, not get truncated to a number.
		{input: "0x0100000000000000000000000000000000000000000000000000000000000000", blockMethod: "eth_getBlockByHash", callBlockArgAsJSON: `{"blockHash":"0x0100000000000000000000000000000000000000000000000000000000000000","requireCanonical":true}`, errMsg: ""},
		// Block numbers with leading and trailing zeros must be preserved.
		{input: "0x0100", blockMethod: "eth_getBlockByNumber", callBlockArgAsJSON: `"0x0100"`, errMsg: ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			blockMethod, callBlockArg, err := ccqCreateBlockRequest(tc.input)
			if tc.errMsg == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.blockMethod, blockMethod)

				bytes, err := json.Marshal(callBlockArg)
				require.NoError(t, err)
				assert.Equal(t, tc.callBlockArgAsJSON, string(bytes))
			} else {
				assert.EqualError(t, err, tc.errMsg)
			}

		})
	}
}
