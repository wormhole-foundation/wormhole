package devnet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	mathrand "math/rand"

	"github.com/libp2p/go-libp2p-core/crypto"
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

// DeterministicP2PPrivKeyByIndex generates a deterministic libp2p crypto.PrivateKey from a given index.
func DeterministicP2PPrivKeyByIndex(idx int64) crypto.PrivKey {
	r := mathrand.New(mathrand.NewSource(int64(idx)))
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, r)
	if err != nil {
		panic(err)
	}

	return priv
}
