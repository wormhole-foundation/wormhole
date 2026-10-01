package sui

import (
	"codeql/canonicalvaaaddressparsing/node/pkg/common"
	"codeql/canonicalvaaaddressparsing/sdk/vaa"
)

type Event struct {
	Sender [32]byte
}

func TypedSender(event Event) common.MessagePublication {
	addr := vaa.Address(event.Sender)
	return common.MessagePublication{EmitterAddress: addr}
}
