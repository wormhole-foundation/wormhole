package vaa

// Addresses
const (
	// AddressBytesLen is the size in bytes of the Wormhole normalized address format.
	// It is the protocol-level 32-byte address representation, not the native address size of any particular chain.
	AddressBytesLen = 32
	// AddressHexLen is the canonical hex string length of the Wormhole normalized address format.
	// It corresponds to the protocol-level 32-byte address representation, not the native address size of any particular chain.
	AddressHexLen = AddressBytesLen * 2
)

// VAA format
const (
	// e.g. for `<chain-id>/<emitter-addr/<seq-num>`
	VAAIDPartsLen = 3
)
