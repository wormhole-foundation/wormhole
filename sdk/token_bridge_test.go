package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestEnvironment_String(t *testing.T) {
	tests := []struct {
		name string
		env  Environment
		want string
	}{
		{
			name: "EnvMainNet",
			env:  EnvMainNet,
			want: "prod",
		},
		{
			name: "EnvTestNet",
			env:  EnvTestNet,
			want: "test",
		},
		{
			name: "EnvDevNet",
			env:  EnvDevNet,
			want: "dev",
		},
		{
			name: "EnvGoTest",
			env:  EnvGoTest,
			want: "unit-test",
		},
		{
			name: "EnvAccountantMock",
			env:  EnvAccountantMock,
			want: "accountant-mock",
		},
		{
			name: "Unknown environment (should default to dev)",
			env:  Environment(99),
			want: "dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.env.String()
			if got != tt.want {
				t.Errorf("Environment.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvironmentFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Environment
	}{
		{
			name:  "prod",
			input: "prod",
			want:  EnvMainNet,
		},
		{
			name:  "test",
			input: "test",
			want:  EnvTestNet,
		},
		{
			name:  "dev",
			input: "dev",
			want:  EnvDevNet,
		},
		{
			name:  "unit-test",
			input: "unit-test",
			want:  EnvGoTest,
		},
		{
			name:  "accountant-mock",
			input: "accountant-mock",
			want:  EnvAccountantMock,
		},
		{
			name:  "unknown (should default to dev)",
			input: "unknown",
			want:  EnvDevNet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnvironmentFromString(tt.input)
			if got != tt.want {
				t.Errorf("EnvironmentFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnvironment_RoundTrip(t *testing.T) {
	// Test that String() and EnvironmentFromString() are inverse operations
	environments := []Environment{
		EnvMainNet,
		EnvTestNet,
		EnvDevNet,
		EnvGoTest,
		EnvAccountantMock,
	}

	for _, env := range environments {
		t.Run(env.String(), func(t *testing.T) {
			str := env.String()
			roundTrip := EnvironmentFromString(str)
			if roundTrip != env {
				t.Errorf("Round trip failed: %v -> %q -> %v", env, str, roundTrip)
			}
		})
	}
}

func TestEmitterType_String(t *testing.T) {
	tests := []struct {
		name string
		et   EmitterType
		want string
	}{
		{name: "unset", et: EmitterTypeUnset, want: "unset"},
		{name: "Core", et: EmitterCoreBridge, want: "Core"},
		{name: "TokenBridge", et: EmitterTokenBridge, want: "TokenBridge"},
		{name: "NFTBridge", et: EmitterNFTBridge, want: "NFTBridge"},
		{name: "unknown", et: EmitterType(99), want: "unknown emitter type: 99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.et.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetTokenBridgeEmitters(t *testing.T) {
	tests := []struct {
		name    string
		env     Environment
		wantLen int
	}{
		{name: "MainNet", env: EnvMainNet, wantLen: len(KnownTokenbridgeEmitters)},
		{name: "TestNet", env: EnvTestNet, wantLen: len(KnownTestnetTokenbridgeEmitters)},
		{name: "DevNet", env: EnvDevNet, wantLen: len(KnownDevnetTokenbridgeEmitters)},
		{name: "GoTest", env: EnvGoTest, wantLen: 0},
		{name: "AccountantMock", env: EnvAccountantMock, wantLen: 0},
		{name: "Unknown", env: Environment(99), wantLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTokenBridgeEmitters(tt.env)
			if tt.wantLen == 0 {
				assert.Nil(t, got)
			} else {
				assert.Len(t, got, tt.wantLen)
			}
		})
	}
}

func TestGetEmitterAddressForChain(t *testing.T) {
	tests := []struct {
		name        string
		chainID     vaa.ChainID
		emitterType EmitterType
		wantErr     bool
	}{
		{
			name:        "Ethereum token bridge",
			chainID:     vaa.ChainIDEthereum,
			emitterType: EmitterTokenBridge,
			wantErr:     false,
		},
		{
			name:        "Solana token bridge",
			chainID:     vaa.ChainIDSolana,
			emitterType: EmitterTokenBridge,
			wantErr:     false,
		},
		{
			name:        "Core bridge on Ethereum",
			chainID:     vaa.ChainIDEthereum,
			emitterType: EmitterCoreBridge,
			wantErr:     true,
		},
		{
			name:        "NFT bridge on Solana",
			chainID:     vaa.ChainIDSolana,
			emitterType: EmitterNFTBridge,
			wantErr:     false,
		},
		{
			name:        "Chain with no known emitters",
			chainID:     vaa.ChainID(9999),
			emitterType: EmitterTokenBridge,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := GetEmitterAddressForChain(tt.chainID, tt.emitterType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, vaa.Address{}, addr)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, vaa.Address{}, addr)
			}
		})
	}
}

func TestIsWTT(t *testing.T) {
	// Using real mainnet token bridge emitter addresses as hex strings
	const (
		ethTokenBridgeHex = "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"
		// #nosec G101 -- addresses, not secrets
		solanaTokenBridgeHex = "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5"
		wrongEmitterHex      = "0000000000000000000000000000000000000000000000000000000000000001"
	)

	tests := []struct {
		name           string
		emitterChain   vaa.ChainID
		emitterAddrHex string
		payload        []byte
		env            Environment
		want           bool
	}{
		{
			name:           "happy path - valid WTT from Ethereum",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01}, // Transfer payload type
			env:            EnvMainNet,
			want:           true,
		},
		{
			name:           "happy path - valid WTT from Solana with payload type 3",
			emitterChain:   vaa.ChainIDSolana,
			emitterAddrHex: solanaTokenBridgeHex,
			payload:        []byte{0x03}, // Transfer with payload type
			env:            EnvMainNet,
			want:           true,
		},
		{
			name:           "failure - wrong payload type",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x02}, // Not a transfer payload
			env:            EnvMainNet,
			want:           false,
		},
		{
			name:           "failure - chain without token bridge",
			emitterChain:   vaa.ChainIDOsmosis,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01},
			env:            EnvMainNet,
			want:           false,
		},
		{
			name:           "failure - emitter address doesn't match token bridge",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: wrongEmitterHex,
			payload:        []byte{0x01},
			env:            EnvMainNet,
			want:           false,
		},
		{
			name:           "failure - test environment (EnvGoTest)",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01},
			env:            EnvGoTest,
			want:           false,
		},
		{
			name:           "failure - mock environment (EnvAccountantMock)",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{0x01},
			env:            EnvAccountantMock,
			want:           false,
		},
		{
			name:           "failure - empty payload",
			emitterChain:   vaa.ChainIDEthereum,
			emitterAddrHex: ethTokenBridgeHex,
			payload:        []byte{},
			env:            EnvMainNet,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emitterAddr, err := vaa.StringToAddress(tt.emitterAddrHex)
			require.NoError(t, err)

			v := &vaa.VAA{
				EmitterChain:   tt.emitterChain,
				EmitterAddress: emitterAddr,
				Payload:        tt.payload,
			}

			got := IsWTT(v, tt.env)
			assert.Equal(t, tt.want, got)
		})
	}
}
