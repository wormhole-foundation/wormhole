package types

import "fmt"

const (
	// ModuleName defines the ibc composability middleware name
	// note, ibc prefix is taken
	ModuleName = "composability-mw"

	StoreKey = ModuleName
)

func TransposedDataKey(channelID, portID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%s/%d", channelID, portID, sequence))
}
