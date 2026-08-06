package db

import "codeql/canonicalvaaaddressparsing/sdk/vaa"

type VAAID struct {
	EmitterAddress vaa.Address
}

func VaaIDFromString(id string) (*VAAID, error) {
	addr, err := vaa.StringToAddress(id)
	if err != nil {
		return nil, err
	}
	return &VAAID{EmitterAddress: addr}, nil
}
