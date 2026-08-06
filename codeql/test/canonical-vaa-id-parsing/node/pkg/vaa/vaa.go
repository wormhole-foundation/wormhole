package vaa

type ChainID uint16

type Address [32]byte

func StringToAddress(s string) (Address, error) {
	return Address{}, nil
}

func VAAIDFromString(id string) (*VAAID, error) {
	return &VAAID{}, nil
}

type VAAID struct {
	EmitterChain   ChainID
	EmitterAddress Address
	Sequence       uint64
}
