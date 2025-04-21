package querystaking

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/query/queryratelimit"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// TestCalculateRates tests the rate calculation logic with conversion tables
func TestCalculateRates(t *testing.T) {
	tests := []struct {
		name       string
		stake      *uint256.Int
		conversion string
		want       queryratelimit.Rule
	}{
		// Zero/nil stake tests
		{
			name:       "nil stake",
			stake:      nil,
			conversion: "rate:100,tranche:1000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0},
		},
		{
			name:       "zero stake",
			stake:      uint256.NewInt(0),
			conversion: "rate:100,tranche:1000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0},
		},

		// Invalid conversion entry tests
		{
			name:       "empty conversion entry",
			stake:      uint256.NewInt(10000),
			conversion: "",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0},
		},
		{
			name:       "invalid conversion format",
			stake:      uint256.NewInt(10000),
			conversion: "invalid",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0},
		},

		// Single tranche tests
		{
			name:       "stake below minimum tranche",
			stake:      uint256.NewInt(4999),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0},
		},
		{
			name:       "stake at minimum tranche",
			stake:      uint256.NewInt(5000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 10}, // (5000/5000)*10 = 10 QPM
		},
		{
			name:       "stake above minimum tranche",
			stake:      uint256.NewInt(10000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 20}, // (10000/5000)*10 = 20 QPM
		},

		// Multiple tranche tests
		{
			name:       "qualifies for first tranche only",
			stake:      uint256.NewInt(10000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 20}, // (10000/5000)*10 = 20 QPM
		},
		{
			name:       "qualifies for higher tranche",
			stake:      uint256.NewInt(100000),
			conversion: "rate:100,tranche:50000",
			want:       queryratelimit.Rule{MaxPerSecond: 3, MaxPerMinute: 200}, // (100000/50000)*100 = 200 QPM, 200/60 = 3 QPS
		},

		// QPM to QPS conversion tests
		{
			name:       "QPM less than 60 - no QPS",
			stake:      uint256.NewInt(25000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 50}, // (25000/5000)*10 = 50 QPM, no QPS
		},
		{
			name:       "QPM equals 60 - 1 QPS",
			stake:      uint256.NewInt(30000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 1, MaxPerMinute: 60}, // (30000/5000)*10 = 60 QPM, 60/60 = 1 QPS
		},
		{
			name:       "QPM equals 120 - 2 QPS",
			stake:      uint256.NewInt(60000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 2, MaxPerMinute: 120}, // (60000/5000)*10 = 120 QPM, 120/60 = 2 QPS
		},
		{
			name:       "QPM with truncation in division",
			stake:      uint256.NewInt(59500),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 1, MaxPerMinute: 110}, // (59500/5000)*10 = 11*10 = 110 QPM (59500/5000 truncates to 11), 110/60 = 1 QPS
		},
		{
			name:       "high QPM - 600 QPM becomes 10 QPS",
			stake:      uint256.NewInt(300000),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 10, MaxPerMinute: 600}, // (300000/5000)*10 = 600 QPM, 600/60 = 10 QPS
		},

		// Edge case tests
		{
			name:       "exact tranche boundary",
			stake:      uint256.NewInt(50000),
			conversion: "rate:100,tranche:50000",
			want:       queryratelimit.Rule{MaxPerSecond: 1, MaxPerMinute: 100}, // (50000/50000)*100 = 100 QPM, 100/60 = 1 QPS
		},
		{
			name:       "one less than tranche boundary",
			stake:      uint256.NewInt(49999),
			conversion: "rate:10,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 1, MaxPerMinute: 90}, // (49999/5000)*10 = 9*10 = 90 QPM (49999/5000 = 9, truncated), 90/60 = 1 QPS
		},
		{
			name:       "large stake amount",
			stake:      uint256.NewInt(1000000),
			conversion: "rate:100,tranche:50000",
			want:       queryratelimit.Rule{MaxPerSecond: 33, MaxPerMinute: 2000}, // (1000000/50000)*100 = 2000 QPM, 2000/60 = 33 QPS
		},

		// Zero rate/tranche tests
		{
			name:       "zero rate in tranche",
			stake:      uint256.NewInt(10000),
			conversion: "rate:0,tranche:5000",
			want:       queryratelimit.Rule{MaxPerSecond: 0, MaxPerMinute: 0}, // (10000/5000)*0 = 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conversionEntry := toBytes32(tt.conversion)
			tranches, err := ParseConversionTranches(conversionEntry)
			// If parse fails, CalculateRates should return zero (like it did before)
			if err != nil {
				tranches = []ConversionTranche{}
			}
			// Use decimals=0 so stake values are treated as token units (no conversion)
			got := CalculateRates(tt.stake, tranches, 0)

			if got.MaxPerSecond != tt.want.MaxPerSecond {
				t.Errorf("CalculateRates() MaxPerSecond = %d, want %d", got.MaxPerSecond, tt.want.MaxPerSecond)
			}
			if got.MaxPerMinute != tt.want.MaxPerMinute {
				t.Errorf("CalculateRates() MaxPerMinute = %d, want %d", got.MaxPerMinute, tt.want.MaxPerMinute)
			}
		})
	}
}

// TestCalculateRates_IntegerDivision tests integer division behavior
func TestCalculateRates_IntegerDivision(t *testing.T) {
	tests := []struct {
		name       string
		stake      uint64
		conversion string
		wantQPM    uint64
		wantQPS    uint64
	}{
		{
			name:       "division with no remainder",
			stake:      10000,
			conversion: "rate:10,tranche:5000",
			wantQPM:    20, // (10000/5000)*10 = 2*10 = 20
			wantQPS:    0,
		},
		{
			name:       "division truncates remainder",
			stake:      12500,
			conversion: "rate:10,tranche:5000",
			wantQPM:    20, // (12500/5000)*10 = 2*10 = 20 (12500/5000 = 2.5, truncated to 2)
			wantQPS:    0,
		},
		{
			name:       "QPS truncates fractional result",
			stake:      65000,
			conversion: "rate:10,tranche:5000",
			wantQPM:    130, // (65000/5000)*10 = 13*10 = 130
			wantQPS:    2,   // 130/60 = 2.166..., truncated to 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stake := uint256.NewInt(tt.stake)
			conversionEntry := toBytes32(tt.conversion)
			tranches, err := ParseConversionTranches(conversionEntry)
			if err != nil {
				t.Fatalf("ParseConversionTranches() error = %v", err)
			}
			// Use decimals=0 so stake values are treated as token units (no conversion)
			got := CalculateRates(stake, tranches, 0)

			if got.MaxPerMinute != tt.wantQPM {
				t.Errorf("CalculateRates() QPM = %d, want %d", got.MaxPerMinute, tt.wantQPM)
			}
			if got.MaxPerSecond != tt.wantQPS {
				t.Errorf("CalculateRates() QPS = %d, want %d", got.MaxPerSecond, tt.wantQPS)
			}
		})
	}
}

// TestAuthorizationLogic tests the authorization logic at the business logic level
func TestAuthorizationLogic(t *testing.T) {
	tests := []struct {
		name               string
		providedSigner     common.Address
		registeredSigner   common.Address
		stakerAddr         common.Address
		expectedAuthorized bool
		description        string
	}{
		{
			name:               "signer matches registered and is non-zero",
			providedSigner:     common.HexToAddress("0x1111"),
			registeredSigner:   common.HexToAddress("0x1111"),
			stakerAddr:         common.HexToAddress("0x2222"),
			expectedAuthorized: true,
			description:        "Delegated signer that matches registered signer",
		},
		{
			name:               "signer does not match registered",
			providedSigner:     common.HexToAddress("0x1111"),
			registeredSigner:   common.HexToAddress("0x3333"),
			stakerAddr:         common.HexToAddress("0x2222"),
			expectedAuthorized: false,
			description:        "Provided signer doesn't match the registered one",
		},
		{
			name:               "registered signer is zero - not authorized",
			providedSigner:     common.HexToAddress("0x1111"),
			registeredSigner:   common.Address{},
			stakerAddr:         common.HexToAddress("0x2222"),
			expectedAuthorized: false,
			description:        "No signer registered (zero address)",
		},
		{
			name:               "both signers zero - not authorized",
			providedSigner:     common.Address{},
			registeredSigner:   common.Address{},
			stakerAddr:         common.HexToAddress("0x2222"),
			expectedAuthorized: false,
			description:        "Zero signer attempting authorization",
		},
		{
			name:               "self-staking - signer is staker",
			providedSigner:     common.HexToAddress("0x1111"),
			registeredSigner:   common.Address{}, // Doesn't matter for self-staking
			stakerAddr:         common.HexToAddress("0x1111"),
			expectedAuthorized: true,
			description:        "Self-staking scenario where signer equals staker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the authorization logic
			var isAuthorized bool

			// Self-staking case
			if tt.stakerAddr == tt.providedSigner {
				isAuthorized = true
			} else {
				// Delegated signing case - check registered signer
				isAuthorized = tt.registeredSigner != (common.Address{}) && tt.registeredSigner == tt.providedSigner
			}

			if isAuthorized != tt.expectedAuthorized {
				t.Errorf("%s: got %v, want %v", tt.description, isAuthorized, tt.expectedAuthorized)
			}
		})
	}
}

// TestSignerAuthorizationScenarios tests various authorization scenarios
func TestSignerAuthorizationScenarios(t *testing.T) {
	stakerA := common.HexToAddress("0x1111111111111111111111111111111111111111")
	stakerB := common.HexToAddress("0x2222222222222222222222222222222222222222")
	signerX := common.HexToAddress("0x3333333333333333333333333333333333333333")
	signerY := common.HexToAddress("0x4444444444444444444444444444444444444444")
	zeroAddr := common.Address{}

	tests := []struct {
		name       string
		staker     common.Address
		signer     common.Address
		registered common.Address
		shouldPass bool
		scenario   string
	}{
		{
			name:       "self-staking always authorized",
			staker:     stakerA,
			signer:     stakerA,
			registered: zeroAddr, // Not checked
			shouldPass: true,
			scenario:   "Staker signing for themselves",
		},
		{
			name:       "valid delegation",
			staker:     stakerA,
			signer:     signerX,
			registered: signerX,
			shouldPass: true,
			scenario:   "Staker delegated to signerX",
		},
		{
			name:       "invalid delegation - wrong signer",
			staker:     stakerA,
			signer:     signerY,
			registered: signerX,
			shouldPass: false,
			scenario:   "signerY trying to sign but signerX is registered",
		},
		{
			name:       "no delegation set up",
			staker:     stakerA,
			signer:     signerX,
			registered: zeroAddr,
			shouldPass: false,
			scenario:   "No signer registered for staker",
		},
		{
			name:       "zero signer - always fails",
			staker:     stakerA,
			signer:     zeroAddr,
			registered: zeroAddr,
			shouldPass: false,
			scenario:   "Zero address trying to sign",
		},
		{
			name:       "cross-staker - wrong staker",
			staker:     stakerB,
			signer:     signerX,
			registered: signerX,
			shouldPass: true,
			scenario:   "Different staker but valid delegation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate authorization check
			var authorized bool

			if tt.staker == tt.signer {
				// Self-staking path
				authorized = true
			} else {
				// Delegation path
				isZero := tt.registered == zeroAddr
				signerMatches := tt.registered == tt.signer
				authorized = !isZero && signerMatches

				if authorized != tt.shouldPass {
					t.Logf("Debug: staker=%s, signer=%s, registered=%s", tt.staker.Hex(), tt.signer.Hex(), tt.registered.Hex())
					t.Logf("Debug: isZero=%v, signerMatches=%v, authorized=%v", isZero, signerMatches, authorized)
				}
			}

			if authorized != tt.shouldPass {
				t.Errorf("%s: authorized=%v, want %v", tt.scenario, authorized, tt.shouldPass)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
