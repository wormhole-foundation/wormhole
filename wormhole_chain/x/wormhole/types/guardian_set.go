package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

func (gs GuardianSet) KeysAsAddresses() (addresses []common.Address) {
	for _, key := range gs.Keys {
		addresses = append(addresses, common.BytesToAddress(key))
	}

	return
}

func (gs GuardianSet) ValidateBasic() error {
	for i, key := range gs.Keys {
		if len(key) != 20 {
			return fmt.Errorf("key [%d]: len %d != 20", i, len(key))
		}
	}

	if len(gs.Keys) == 0 {
		return fmt.Errorf("guardian set must not be empty")
	}

	if len(gs.Keys) > 255 {
		return fmt.Errorf("guardian set length must be <= 255, is %d", len(gs.Keys))
	}

	return nil
}
