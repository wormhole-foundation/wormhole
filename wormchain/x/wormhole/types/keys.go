package types

const (
	// ModuleName defines the module name
	ModuleName = "wormhole"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for slashing
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_wormhole"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	GuardianSetKey      = "GuardianSet-value-"
	GuardianSetCountKey = "GuardianSet-count-"
)

const (
	ConfigKey = "Config-value-"
)

const (
	ConsensusGuardianSetIndexKey = "ConsensusGuardianSetIndex-value-"
)

const (
	ValidatorAllowlistKey         = "VAK"
	WasmInstantiateAllowlistKey   = "WasmInstiantiateAllowlist"
	IbcComposabilityMwContractKey = "IbcComposabilityMwContract"
)
