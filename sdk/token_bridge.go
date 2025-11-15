package sdk

import (
	"bytes"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Environment represents the Wormhole network environment
type Environment uint8

const (
	EnvMainNet Environment = iota
	EnvTestNet
	EnvDevNet
	EnvGoTest
	EnvAccountantMock
)

// GetTokenBridgeEmitters returns the token bridge emitter map for the given environment.
// Returns nil for test/mock environments (EnvGoTest, EnvAccountantMock).
func GetTokenBridgeEmitters(env Environment) map[vaa.ChainID][]byte {
	switch env {
	case EnvMainNet:
		return KnownTokenbridgeEmitters
	case EnvTestNet:
		return KnownTestnetTokenbridgeEmitters
	case EnvDevNet:
		return KnownDevnetTokenbridgeEmitters
	case EnvGoTest, EnvAccountantMock:
		// Test and mock environments don't have real token bridge emitters
		return nil
	default:
		return nil
	}
}

// String returns the string representation of the Environment.
// The output corresponds to the input format used by EnvironmentFromString.
func (e Environment) String() string {
	switch e {
	case EnvMainNet:
		return "prod"
	case EnvTestNet:
		return "test"
	case EnvDevNet:
		return "dev"
	case EnvGoTest:
		return "unit-test"
	case EnvAccountantMock:
		return "accountant-mock"
	default:
		return "dev"
	}
}

// EnvironmentFromString converts a common.Environment string to sdk.Environment.
// This helper is useful for node code that needs to convert between the two types.
func EnvironmentFromString(env string) Environment {
	switch env {
	case "prod":
		return EnvMainNet
	case "test":
		return EnvTestNet
	case "dev":
		return EnvDevNet
	case "unit-test":
		return EnvGoTest
	case "accountant-mock":
		return EnvAccountantMock
	default:
		return EnvDevNet
	}
}

// IsWTT checks if the VAA represents a valid wrapped token transfer for a given environment.
// It verifies:
// 1. The payload is a transfer (payload type 1 or 3) via vaa.IsTransfer
// 2. The emitter is a known token bridge emitter for the specified environment
//
// This function validates WTTs with respect to an environment's known token bridge emitters.
// For a context-free check that only verifies the payload type, use vaa.IsTransfer instead.
//
// Returns false for test/mock environments (EnvGoTest, EnvAccountantMock) and if either check fails.
//
// Note: This function uses the same validation logic as MessagePublication.IsWTT.
func IsWTT(v *vaa.VAA, env Environment) bool {
	// Check if it's a transfer payload
	if !vaa.IsTransfer(v.Payload) {
		return false
	}

	// Get token bridge emitters for the environment
	tbEmitters := GetTokenBridgeEmitters(env)
	if tbEmitters == nil {
		return false
	}

	// Check if the emitter chain has a known token bridge
	tokenBridge, ok := tbEmitters[v.EmitterChain]
	if !ok {
		return false
	}

	// Make a defensive copy to prevent external mutation from affecting the comparison
	tokenBridgeCopy := make([]byte, len(tokenBridge))
	copy(tokenBridgeCopy, tokenBridge)

	// Check if the emitter address matches the token bridge
	return bytes.Equal(v.EmitterAddress.Bytes(), tokenBridgeCopy)
}
