package txverifier

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

func Test_deleteEntries_StringKeys(t *testing.T) {
	tests := []struct {
		name         string
		setupCache   func() *map[string]vaa.Address
		want         int
		wantErr      bool
		wantFinalLen int
	}{
		{
			name: "nil pointer",
			setupCache: func() *map[string]vaa.Address {
				return nil
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "pointer to nil map",
			setupCache: func() *map[string]vaa.Address {
				var m map[string]vaa.Address = nil
				return &m
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "cache within limits - no deletion needed",
			setupCache: func() *map[string]vaa.Address {
				m := make(map[string]vaa.Address)
				// Add entries below CacheMaxSize
				for i := range CacheMaxSize - 10 {
					m[fmt.Sprintf("key%d", i)] = vaa.Address{}
				}
				return &m
			},
			want:         0,
			wantErr:      false,
			wantFinalLen: CacheMaxSize - 10,
		},
		{
			name: "cache exactly at limit - no deletion needed",
			setupCache: func() *map[string]vaa.Address {
				m := make(map[string]vaa.Address)
				for i := range CacheMaxSize {
					m[fmt.Sprintf("key%d", i)] = vaa.Address{}
				}
				return &m
			},
			want:         0,
			wantErr:      false,
			wantFinalLen: CacheMaxSize,
		},
		{
			name: "cache way over limit - delete enough to reach CacheMaxSize",
			setupCache: func() *map[string]vaa.Address {
				m := make(map[string]vaa.Address)
				for i := range CacheMaxSize + 50 {
					m[fmt.Sprintf("key%d", i)] = vaa.Address{}
				}
				return &m
			},
			want:         50, // CacheMaxSize+50-CacheMaxSize = 50 (more than CacheDeleteCount)
			wantErr:      false,
			wantFinalLen: CacheMaxSize,
		},
		{
			name: "small cache over limit",
			setupCache: func() *map[string]vaa.Address {
				m := make(map[string]vaa.Address)
				for i := range CacheMaxSize + 3 {
					m[fmt.Sprintf("key%d", i)] = vaa.Address{}
				}
				return &m
			},
			want:         CacheDeleteCount, // max(10, 3) = 10
			wantErr:      false,
			wantFinalLen: CacheMaxSize + 3 - CacheDeleteCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cachePtr := tt.setupCache()

			// Store original length for verification
			var originalLen int
			if cachePtr != nil && *cachePtr != nil {
				originalLen = len(*cachePtr)
			}

			got, err := deleteEntries(cachePtr)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check return value
			if got != tt.want {
				t.Errorf("deleteEntries() returned %v, want %v", got, tt.want)
				return
			}

			// If no error expected, verify the cache state
			if !tt.wantErr && cachePtr != nil && *cachePtr != nil {
				finalLen := len(*cachePtr)
				if finalLen != tt.wantFinalLen {
					t.Errorf("deleteEntries() final cache length = %v, want %v (original: %v, deleted: %v)",
						finalLen, tt.wantFinalLen, originalLen, got)
				}

				// Verify that the returned count matches actual deletions
				expectedDeletions := originalLen - finalLen
				if got != expectedDeletions {
					t.Errorf("deleteEntries() returned %v deletions, but actual deletions = %v",
						got, expectedDeletions)
				}
			}
		})
	}
}

//nolint:gosec // Testing on the uint8 value types, but ranging over a size gives int. The truncation issues don't matter here.
func Test_deleteEntries_AddressKeys(t *testing.T) {
	tests := []struct {
		name         string
		setupCache   func() *map[common.Address]uint8
		want         int
		wantErr      bool
		wantFinalLen int
	}{
		{
			name: "nil pointer",
			setupCache: func() *map[common.Address]uint8 {
				return nil
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "pointer to nil map",
			setupCache: func() *map[common.Address]uint8 {
				var m map[common.Address]uint8 = nil
				return &m
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "cache within limits - no deletion needed",
			setupCache: func() *map[common.Address]uint8 {
				m := make(map[common.Address]uint8)
				// Add entries below CacheMaxSize
				for i := range CacheMaxSize - 10 {
					// TODO needs to be common.Address
					m[common.BytesToAddress([]byte{byte(i)})] = uint8(i)
				}
				return &m
			},
			want:         0,
			wantErr:      false,
			wantFinalLen: CacheMaxSize - 10,
		},
		{
			name: "cache exactly at limit - no deletion needed",
			setupCache: func() *map[common.Address]uint8 {
				m := make(map[common.Address]uint8)
				for i := range CacheMaxSize {
					m[common.BytesToAddress([]byte{byte(i)})] = uint8(i)
				}
				return &m
			},
			want:         0,
			wantErr:      false,
			wantFinalLen: CacheMaxSize,
		},
		{
			name: "cache way over limit - delete enough to reach CacheMaxSize",
			setupCache: func() *map[common.Address]uint8 {
				m := make(map[common.Address]uint8)
				for i := range CacheMaxSize + 50 {
					m[common.BytesToAddress([]byte{byte(i)})] = uint8(i)
				}
				return &m
			},
			want:         50, // CacheMaxSize+50-CacheMaxSize = 50 (more than CacheDeleteCount)
			wantErr:      false,
			wantFinalLen: CacheMaxSize,
		},
		{
			name: "small cache over limit",
			setupCache: func() *map[common.Address]uint8 {
				m := make(map[common.Address]uint8)
				for i := range CacheMaxSize + 3 {
					m[common.BytesToAddress([]byte{byte(i)})] = uint8(i)
				}
				return &m
			},
			want:         CacheDeleteCount, // max(10, 3) = 10
			wantErr:      false,
			wantFinalLen: CacheMaxSize + 3 - CacheDeleteCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cachePtr := tt.setupCache()

			// Store original length for verification
			var originalLen int
			if cachePtr != nil && *cachePtr != nil {
				originalLen = len(*cachePtr)
			}

			got, err := deleteEntries(cachePtr)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check return value
			if got != tt.want {
				t.Errorf("deleteEntries() returned %v, want %v", got, tt.want)
				return
			}

			// If no error expected, verify the cache state
			if !tt.wantErr && cachePtr != nil && *cachePtr != nil {
				finalLen := len(*cachePtr)
				if finalLen != tt.wantFinalLen {
					t.Errorf("deleteEntries() final cache length = %v, want %v (original: %v, deleted: %v)",
						finalLen, tt.wantFinalLen, originalLen, got)
				}

				// Verify that the returned count matches actual deletions
				expectedDeletions := originalLen - finalLen
				if got != expectedDeletions {
					t.Errorf("deleteEntries() returned %v deletions, but actual deletions = %v",
						got, expectedDeletions)
				}
			}
		})
	}
}
