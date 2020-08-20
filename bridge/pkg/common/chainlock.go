package common

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type ChainLock struct {
	TxHash    common.Hash
	Timestamp time.Time
	
	Nonce uint32

	SourceAddress vaa.Address
	TargetAddress vaa.Address

	SourceChain vaa.ChainID
	TargetChain vaa.ChainID

	TokenChain   vaa.ChainID
	TokenAddress vaa.Address

	Amount *big.Int
}
