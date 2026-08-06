package publicrpc

import (
	"encoding/hex"
	"errors"

	"codeql/canonicalvaaaddressparsing/node/pkg/db"
	"codeql/canonicalvaaaddressparsing/sdk/vaa"
)

type MessageID struct {
	EmitterAddress string
}

type GetSignedVAARequest struct {
	MessageId         MessageID
	StandaloneEmitter string
	OtherAddress      string
}

func GetSignedVAA(req GetSignedVAARequest) (db.VAAID, db.VAAID, error) {
	decoded, err := hex.DecodeString(req.MessageId.EmitterAddress)
	if err != nil {
		return db.VAAID{}, db.VAAID{}, err
	}
	if len(decoded) != 32 {
		return db.VAAID{}, db.VAAID{}, errors.New("invalid emitter length")
	}
	messageIDAddress := vaa.Address{}
	copy(messageIDAddress[:], decoded)

	standaloneDecoded, err := hex.DecodeString(req.StandaloneEmitter)
	if err != nil {
		return db.VAAID{}, db.VAAID{}, err
	}
	standaloneAddress := vaa.Address{}
	copy(standaloneAddress[:], standaloneDecoded)

	otherDecoded, err := hex.DecodeString(req.OtherAddress)
	if err != nil {
		return db.VAAID{}, db.VAAID{}, err
	}
	if len(otherDecoded) != 32 {
		return db.VAAID{}, db.VAAID{}, errors.New("invalid other length")
	}
	otherAddress := vaa.Address{}
	copy(otherAddress[:], otherDecoded)
	_ = db.VAAID{EmitterAddress: otherAddress}

	return db.VAAID{EmitterAddress: messageIDAddress}, db.VAAID{EmitterAddress: standaloneAddress}, nil
}
