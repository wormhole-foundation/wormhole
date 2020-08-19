package devnet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	mathrand "math/rand"

	"github.com/libp2p/go-libp2p-core/crypto"
)

// DeterministicEcdsaKeyByIndex generates a deterministic ecdsa.PrivateKey from a given index.
func DeterministicEcdsaKeyByIndex(c elliptic.Curve, idx uint64) *ecdsa.PrivateKey {
	buf := make([]byte, 200)
	binary.LittleEndian.PutUint64(buf, idx)

	worstRNG := bytes.NewBuffer(buf)

	key, err := ecdsa.GenerateKey(c, bytes.NewReader(worstRNG.Bytes()))
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
