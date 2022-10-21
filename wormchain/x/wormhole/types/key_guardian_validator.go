package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// GuardianValidatorKeyPrefix is the prefix to retrieve all GuardianValidator
	GuardianValidatorKeyPrefix = "GuardianValidator/value/"
)

// GuardianValidatorKey returns the store key to retrieve a GuardianValidator from the index fields
func GuardianValidatorKey(
	guardianKey []byte,
) []byte {
	var key []byte

	key = append(key, guardianKey...)
	key = append(key, []byte("/")...)

	return key
}
