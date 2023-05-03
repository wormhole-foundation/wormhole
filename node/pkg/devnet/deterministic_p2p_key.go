package devnet

import (
	mathrand "math/rand"

	"github.com/libp2p/go-libp2p/core/crypto"
)

// DeterministicP2PPrivKeyByIndex generates a deterministic libp2p crypto.PrivateKey from a given index.
func DeterministicP2PPrivKeyByIndex(idx int64) crypto.PrivKey {
	r := mathrand.New(mathrand.NewSource(idx)) //#nosec G404 testnet / devnet keys are public knowledge
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, -1, r)
	if err != nil {
		panic(err)
	}

	return priv
}
