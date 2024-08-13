package tss

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"math/big"

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

func unmarshalEcdsaPublickey(curve elliptic.Curve, bz []byte) *ecdsa.PublicKey {
	x, y := elliptic.UnmarshalCompressed(curve, bz)
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
}

func marshalEcdsaPublickey(pk *ecdsa.PublicKey) []byte {
	return elliptic.MarshalCompressed(pk.Curve, pk.X, pk.Y)
}
