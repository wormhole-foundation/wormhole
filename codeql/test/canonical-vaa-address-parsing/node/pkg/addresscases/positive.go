package addresscases

import (
	"encoding/hex"

	"codeql/canonicalvaaaddressparsing/node/pkg/common"
	"codeql/canonicalvaaaddressparsing/node/pkg/db"
	"codeql/canonicalvaaaddressparsing/sdk/vaa"
)

func DirectStringConversion(s string) common.MessagePublication {
	addr := vaa.Address([]byte(s))
	return common.MessagePublication{EmitterAddress: addr}
}

func DirectByteConversion(b []byte) common.MessagePublication {
	addr := vaa.Address(b)
	return common.MessagePublication{EmitterAddress: addr}
}

func ManualCopy(req string) db.VAAID {
	decoded, _ := hex.DecodeString(req)
	addr := vaa.Address{}
	copy(addr[:], decoded)
	return db.VAAID{EmitterAddress: addr}
}

func UnsafeHelperUse(s string) common.MessagePublication {
	addr, _ := ParseEmitterAddress(s)
	return common.MessagePublication{EmitterAddress: addr}
}

func ParseEmitterAddress(s string) (vaa.Address, error) {
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return vaa.Address{}, err
	}
	addr := vaa.Address{}
	copy(addr[:], decoded)
	return addr, nil
}

func StringViaBytesHelperUse(s string) common.MessagePublication {
	addr, _ := StringViaBytesAddress(s)
	return common.MessagePublication{EmitterAddress: addr}
}

func StringViaBytesAddress(s string) (vaa.Address, error) {
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return vaa.Address{}, err
	}
	return vaa.BytesToAddress(decoded)
}

func ManualWholeIDComponent(id string, component string) db.VAAID {
	_ = id
	return db.VAAID{EmitterAddress: vaa.Address([]byte(component))}
}
