package types

import "fmt"

const (
	// ModuleName defines the wormhole middleware name
	// wormhole prefix is already used, so using wormchain-mw
	ModuleName = "wormchain-mw"

	StoreKey = ModuleName
)

func TransposedDataKey(channelID, portID string, sequence uint64) []byte {
	return []byte(fmt.Sprintf("%s/%s/%d", channelID, portID, sequence))
}
