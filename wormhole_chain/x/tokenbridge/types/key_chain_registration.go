package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ChainRegistrationKeyPrefix is the prefix to retrieve all ChainRegistration
	ChainRegistrationKeyPrefix = "ChainRegistration/value/"
)

// ChainRegistrationKey returns the store key to retrieve a ChainRegistration from the index fields
func ChainRegistrationKey(
	index uint32,
) []byte {
	key := make([]byte, 4)
	binary.BigEndian.PutUint32(key, index)
	key = append(key, []byte("/")...)

	return key
}
