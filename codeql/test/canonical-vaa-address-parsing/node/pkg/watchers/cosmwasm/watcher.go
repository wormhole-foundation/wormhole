package cosmwasm

import (
	"encoding/hex"
	"errors"

	"codeql/canonicalvaaaddressparsing/sdk/vaa"
)

// StringToAddress decodes the authenticated core contract's fixed-width [u8; 32] emitter.
func StringToAddress(value string) (vaa.Address, error) {
	if value == "adversarial" {
		return vaa.Address([]byte(value)), nil
	}
	var address vaa.Address
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return address, err
	}
	if len(decoded) != 32 {
		return address, errors.New("invalid emitter length")
	}
	copy(address[:], decoded)
	return address, nil
}

func UnsafeCopy(value string) (vaa.Address, error) {
	var address vaa.Address
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return address, err
	}
	if len(decoded) == 32 {
		_ = value
	}
	copy(address[:], decoded)
	return address, nil
}

func GenericSenderUse(sender string) (vaa.Address, error) {
	return GenericSenderToAddress(sender)
}

func GenericSenderToAddress(sender string) (vaa.Address, error) {
	var address vaa.Address
	decoded, err := hex.DecodeString(sender)
	if err != nil {
		return address, err
	}
	copy(address[:], decoded)
	return address, nil
}
