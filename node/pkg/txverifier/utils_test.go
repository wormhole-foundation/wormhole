package txverifier

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"sync"
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
		{
			name:     "DataIsNil",
			data:     nil,
			path:     "test",
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

func Test_lockedMap_GetSetLen(t *testing.T) {
	lm := NewLockedMap[string, int]()

	if got := lm.Len(); got != 0 {
		t.Errorf("Len() on new lockedMap = %d, want 0", got)
	}

	// A missing key must return the zero value and exists=false.
	if val, exists := lm.Get("missing"); exists || val != 0 {
		t.Errorf("Get(missing) = (%d, %v), want (0, false)", val, exists)
	}

	// Set then Get returns the stored value.
	lm.Set("key", 42)
	if val, exists := lm.Get("key"); !exists || val != 42 {
		t.Errorf("Get(key) = (%d, %v), want (42, true)", val, exists)
	}

	// Overwriting a key updates the value without changing the length.
	lm.Set("key", 43)
	if val, exists := lm.Get("key"); !exists || val != 43 {
		t.Errorf("Get(key) after overwrite = (%d, %v), want (43, true)", val, exists)
	}
	if got := lm.Len(); got != 1 {
		t.Errorf("Len() after overwrite = %d, want 1", got)
	}

	// A stored zero value must be distinguishable from a missing key.
	lm.Set("zero", 0)
	if _, exists := lm.Get("zero"); !exists {
		t.Error("Get(zero) exists = false, want true for an explicitly stored zero value")
	}
}

func Test_lockedMap_PruneIfFull(t *testing.T) {
	tests := []struct {
		name       string
		numEntries int
		wantGone   int
	}{
		{"empty map", 0, 0},
		{"below the limit", CacheMaxSize - 1, 0},
		{"exactly at the limit", CacheMaxSize, 0},
		{"one over the limit", CacheMaxSize + 1, CacheDeleteCount},
		{"overshoot beyond the limit", CacheMaxSize + 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewLockedMap[string, int]()
			for i := 0; i < tt.numEntries; i++ {
				lm.Set(fmt.Sprintf("key-%04d", i), i)
			}

			// Which entries get deleted is not deterministic, only the count is.
			if got := lm.PruneIfFull(); got != tt.wantGone {
				t.Errorf("PruneIfFull() deleted %d entries, want %d", got, tt.wantGone)
			}
			if wantLen := tt.numEntries - tt.wantGone; lm.Len() != wantLen {
				t.Errorf("Len() after prune = %d, want %d", lm.Len(), wantLen)
			}
		})
	}
}

// Test_lockedMap_ConcurrentAccess exercises concurrent use of every method.
// It has no meaningful sequential assertions; its value is running under
// `go test -race`, where unsynchronized access inside lockedMap would fail.
func Test_lockedMap_ConcurrentAccess(t *testing.T) {
	// Keep the key space just above CacheMaxSize so PruneIfFull has work to do.
	const numKeys = CacheMaxSize + CacheDeleteCount
	const goroutines = 8
	const iterations = 500

	lm := NewLockedMap[string, int]()

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("key-%04d", (g*i)%numKeys)
				lm.Set(key, i)
				lm.Get(key)
				lm.Len()
				if i%50 == 0 {
					lm.PruneIfFull()
				}
			}
		}()
	}
	wg.Wait()

	if got := lm.Len(); got < 0 || got > numKeys {
		t.Errorf("Len() = %d after concurrent access, want within [0, %d]", got, numKeys)
	}
}

func Test_evaluationsLockedMap_PruneOlderThan(t *testing.T) {
	tests := []struct {
		name       string
		blocks     []uint64 // non-nil evaluations, one per block number
		nils       int      // number of nil evaluations
		threshold  uint64
		wantPruned int
	}{
		{"empty map", nil, 0, 100, 0},
		{"nothing older than the threshold", []uint64{100, 150, 200}, 0, 100, 0},
		{"boundary: block numbers equal to the threshold are kept", []uint64{99, 100, 101}, 0, 100, 1},
		{"all entries older than the threshold", []uint64{1, 50, 99}, 0, 100, 3},
		{"nil entries are always pruned", []uint64{150, 200}, 2, 100, 2},
		{"mixed nil and outdated entries", []uint64{40, 150}, 1, 100, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			elm := NewEvaluationsLockedMap()
			nextKey := 0
			for _, block := range tt.blocks {
				elm.Set(common.HexToHash(fmt.Sprintf("%064x", nextKey)), &receiptEvaluation{blockNumber: block})
				nextKey++
			}
			for i := 0; i < tt.nils; i++ {
				elm.Set(common.HexToHash(fmt.Sprintf("%064x", nextKey)), nil)
				nextKey++
			}

			if got := elm.PruneOlderThan(tt.threshold); got != tt.wantPruned {
				t.Errorf("PruneOlderThan(%d) pruned %d entries, want %d", tt.threshold, got, tt.wantPruned)
			}
			if wantRemaining := nextKey - tt.wantPruned; elm.Len() != wantRemaining {
				t.Errorf("Len() after prune = %d, want %d", elm.Len(), wantRemaining)
			}
		})
	}
}

func Test_evaluationsLockedMap_SetIfAbsent(t *testing.T) {
	elm := NewEvaluationsLockedMap()
	key := common.HexToHash("0x1")
	first := &receiptEvaluation{blockNumber: 10}
	second := &receiptEvaluation{blockNumber: 20}

	if !elm.SetIfAbsent(key, first) {
		t.Error("SetIfAbsent on an empty key = false, want true")
	}
	if elm.SetIfAbsent(key, second) {
		t.Error("SetIfAbsent on an occupied key = true, want false")
	}
	// The first writer must win: addToCache relies on cached entries never being replaced.
	if got, exists := elm.Get(key); !exists || got != first {
		t.Error("Get after competing SetIfAbsent calls returned the wrong evaluation, want the first writer's")
	}
	if got := elm.Len(); got != 1 {
		t.Errorf("Len() = %d, want 1", got)
	}
}

// Test_evaluationsLockedMap_ConcurrentAccess exercises the specialized map under
// contention. Like Test_lockedMap_ConcurrentAccess, its value is running under
// `go test -race`.
func Test_evaluationsLockedMap_ConcurrentAccess(t *testing.T) {
	const goroutines = 8
	const iterations = 500

	elm := NewEvaluationsLockedMap()

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := common.HexToHash(fmt.Sprintf("%064x", (g*i)%32))
				elm.Set(key, &receiptEvaluation{blockNumber: uint64(i)})
				elm.SetIfAbsent(key, &receiptEvaluation{blockNumber: 0})
				elm.Get(key)
				if i%50 == 0 {
					elm.PruneOlderThan(uint64(i))
				}
			}
		}()
	}
	wg.Wait()

	if got := elm.Len(); got < 0 || got > 32 {
		t.Errorf("Len() = %d after concurrent access, want within [0, 32]", got)
	}
}

func Test_validateSolvency(t *testing.T) {
	tests := []struct {
		name             string
		requests         MsgIdToRequestOutOfBridge
		transfers        AssetKeyToTransferIntoBridge
		wantErr          bool
		invalidTransfers []string
	}{
		{
			name: "SingleRequest_SingleAsset_Solvent",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(100),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers: AssetKeyToTransferIntoBridge{
				"asset1": {
					Amount: big.NewInt(200),
				},
			},
			wantErr:          false,
			invalidTransfers: []string{},
		},
		{
			name: "SingleRequest_SingleAsset_Insolvent",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(300),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers: AssetKeyToTransferIntoBridge{
				"asset1": {
					Amount: big.NewInt(200),
				},
			},
			wantErr:          false,
			invalidTransfers: []string{"msg1"},
		},
		{
			name: "MultipleRequests_MultipleAssets_MixedSolvency",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(100),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
				"msg2": {
					AssetKey:       "asset2",
					Amount:         big.NewInt(150),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
				"msg3": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(50),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers: AssetKeyToTransferIntoBridge{
				"asset1": {
					Amount: big.NewInt(120),
				},
				"asset2": {
					Amount: big.NewInt(200),
				},
			},
			wantErr: false,
			// asset1 is insolvent because of msg1 and msg3
			invalidTransfers: []string{"msg1", "msg3"},
		},
		{
			name: "RequestWithNoMatchingTransfer",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(100),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers:        AssetKeyToTransferIntoBridge{},
			wantErr:          false,
			invalidTransfers: []string{"msg1"}, // no transfer for asset1
		},
		{
			name:             "EmptyRequests",
			requests:         MsgIdToRequestOutOfBridge{},
			transfers:        AssetKeyToTransferIntoBridge{},
			wantErr:          false,
			invalidTransfers: []string{},
		},
		{
			name: "EmptyTransfers",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(100),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers:        AssetKeyToTransferIntoBridge{},
			wantErr:          false,
			invalidTransfers: []string{"msg1"}, // no transfer for asset1
		},
		{
			name:             "EmptyRequests",
			requests:         MsgIdToRequestOutOfBridge{},
			transfers:        AssetKeyToTransferIntoBridge{},
			wantErr:          false,
			invalidTransfers: []string{},
		},
		{
			name: "RequestWithNilAmount",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         nil,
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers:        AssetKeyToTransferIntoBridge{},
			wantErr:          true,
			invalidTransfers: []string{},
		},
		{
			name: "TransferWithNilAmount",
			requests: MsgIdToRequestOutOfBridge{
				"msg1": {
					AssetKey:       "asset1",
					Amount:         big.NewInt(100),
					DepositMade:    false,
					DepositSolvent: false, // will be updated by validateSolvency
				},
			},
			transfers: AssetKeyToTransferIntoBridge{
				"asset1": {
					Amount: nil,
				},
			},
			wantErr:          true,
			invalidTransfers: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := validateSolvency(tt.requests, tt.transfers)

			if (err != nil) != tt.wantErr {
				t.Errorf("validateSolvency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for msgIdStr, req := range resolved {
				reqIsValid := req.DepositSolvent && req.DepositMade
				shouldBeInvalid := slices.Contains(tt.invalidTransfers, msgIdStr)

				if reqIsValid && !shouldBeInvalid {
					// Valid and should be valid, all good.
				} else if reqIsValid && shouldBeInvalid {
					// Request was marked as valid, but should be invalid
					t.Errorf("Expected message ID %s to be marked as invalid, but it was marked as valid", msgIdStr)
				} else if !reqIsValid && !shouldBeInvalid {
					// Request was marked as invalid, but should be valid
					t.Errorf("Expected message ID %s to be marked as valid, but it was marked as invalid", msgIdStr)
				}
			}
		})
	}
}
