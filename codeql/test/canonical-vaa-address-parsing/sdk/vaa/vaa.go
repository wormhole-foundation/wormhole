package vaa

type Address [32]byte
type Hash [32]byte

type VAA struct {
	EmitterAddress Address
}

var KnownTokenbridgeEmitters = map[uint16][]byte{1: []byte{1, 2, 3}}

func StringToAddress(s string) (Address, error) {
	return Address([]byte(s)), nil
}

func BytesToAddress(b []byte) (Address, error) {
	return Address(b), nil
}

func StringToHash(s string) (Hash, error) {
	return Hash([]byte(s)), nil
}
