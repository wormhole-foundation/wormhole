package stacks

const (
	// Size and buffer limits
	MaxClarityValueHexSize = 2 * 1024 * 1024 // 2MB in hex encoding (1MB decoded)
	MaxPayloadSize         = 8192            // Maximum Wormhole message payload size in bytes
	EmitterAddressSize     = 32              // Size of emitter address buffer in bytes
	TransactionIDSize      = 32              // Size of Stacks transaction ID in bytes (64 hex chars)
)

const (
	// Stacks protocol identifiers
	OkPrefixHex     = "0x07"    // Hex prefix for Clarity ok response
	OkPrefix        = "(ok "    // String prefix for transaction ok result
	NakamotoEpochID = "Epoch30" // Nakamoto upgrade epoch identifier
)
