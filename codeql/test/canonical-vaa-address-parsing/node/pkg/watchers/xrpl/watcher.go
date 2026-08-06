package xrpl

import "codeql/canonicalvaaaddressparsing/sdk/vaa"

func CoreEmitterAccount(account [20]byte) vaa.Address {
	emitter := vaa.Address{}
	copy(emitter[12:], account[:])
	return emitter
}
