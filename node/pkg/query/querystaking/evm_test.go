package querystaking

import (
	"testing"

	"github.com/holiman/uint256"
)

func uint64Ptr(v uint64) *uint64 {
	return &v
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
			name: "valid stake info with large uint256 values",
			input: buildStakeInfoBytes(
				18446744073709551615, // max uint64 for amount
				1000000000000,        // large conversionTableIndex
				281474976710655,      // max uint48 for lockupEnd
				281474976710655,      // max uint48 for accessEnd
				281474976710655,      // max uint48 for lastClaimed
				18446744073709551615, // max uint64 for capacity
			),
			wantError: false,
			validate: func(t *testing.T, si *StakeInfo) {
				if si.Amount.Uint64() != 18446744073709551615 {
					t.Errorf("Amount = %d, want max uint64", si.Amount.Uint64())
				}
				if si.ConversionTableIndex.Uint64() != 1000000000000 {
					t.Errorf("ConversionTableIndex = %d, want 1000000000000", si.ConversionTableIndex.Uint64())
				}
				if si.LockupEnd != 281474976710655 {
					t.Errorf("LockupEnd = %d, want max uint48", si.LockupEnd)
				}
				if si.AccessEnd != 281474976710655 {
					t.Errorf("AccessEnd = %d, want max uint48", si.AccessEnd)
				}
				if si.LastClaimed != 281474976710655 {
					t.Errorf("LastClaimed = %d, want max uint48", si.LastClaimed)
				}
				if si.Capacity.Uint64() != 18446744073709551615 {
					t.Errorf("Capacity = %d, want max uint64", si.Capacity.Uint64())
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
			name                 string
			accessEnd            uint64
			timestamp            uint64
			cacheDurationSeconds uint64
			want                 bool
		}{
			{
				name:                 "before access end (no cache)",
				accessEnd:            2000,
				timestamp:            1500,
				cacheDurationSeconds: 0,
				want:                 false,
			},
			{
				name:                 "at access end (no cache)",
				accessEnd:            2000,
				timestamp:            2000,
				cacheDurationSeconds: 0,
				want:                 true,
			},
			{
				name:                 "after access end (no cache)",
				accessEnd:            2000,
				timestamp:            2500,
				cacheDurationSeconds: 0,
				want:                 true,
			},
			{
				name:                 "before access end but within cache duration",
				accessEnd:            2000,
				timestamp:            1700,
				cacheDurationSeconds: 300,  // 5 minutes
				want:                 true, // 1700 + 300 = 2000, so considered expired
			},
			{
				name:                 "well before access end even with cache",
				accessEnd:            2000,
				timestamp:            1500,
				cacheDurationSeconds: 300,   // 5 minutes
				want:                 false, // 1500 + 300 = 1800 < 2000
			},
			{
				name:                 "at exact boundary with cache",
				accessEnd:            2000,
				timestamp:            1701,
				cacheDurationSeconds: 300,
				want:                 true, // 1701 + 300 = 2001 >= 2000
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				stake := &StakeInfo{AccessEnd: tt.accessEnd}
				if got := stake.HasExpired(tt.timestamp, tt.cacheDurationSeconds); got != tt.want {
					t.Errorf("HasExpired() = %v, want %v", got, tt.want)
				}
			})
		}
	})
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
				EVM: map[string]RateConfig{
					"5000": {QPM: uint64Ptr(1)},
				},
			},
			chainName: "EVM",
			want: []ConversionTranche{
				{RatePerSecond: 0, RatePerMinute: 1, Tranche: 5000},
			},
			wantError: false,
		},
		{
			name: "EVM chain with multiple tranches",
			table: ConversionTable{
				EVM: map[string]RateConfig{
					"5000":   {QPM: uint64Ptr(1)},
					"50000":  {QPS: uint64Ptr(1), QPM: uint64Ptr(60)},
					"500000": {QPS: uint64Ptr(10), QPM: uint64Ptr(600)},
				},
			},
			chainName: "EVM",
			want: []ConversionTranche{
				{RatePerSecond: 0, RatePerMinute: 1, Tranche: 5000},
				{RatePerSecond: 1, RatePerMinute: 60, Tranche: 50000},
				{RatePerSecond: 10, RatePerMinute: 600, Tranche: 500000},
			},
			wantError: false,
		},
		{
			name: "Solana chain",
			table: ConversionTable{
				Solana: map[string]RateConfig{
					"12500":  {QPM: uint64Ptr(1)},
					"125000": {QPS: uint64Ptr(1), QPM: uint64Ptr(60)},
				},
			},
			chainName: "Solana",
			want: []ConversionTranche{
				{RatePerSecond: 0, RatePerMinute: 1, Tranche: 12500},
				{RatePerSecond: 1, RatePerMinute: 60, Tranche: 125000},
			},
			wantError: false,
		},
		{
			name: "unknown chain",
			table: ConversionTable{
				EVM: map[string]RateConfig{
					"5000": {QPM: uint64Ptr(1)},
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
				EVM: map[string]RateConfig{
					"abc": {QPM: uint64Ptr(1)},
				},
			},
			chainName: "EVM",
			wantError: true,
		},
		{
			name: "no rate specified",
			table: ConversionTable{
				EVM: map[string]RateConfig{
					"5000": {},
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
				if got[i].RatePerSecond != tt.want[i].RatePerSecond {
					t.Errorf("GetTranchesByChain() tranche[%d].RatePerSecond = %d, want %d", i, got[i].RatePerSecond, tt.want[i].RatePerSecond)
				}
				if got[i].RatePerMinute != tt.want[i].RatePerMinute {
					t.Errorf("GetTranchesByChain() tranche[%d].RatePerMinute = %d, want %d", i, got[i].RatePerMinute, tt.want[i].RatePerMinute)
				}
				if got[i].Tranche != tt.want[i].Tranche {
					t.Errorf("GetTranchesByChain() tranche[%d].Tranche = %d, want %d", i, got[i].Tranche, tt.want[i].Tranche)
				}
			}
		})
	}
}
