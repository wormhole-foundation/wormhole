package common

import (
	"fmt"
	"strings"
)

type Environment string

const (
	MainNet        Environment = "prod"
	UnsafeDevNet   Environment = "dev"  // local devnet; Keys are deterministic and many security controls are disabled
	TestNet        Environment = "test" // public testnet (needs to be reliable, but run with less Guardians and faster finality)
	GoTest         Environment = "unit-test"
	AccountantMock Environment = "accountant-mock" // Used for mocking accountant with a Wormchain connection
)

// ParseEnvironment parses a string into the corresponding Environment value, allowing various reasonable variations.
func ParseEnvironment(str string) (Environment, error) {
	str = strings.ToLower(str)
	if str == "prod" || str == "mainnet" {
		return MainNet, nil
	}
	if str == "test" || str == "testnet" {
		return TestNet, nil
	}
	if str == "dev" || str == "devnet" || str == "unsafedevnet" {
		return UnsafeDevNet, nil
	}
	if str == "unit-test" || str == "gotest" {
		return GoTest, nil
	}
	if str == "accountant-mock" || str == "accountantmock" {
		return AccountantMock, nil
	}
	return UnsafeDevNet, fmt.Errorf("invalid environment string: %s", str)
}
