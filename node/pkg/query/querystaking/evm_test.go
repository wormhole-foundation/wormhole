package querystaking

import (
	"testing"

	"github.com/holiman/uint256"
)

// Helper functions for tests

func toBytes32(s string) [32]byte {
	var arr [32]byte
	copy(arr[:], s)
	return arr
}

// Helper function to build StakeInfo bytes for testing
func buildStakeInfoBytes(amount, conversionTableIndex, lockupEnd, accessEnd, lastClaimed, capacity uint64) []byte {
	result := make([]byte, 192)

	// amount (uint256) - bytes 0-31
	amountInt := uint256.NewInt(amount)
	amountBytes := amountInt.Bytes32()
	copy(result[0:32], amountBytes[:])

	// conversionTableIndex (uint256) - bytes 32-63
	indexInt := uint256.NewInt(conversionTableIndex)
	indexBytes := indexInt.Bytes32()
	copy(result[32:64], indexBytes[:])

	// lockupEnd (uint48) - bytes 64-69 (but stored in 32 bytes by contract)
	lockupBytes := uint256.NewInt(lockupEnd).Bytes32()
	copy(result[64:96], lockupBytes[:])

	// accessEnd (uint48) - bytes 96-101 (but stored in 32 bytes by contract)
	accessBytes := uint256.NewInt(accessEnd).Bytes32()
	copy(result[96:128], accessBytes[:])

	// lastClaimed (uint48) - bytes 128-133 (but stored in 32 bytes by contract)
	claimedBytes := uint256.NewInt(lastClaimed).Bytes32()
	copy(result[128:160], claimedBytes[:])

	// capacity (uint256) - bytes 160-191
	capacityInt := uint256.NewInt(capacity)
	capacityBytes := capacityInt.Bytes32()
	copy(result[160:192], capacityBytes[:])

	return result
}

// TestParseStakeInfo tests parsing of stake info from contract data
func TestParseStakeInfo(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantError bool
		validate  func(*testing.T, *StakeInfo)
	}{
		{
			name: "valid stake info with all fields",
			input: buildStakeInfoBytes(
				1000,  // amount
				2,     // conversionTableIndex
				10000, // lockupEnd
				20000, // accessEnd
				5000,  // lastClaimed
				1000,  // capacity
			),
			wantError: false,
			validate: func(t *testing.T, si *StakeInfo) {
				if si.Amount.Uint64() != 1000 {
					t.Errorf("Amount = %d, want 1000", si.Amount.Uint64())
				}
				if si.ConversionTableIndex.Uint64() != 2 {
					t.Errorf("ConversionTableIndex = %d, want 2", si.ConversionTableIndex.Uint64())
				}
				if si.LockupEnd != 10000 {
					t.Errorf("LockupEnd = %d, want 10000", si.LockupEnd)
				}
				if si.AccessEnd != 20000 {
					t.Errorf("AccessEnd = %d, want 20000", si.AccessEnd)
				}
				if si.LastClaimed != 5000 {
					t.Errorf("LastClaimed = %d, want 5000", si.LastClaimed)
				}
				if si.Capacity.Uint64() != 1000 {
					t.Errorf("Capacity = %d, want 1000", si.Capacity.Uint64())
				}
			},
		},
		{
			name: "valid stake info with zero values",
			input: buildStakeInfoBytes(
				0, // amount
				0, // conversionTableIndex
				0, // lockupEnd
				0, // accessEnd
				0, // lastClaimed
				0, // capacity
			),
			wantError: false,
			validate: func(t *testing.T, si *StakeInfo) {
				if si.Amount.Uint64() != 0 {
					t.Errorf("Amount = %d, want 0", si.Amount.Uint64())
				}
				if si.ConversionTableIndex.Uint64() != 0 {
					t.Errorf("ConversionTableIndex = %d, want 0", si.ConversionTableIndex.Uint64())
				}
			},
		},
		{
			name: "valid stake info with max uint48 values",
			input: buildStakeInfoBytes(
				999999999,
				100,
				281474976710655,
				281474976710655,
				281474976710655,
				999999999,
			),
			wantError: false,
			validate: func(t *testing.T, si *StakeInfo) {
				if si.LockupEnd != 281474976710655 {
					t.Errorf("LockupEnd = %d, want max uint48", si.LockupEnd)
				}
			},
		},
		{
			name:      "invalid length - too short",
			input:     make([]byte, 191),
			wantError: true,
		},
		{
			name:      "invalid length - too long",
			input:     make([]byte, 193),
			wantError: true,
		},
		{
			name:      "empty input",
			input:     []byte{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStakeInfo(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseStakeInfo() error = nil, wantError = true")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseStakeInfo() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, got)
			}
		})
	}
}

// TestStakeInfoMethods tests StakeInfo helper methods
func TestStakeInfoMethods(t *testing.T) {
	t.Run("HasStake", func(t *testing.T) {
		tests := []struct {
			name  string
			stake *StakeInfo
			want  bool
		}{
			{
				name:  "nil amount",
				stake: &StakeInfo{Amount: nil},
				want:  false,
			},
			{
				name:  "zero amount",
				stake: &StakeInfo{Amount: uint256.NewInt(0)},
				want:  false,
			},
			{
				name:  "non-zero amount",
				stake: &StakeInfo{Amount: uint256.NewInt(1000)},
				want:  true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := tt.stake.HasStake(); got != tt.want {
					t.Errorf("HasStake() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("HasExpired", func(t *testing.T) {
		tests := []struct {
			name      string
			accessEnd uint64
			timestamp uint64
			want      bool
		}{
			{
				name:      "before access end",
				accessEnd: 2000,
				timestamp: 1500,
				want:      false,
			},
			{
				name:      "at access end",
				accessEnd: 2000,
				timestamp: 2000,
				want:      true,
			},
			{
				name:      "after access end",
				accessEnd: 2000,
				timestamp: 2500,
				want:      true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				stake := &StakeInfo{AccessEnd: tt.accessEnd}
				if got := stake.HasExpired(tt.timestamp); got != tt.want {
					t.Errorf("HasExpired() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

// TestParseRateString tests rate string parsing
func TestParseRateString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      uint64
		wantError bool
	}{
		{
			name:      "1 QPS",
			input:     "1 QPS",
			want:      60,
			wantError: false,
		},
		{
			name:      "10 QPS",
			input:     "10 QPS",
			want:      600,
			wantError: false,
		},
		{
			name:      "1 QPM",
			input:     "1 QPM",
			want:      1,
			wantError: false,
		},
		{
			name:      "100 QPM",
			input:     "100 QPM",
			want:      100,
			wantError: false,
		},
		{
			name:      "lowercase qps",
			input:     "5 qps",
			want:      300,
			wantError: false,
		},
		{
			name:      "lowercase qpm",
			input:     "50 qpm",
			want:      50,
			wantError: false,
		},
		{
			name:      "invalid format - no space",
			input:     "1QPS",
			wantError: true,
		},
		{
			name:      "invalid format - no unit",
			input:     "1",
			wantError: true,
		},
		{
			name:      "invalid unit",
			input:     "1 RPS",
			wantError: true,
		},
		{
			name:      "invalid number",
			input:     "abc QPS",
			wantError: true,
		},
		{
			name:      "negative number",
			input:     "-1 QPS",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRateString(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseRateString() error = nil, wantError = true")
				}
				return
			}

			if err != nil {
				t.Errorf("parseRateString() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("parseRateString() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestConversionTableGetTranchesByChain tests the GetTranchesByChain method
func TestConversionTableGetTranchesByChain(t *testing.T) {
	tests := []struct {
		name      string
		table     ConversionTable
		chainName string
		want      []ConversionTranche
		wantError bool
	}{
		{
			name: "EVM chain with single tranche",
			table: ConversionTable{
				EVM: map[string]string{
					"5000": "1 QPM",
				},
			},
			chainName: "EVM",
			want: []ConversionTranche{
				{Rate: 1, Tranche: 5000},
			},
			wantError: false,
		},
		{
			name: "EVM chain with multiple tranches",
			table: ConversionTable{
				EVM: map[string]string{
					"5000":   "1 QPM",
					"50000":  "1 QPS",
					"500000": "10 QPS",
				},
			},
			chainName: "EVM",
			want: []ConversionTranche{
				{Rate: 1, Tranche: 5000},
				{Rate: 60, Tranche: 50000},
				{Rate: 600, Tranche: 500000},
			},
			wantError: false,
		},
		{
			name: "Solana chain",
			table: ConversionTable{
				Solana: map[string]string{
					"12500":  "1 QPM",
					"125000": "1 QPS",
				},
			},
			chainName: "Solana",
			want: []ConversionTranche{
				{Rate: 1, Tranche: 12500},
				{Rate: 60, Tranche: 125000},
			},
			wantError: false,
		},
		{
			name: "unknown chain",
			table: ConversionTable{
				EVM: map[string]string{
					"5000": "1 QPM",
				},
			},
			chainName: "Bitcoin",
			wantError: true,
		},
		{
			name: "chain with no rates",
			table: ConversionTable{
				EVM: nil,
			},
			chainName: "EVM",
			wantError: true,
		},
		{
			name: "invalid tranche amount",
			table: ConversionTable{
				EVM: map[string]string{
					"abc": "1 QPM",
				},
			},
			chainName: "EVM",
			wantError: true,
		},
		{
			name: "invalid rate string",
			table: ConversionTable{
				EVM: map[string]string{
					"5000": "invalid",
				},
			},
			chainName: "EVM",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.table.GetTranchesByChain(tt.chainName)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetTranchesByChain() error = nil, wantError = true")
				}
				return
			}

			if err != nil {
				t.Errorf("GetTranchesByChain() unexpected error = %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetTranchesByChain() got %d tranches, want %d", len(got), len(tt.want))
				return
			}

			// Check that tranches are sorted by tranche amount
			for i := 1; i < len(got); i++ {
				if got[i].Tranche <= got[i-1].Tranche {
					t.Errorf("GetTranchesByChain() tranches not sorted: %v", got)
					break
				}
			}

			// Check each tranche
			for i := range got {
				if got[i].Rate != tt.want[i].Rate {
					t.Errorf("GetTranchesByChain() tranche[%d].Rate = %d, want %d", i, got[i].Rate, tt.want[i].Rate)
				}
				if got[i].Tranche != tt.want[i].Tranche {
					t.Errorf("GetTranchesByChain() tranche[%d].Tranche = %d, want %d", i, got[i].Tranche, tt.want[i].Tranche)
				}
			}
		})
	}
}
