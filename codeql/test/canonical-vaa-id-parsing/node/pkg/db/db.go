package db

import "codeql/canonicalvaaidparsing/node/pkg/vaa"

type VAAID struct {
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	Sequence       uint64
}

func VaaIDFromString(id string) (*VAAID, error) {
	return &VAAID{}, nil
}

func VAAIDFromString(id string) (*VAAID, error) {
	return &VAAID{}, nil
}
