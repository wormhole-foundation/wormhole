package celo

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/celo-org/celo-blockchain/common"
)

// PadAddress creates 32-byte VAA.Address from 20-byte Celo addresses by adding 12 0-bytes at the left
func PadAddress(address common.Address) vaa.Address {
	paddedAddress := common.LeftPadBytes(address[:], 32)

	addr := vaa.Address{}
	copy(addr[:], paddedAddress)

	return addr
}
