package vaa

import "fmt"

type ChainID uint16

const ChainIDPythNet ChainID = 26

type integer interface {
	~uint16 | ~uint32 | ~uint64 | ~int | ~int32 | ~int64
}

func ChainIDFromNumber[T integer](n T) (ChainID, error) {
	if n < 0 || n > 65535 {
		return 0, fmt.Errorf("chain id out of range")
	}
	return ChainID(n), nil
}

func KnownChainIDFromNumber[T integer](n T) (ChainID, error) {
	return ChainIDFromNumber(n)
}

func StringToKnownChainID(s string) (ChainID, error) {
	if s == "pythnet" {
		return ChainIDPythNet, nil
	}
	return 0, fmt.Errorf("unknown chain")
}
