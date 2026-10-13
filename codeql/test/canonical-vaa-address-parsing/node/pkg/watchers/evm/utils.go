package evm

import "codeql/canonicalvaaaddressparsing/sdk/vaa"

type Address [20]byte

func PadAddress(address Address) vaa.Address {
	paddedAddress := vaa.Address{}
	copy(paddedAddress[12:], address[:])
	return paddedAddress
}
