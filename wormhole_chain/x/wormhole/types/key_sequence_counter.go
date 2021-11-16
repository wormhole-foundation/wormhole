package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SequenceCounterKeyPrefix is the prefix to retrieve all SequenceCounter
	SequenceCounterKeyPrefix = "SequenceCounter/value/"
)

// SequenceCounterKey returns the store key to retrieve a SequenceCounter from the index fields
func SequenceCounterKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
