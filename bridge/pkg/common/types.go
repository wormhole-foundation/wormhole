package common

import (
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"math/big"
)

type (
	BridgeWatcher interface {
		WatchLockups(events chan *ChainLock) error
	}

	ChainLock struct {
		SourceAddress vaa.Address
		TargetAddress vaa.Address

		SourceChain vaa.ChainID
		TargetChain vaa.ChainID

		TokenChain   vaa.ChainID
		TokenAddress vaa.Address

		Amount *big.Int
	}
)
