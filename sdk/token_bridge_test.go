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
			emitterChain:   vaa.ChainIDTerra,
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
