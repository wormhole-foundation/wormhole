package txverifier

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestExtractFromJsonPath(t *testing.T) {
	testcases := []struct {
		name     string
		data     json.RawMessage
		path     string
		expected interface{}
		wantErr  bool
		typ      string
	}{
		{
			name:     "ValidPathString",
			data:     json.RawMessage(`{"key1": {"key2": "value"}}`),
			path:     "key1.key2",
			expected: "value",
			wantErr:  false,
			typ:      "string",
		},
		{
			name:     "ValidPathFloat",
			data:     json.RawMessage(`{"key1": {"key2": 123.45}}`),
			path:     "key1.key2",
			expected: 123.45,
			wantErr:  false,
			typ:      "float64",
		},
		{
			name:     "InvalidPath",
			data:     json.RawMessage(`{"key1": {"key2": "value"}}`),
			path:     "key1.key3",
			expected: nil,
			wantErr:  true,
			typ:      "string",
		},
		{
			name:     "NestedPath",
			data:     json.RawMessage(`{"key1": {"key2": {"key3": "value"}}}`),
			path:     "key1.key2.key3",
			expected: "value",
			wantErr:  false,
			typ:      "string",
		},
		{
			name:     "EmptyPath",
			data:     json.RawMessage(`{"key1": {"key2": "value"}}`),
			path:     "",
			expected: nil,
			wantErr:  true,
			typ:      "string",
		},
		{
			name:     "NonExistentPath",
			data:     json.RawMessage(`{"key1": {"key2": "value"}}`),
			path:     "key3.key4",
			expected: nil,
			wantErr:  true,
			typ:      "string",
		},
		{
			name:     "MalformedJson",
			data:     json.RawMessage(`{"key1": {"key2": "value"`),
			path:     "key1.key2",
			expected: nil,
			wantErr:  true,
			typ:      "string",
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			var err error
			switch tt.typ {
			case "string":
				var res string
				res, err = extractFromJsonPath[string](tt.data, tt.path)
				result = res
			case "float64":
				var res float64
				res, err = extractFromJsonPath[float64](tt.data, tt.path)
				result = res
			default:
				t.Fatalf("Unsupported type: %v", tt.typ)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error: %v, got: %v", tt.wantErr, err)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	testcases := []struct {
		name     string
		amount   *big.Int
		decimals uint8
		expected *big.Int
	}{
		{
			name:     "AmountWithMoreThan8Decimals",
			amount:   big.NewInt(1000000000000000000),
			decimals: 18,
			expected: big.NewInt(100000000),
		},
		{
			name:     "AmountWithExactly8Decimals",
			amount:   big.NewInt(12345678),
			decimals: 8,
			expected: big.NewInt(12345678),
		},
		{
			name:     "AmountWithLessThan8Decimals",
			amount:   big.NewInt(12345),
			decimals: 5,
			expected: big.NewInt(12345),
		},
		{
			name:     "AmountWithZeroDecimals",
			amount:   big.NewInt(12345678),
			decimals: 0,
			expected: big.NewInt(12345678),
		},
		{
			name:     "AmountWith9Decimals",
			amount:   big.NewInt(123456789),
			decimals: 9,
			expected: big.NewInt(12345678),
		},
		{
			name:     "AmountWith10Decimals",
			amount:   big.NewInt(1234567890),
			decimals: 10,
			expected: big.NewInt(12345678),
		},
		{
			name:     "AmountEqualsNil",
			amount:   nil,
			decimals: 18,
			expected: nil,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			result := normalize(tt.amount, tt.decimals)
			if result.Cmp(tt.expected) != 0 {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDenormalize(t *testing.T) {
	t.Parallel() // marks TLog as capable of running in parallel with other tests
	tests := map[string]struct {
		amount   *big.Int
		decimals uint8
		expected *big.Int
	}{
		"noop: decimals less than 8": {
			amount:   big.NewInt(123000),
			decimals: 1,
			expected: big.NewInt(123000),
		},
		"noop: decimals equal to 8": {
			amount:   big.NewInt(123000),
			decimals: 8,
			expected: big.NewInt(123000),
		},
		"denormalize: decimals greater than 8": {
			amount:   big.NewInt(123000),
			decimals: 12,
			expected: big.NewInt(1230000000),
		},
		// NOTE: some tokens on NEAR have as many as 24 decimals so this isn't a strict limit for Wormhole
		// overall, but should be true for EVM chains.
		"denormalize: decimals at maximum expected size": {
			amount:   big.NewInt(123_000_000),
			decimals: 18,
			expected: big.NewInt(1_230_000_000_000_000_000),
		},
		// https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0003_token_bridge.md#handling-of-token-amounts-and-decimals
		"denormalize: whitepaper example 1": {
			amount:   big.NewInt(100000000),
			decimals: 18,
			expected: big.NewInt(1000000000000000000),
		},
		"denormalize: whitepaper example 2": {
			amount:   big.NewInt(20000),
			decimals: 4,
			expected: big.NewInt(20000),
		},
	}
	for name, test := range tests {
		test := test // NOTE: uncomment for Go < 1.22, see /doc/faq#closures_and_goroutines
		t.Run(name, func(t *testing.T) {
			t.Parallel() // marks each test case as capable of running in parallel with each other

			if got := denormalize(test.amount, test.decimals); got.Cmp(test.expected) != 0 {
				t.Fatalf("denormalize(%s, %d) returned %s; expected %s",
					test.amount.String(),
					test.decimals,
					got,
					test.expected.String(),
				)
			}

		})
	}
}

func TestValidateChains(t *testing.T) {
	type args struct {
		input []uint
	}
	tests := []struct {
		name    string
		args    args
		want    []vaa.ChainID
		wantErr bool
	}{
		{
			name: "invalid chainId",
			args: args{
				input: []uint{65535},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "unsupported chainId",
			args: args{
				input: []uint{22},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty input",
			args: args{
				input: []uint{},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "happy path",
			args: args{
				input: []uint{2},
			},
			want:    []vaa.ChainID{vaa.ChainIDEthereum},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateChains(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChains() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateChains() = %v, want %v", got, tt.want)
			}
		})
	}
}
