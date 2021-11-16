package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ReplayProtectionKeyPrefix is the prefix to retrieve all ReplayProtection
	ReplayProtectionKeyPrefix = "ReplayProtection/value/"
)

// ReplayProtectionKey returns the store key to retrieve a ReplayProtection from the index fields
func ReplayProtectionKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
