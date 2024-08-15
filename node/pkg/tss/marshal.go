package tss

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"

	"github.com/yossigi/tss-lib/v2/crypto"
	"github.com/yossigi/tss-lib/v2/tss"
)

func marshalEcdsaSecretkey(key *ecdsa.PrivateKey) []byte {
	return key.D.Bytes()
}

func unmarshalEcdsaSecretKey(bz []byte) *ecdsa.PrivateKey {
	// TODO: I straggled with unmarshalling ecdh.PrivateKey, as a result I resorted to "unsafe" and deprecated method:
	x, y := tss.S256().ScalarBaseMult(bz)
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: tss.S256(),
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(bz),
	}
}

var errInvalidPublicKey = errors.New("invalid public key")

func unmarshalEcdsaPublickey(curve elliptic.Curve, bz []byte) (*ecdsa.PublicKey, error) {
	pnt := &crypto.ECPoint{}
	if err := pnt.GobDecode(bz); err != nil {
		return nil, err
	}

	if !curve.IsOnCurve(pnt.X(), pnt.Y()) {
		return nil, errInvalidPublicKey
	}

	pnt.SetCurve(curve)

	return pnt.ToECDSAPubKey(), nil
}

func marshalEcdsaPublickey(pk *ecdsa.PublicKey) ([]byte, error) {
	pnt, err := crypto.NewECPoint(pk.Curve, pk.X, pk.Y)
	if err != nil {
		return nil, err
	}
	return pnt.GobEncode()
}
