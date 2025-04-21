package querystaking

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type StakeAndSigner struct {
	StakeInfo *StakeInfo     `abi:"StakeInfo"`
	Signer    common.Address `abi:"Signer"`
}

type StakeInfo struct {
	Amount               *uint256.Int `abi:"uint256"`
	ConversionTableIndex *uint256.Int `abi:"uint256"`
	LockupEnd            uint64       `abi:"uint64"`
	AccessEnd            uint64       `abi:"uint64"`
}

var ErrInvalidLength = errors.New("invalid length")

func ParseStakeInfo(data []byte) (*StakeInfo, error) {
	empty := &StakeInfo{}

	if len(data) != 32*4 {
		return nil, fmt.Errorf("invalid length: got %d want %d", len(data), 32*4)
	}
	empty.Amount = uint256.NewInt(0).SetBytes(data[0:32])
	empty.ConversionTableIndex = uint256.NewInt(0).SetBytes(data[32:64])
	tmp := uint256.NewInt(0)
	empty.LockupEnd = tmp.SetBytes(data[64:72]).Uint64()
	empty.AccessEnd = tmp.SetBytes(data[72:80]).Uint64()
	return empty, nil
}
