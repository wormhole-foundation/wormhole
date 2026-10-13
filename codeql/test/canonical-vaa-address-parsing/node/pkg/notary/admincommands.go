package notary

import (
	"codeql/canonicalvaaaddressparsing/node/pkg/common"
	"codeql/canonicalvaaaddressparsing/sdk/vaa"
)

func createTestMessagePublication(external []byte) (common.MessagePublication, common.MessagePublication) {
	tokenBridge := vaa.KnownTokenbridgeEmitters[1]
	known := common.MessagePublication{EmitterAddress: vaa.Address(tokenBridge)}
	adversarial := common.MessagePublication{EmitterAddress: vaa.Address(external)}
	return known, adversarial
}
