package addresscases

import (
	"strings"

	"codeql/canonicalvaaaddressparsing/node/pkg/common"
	"codeql/canonicalvaaaddressparsing/node/pkg/db"
	"codeql/canonicalvaaaddressparsing/sdk/vaa"
)

func CanonicalString(s string) (common.MessagePublication, error) {
	addr, err := vaa.StringToAddress(s)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{EmitterAddress: addr}, nil
}

func CanonicalBytes(b []byte) (common.MessagePublication, error) {
	addr, err := vaa.BytesToAddress(b)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{EmitterAddress: addr}, nil
}

func CanonicalWrapper(s string) (common.MessagePublication, error) {
	addr, err := WrappedStringToAddress(s)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{EmitterAddress: addr}, nil
}

func WrappedStringToAddress(s string) (vaa.Address, error) {
	return vaa.StringToAddress(s)
}

func UnrelatedStringBytesWrapper(s string, b []byte) (common.MessagePublication, error) {
	_ = s
	addr, err := BytesAddressWithUnrelatedString(s, b)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{EmitterAddress: addr}, nil
}

func BytesAddressWithUnrelatedString(s string, b []byte) (vaa.Address, error) {
	_ = s
	return vaa.BytesToAddress(b)
}

func TypedPropagation(msg common.MessagePublication) db.VAAID {
	return db.VAAID{EmitterAddress: msg.EmitterAddress}
}

func VaaFieldPropagation(v vaa.VAA) db.VAAID {
	return db.VAAID{EmitterAddress: v.EmitterAddress}
}

func WholeIDParser(id string) (*db.VAAID, error) {
	return db.VaaIDFromString(id)
}

func SplitWholeVaaIDOwnedBySibling(vaaID string) db.VAAID {
	parts := strings.Split(vaaID, "/")
	return db.VAAID{EmitterAddress: vaa.Address([]byte(parts[1]))}
}

func HashNormalization(txHash string) (vaa.Hash, error) {
	return vaa.StringToHash(txHash)
}
