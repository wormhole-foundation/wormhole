package evm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// PadAddress creates 32-byte VAA.Address from 20-byte Ethereum addresses by adding 12 0-bytes at the left
func PadAddress(address common.Address) vaa.Address {
	paddedAddress := common.LeftPadBytes(address[:], 32)

	addr := vaa.Address{}
	copy(addr[:], paddedAddress)

	return addr
}
