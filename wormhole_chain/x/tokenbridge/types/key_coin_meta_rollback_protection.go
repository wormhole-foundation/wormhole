package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// CoinMetaRollbackProtectionKeyPrefix is the prefix to retrieve all CoinMetaRollbackProtection
	CoinMetaRollbackProtectionKeyPrefix = "CoinMetaRollbackProtection/value/"
)

// CoinMetaRollbackProtectionKey returns the store key to retrieve a CoinMetaRollbackProtection from the index fields
func CoinMetaRollbackProtectionKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
