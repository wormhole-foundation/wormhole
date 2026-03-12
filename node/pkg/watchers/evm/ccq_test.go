package evm

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/query"
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

type testRPCError struct {
	code int
	msg  string
}

func (e testRPCError) Error() string  { return e.msg }
func (e testRPCError) ErrorCode() int { return e.code }

func TestCcqIsExecutionRevert(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "rpc code 3",
			err:  testRPCError{code: 3, msg: "execution reverted"},
			want: true,
		},
		{
			name: "rpc non-3",
			err:  testRPCError{code: -32000, msg: "some rpc error"},
			want: false,
		},
		{
			name: "string match",
			err:  errors.New("execution reverted: insufficient balance"),
			want: true,
		},
		{
			name: "string match reverted",
			err:  errors.New("transaction reverted"),
			want: true,
		},
		{
			name: "string match vm execution error",
			err:  errors.New("VM execution error"),
			want: true,
		},
		{
			name: "string match invalid opcode",
			err:  errors.New("invalid opcode"),
			want: true,
		},
		{
			name: "string non-match",
			err:  errors.New("connection refused"),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ccqIsExecutionRevert(tc.err))
		})
	}
}

func TestCcqVerifyAndExtractQueryResults_Status(t *testing.T) {
	w := &Watcher{ccqLogger: zap.NewNop()}
	requestId := "test"

	makeCallData := func(result hexutil.Bytes, callErr error) EvmCallData {
		return EvmCallData{
			CallResult: &result,
			callErr:    callErr,
		}
	}

	tests := []struct {
		name   string
		batch  []rpc.BatchElem
		calls  []EvmCallData
		status query.QueryStatus
	}{
		{
			name:   "fatal on revert",
			batch:  []rpc.BatchElem{{Error: testRPCError{code: 3, msg: "execution reverted"}}},
			calls:  []EvmCallData{makeCallData(hexutil.Bytes{0x01}, nil)},
			status: query.QueryFatalError,
		},
		{
			name:   "retry on non-revert batch error",
			batch:  []rpc.BatchElem{{Error: errors.New("connection refused")}},
			calls:  []EvmCallData{makeCallData(hexutil.Bytes{0x01}, nil)},
			status: query.QueryRetryNeeded,
		},
		{
			name:   "retry on callErr",
			batch:  []rpc.BatchElem{{}},
			calls:  []EvmCallData{makeCallData(hexutil.Bytes{0x01}, errors.New("dial error"))},
			status: query.QueryRetryNeeded,
		},
		{
			name:   "retry on empty result",
			batch:  []rpc.BatchElem{{}},
			calls:  []EvmCallData{makeCallData(hexutil.Bytes{}, nil)},
			status: query.QueryRetryNeeded,
		},
		{
			name:   "success",
			batch:  []rpc.BatchElem{{}},
			calls:  []EvmCallData{makeCallData(hexutil.Bytes{0x01, 0x02}, nil)},
			status: query.QuerySuccess,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, status, err := w.ccqVerifyAndExtractQueryResults(requestId, tc.batch, tc.calls)
			if tc.status == query.QuerySuccess {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, tc.status, status)
		})
	}
}
