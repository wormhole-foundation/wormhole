package evm

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
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

func TestCcqBatchHasRevert(t *testing.T) {
	type test struct {
		name     string
		batch    []rpc.BatchElem
		numCalls int
		expected bool
	}

	tests := []test{
		{
			name:     "no errors",
			batch:    []rpc.BatchElem{{}, {}},
			numCalls: 2,
			expected: false,
		},
		{
			name: "revert error",
			batch: []rpc.BatchElem{
				{Error: fmt.Errorf("execution reverted")},
			},
			numCalls: 1,
			expected: true,
		},
		{
			name: "non-revert error",
			batch: []rpc.BatchElem{
				{Error: fmt.Errorf("connection refused")},
			},
			numCalls: 1,
			expected: false,
		},
		{
			name: "revert past numCalls boundary",
			batch: []rpc.BatchElem{
				{},
				{Error: fmt.Errorf("execution reverted")},
			},
			numCalls: 1,
			expected: false,
		},
		{
			name: "mixed case revert",
			batch: []rpc.BatchElem{
				{Error: fmt.Errorf("Execution Reverted")},
			},
			numCalls: 1,
			expected: true,
		},
		{
			name: "revert with extra context",
			batch: []rpc.BatchElem{
				{Error: fmt.Errorf("execution reverted: insufficient balance")},
			},
			numCalls: 1,
			expected: true,
		},
		{
			name: "numCalls exceeds batch length",
			batch: []rpc.BatchElem{
				{Error: fmt.Errorf("execution reverted")},
			},
			numCalls: 2,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ccqBatchHasRevert(tc.batch, tc.numCalls)
			assert.Equal(t, tc.expected, result)
		})
	}
}
