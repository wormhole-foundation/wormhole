package guardiansigner

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// The GeneratedSigner is a signer that is intended for use in tests. It uses the private
// key supplied to GenerateSignerWithPrivatekeyUnsafe, or defaults to generating a random
// private key if no private key is supplied.
type GeneratedSigner struct {
	privateKey *ecdsa.PrivateKey
}

// NewGeneratedSigner creates a new GeneratedSigner. If key is nil, a random private key
// is generated. Otherwise, the private key is used as-is.
func NewGeneratedSigner(key *ecdsa.PrivateKey) (*GeneratedSigner, error) {
	if key == nil {
		privateKey, err := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
		return &GeneratedSigner{privateKey: privateKey}, err
	} else {
		return &GeneratedSigner{privateKey: key}, nil
	}

}

func (gs *GeneratedSigner) Sign(ctx context.Context, hash []byte) (sig []byte, err error) {
	// Sign the hash
	sig, err = ethcrypto.Sign(hash, gs.privateKey)

	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	return sig, nil
}

func (gs *GeneratedSigner) PublicKey(ctx context.Context) (pubKey ecdsa.PublicKey) {
	return gs.privateKey.PublicKey
}

func (gs *GeneratedSigner) Verify(ctx context.Context, sig []byte, hash []byte) (valid bool, err error) {
	recoveredPubKey, err := ethcrypto.SigToPub(hash, sig)

	if err != nil {
		return false, err
	}

	// Need to use gs.privateKey.Public() instead of PublicKey to ensure
	// the returned public key has the right interface for Equal() to work.
	fsPubkey := gs.privateKey.Public()

	return recoveredPubKey.Equal(fsPubkey), nil
}

// This function is meant to be a helper function that returns a guardian signer for tests
// that simply require a private key. The caller can specify a private key to be used, or
// pass nil to have `NewGeneratedSigner` generate a random private key.
func GenerateSignerWithPrivatekeyUnsafe(key *ecdsa.PrivateKey) (GuardianSigner, error) {
	return NewGeneratedSigner(key)
}
