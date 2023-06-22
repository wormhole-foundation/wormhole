package devnet

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	eth_crypto "github.com/ethereum/go-ethereum/crypto"
)

// InsecureDeterministicEcdsaKeyByIndex generates a deterministic ecdsa.PrivateKey from a given index.
func InsecureDeterministicEcdsaKeyByIndex(c elliptic.Curve, idx uint64) *ecdsa.PrivateKey {

	// with golang <= 1.19, we used the following code to generate deterministic keys.
	// But in golang 1.20, ecdsa.GenerateKey became non-deterministic and therefore the keys are now hardcoded.
	/*
		// use 555 as offset to deterministically generate key 0 to match vaa-test such that
		// we generate the same key.
		r := mathrand.New(mathrand.NewSource(int64(555 + idx))) //#nosec G404 Testnet/devnet keys are not secret.
		key, err := ecdsa.GenerateKey(c, r)
		if err != nil {
			panic(err)
		}
	*/

	keys := []string{
		"cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
		"c3b2e45c422a1602333a64078aeb42637370b0f48fe385f9cfa6ad54a8e0c47e",
		"9f790d3f08bc4b5cd910d4278f3deb406e57bb5e924906ccd52052bb078ccd47",
		"b20cc49d6f2c82a5e6519015fc18aa3e562867f85f872c58f1277cfbd2a0c8e4",
	}
	privKey, err := eth_crypto.HexToECDSA(keys[idx])

	if err != nil {
		panic(err)
	}

	return privKey
}
