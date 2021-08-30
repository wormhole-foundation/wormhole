package devnet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	mathrand "math/rand"
)

// DeterministicEcdsaKeyByIndex generates a deterministic ecdsa.PrivateKey from a given index.
func DeterministicEcdsaKeyByIndex(c elliptic.Curve, idx uint64) *ecdsa.PrivateKey {
	// use 555 as offset to deterministically generate key 0 to match vaa-test such that
	// we generate the same key.
	r := mathrand.New(mathrand.NewSource(int64(555 + idx)))
	key, err := ecdsa.GenerateKey(c, r)
	if err != nil {
		panic(err)
	}

	return key
}
