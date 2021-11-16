package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EmitterAddress struct {
	bytes []byte
}

func (emitterAddress EmitterAddress) Bytes() []byte {
	return emitterAddress.bytes
}

func EmitterAddressFromBytes32(bytes []byte) (EmitterAddress, error) {
	if len(bytes) != 32 {
		return EmitterAddress{
			bytes: nil,
		}, fmt.Errorf("emitter must be 32 bytes long, was %d", len(bytes))
	}

	return EmitterAddress{bytes}, nil
}

func EmitterAddressFromAccAddress(addr sdk.AccAddress) EmitterAddress {
	bytes := addr.Bytes()
	// NOTE: this code could technically underflow, if the address is longer
	// than 32 bytes. We make the assumption here (which assumption appears
	// elsewhere in cosmos) that addresses are not longer than 32 bytes (in
	// fact, they will always be 20 bytes).
	zeros := make([]byte, 32-len(bytes))

	return EmitterAddress{
		bytes: append(zeros[:], bytes[:]...),
	}
}
