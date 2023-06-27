package params

const (
	// Name defines the application name of Wormchain.
	Name = "uworm"

	// BondDenom defines the native staking token denomination.
	BondDenom = "uworm"

	// DisplayDenom defines the name, symbol, and display value of the worm token.
	DisplayDenom = "WORM"

	// DefaultGasLimit - set to the same value as cosmos-sdk flags.DefaultGasLimit
	// this value is currently only used in tests.
	DefaultGasLimit = 200000
)
