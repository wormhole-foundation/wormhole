package common

import (
	"crypto/sha256"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type ChainLock struct {
	TxHash common.Hash

	SourceAddress vaa.Address
	TargetAddress vaa.Address

	SourceChain vaa.ChainID
	TargetChain vaa.ChainID

	TokenChain   vaa.ChainID
	TokenAddress vaa.Address

	Amount *big.Int
}

// Hash returns a deterministic hash of the given ChainLock, meant to be used
// as primary key for the distributed round of signing.
//
// TODO: the ETH contract is missing a nonce, so it's not yet deterministic
func (l *ChainLock) Hash() []byte {

	// TODO: json.Marshal being deterministic is an implementation detail - what guarantees do we need?
	// We do not necessarily need stable serialization across releases, but they do need to be unique.

	b, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}

	h := sha256.New()
	h.Write(b)

	return h.Sum(nil)
}
