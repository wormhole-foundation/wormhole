package types

import "github.com/ethereum/go-ethereum/common"

func (gs GuardianSet) KeysAsAddresses() (addresses []common.Address) {
	for _, key := range gs.Keys {
		address := common.Address{}
		copy(address[:], key)
		addresses = append(addresses, address)
	}

	return
}
