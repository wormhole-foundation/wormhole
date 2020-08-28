package common

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type ChainLock struct {
	TxHash    common.Hash // TODO: rename to identifier? on Solana, this isn't actually the tx hash
	Timestamp time.Time

	Nonce uint32

	SourceAddress vaa.Address
	TargetAddress vaa.Address

	SourceChain vaa.ChainID
	TargetChain vaa.ChainID

	TokenChain    vaa.ChainID
	TokenAddress  vaa.Address
	TokenDecimals uint8

	Amount *big.Int
}
