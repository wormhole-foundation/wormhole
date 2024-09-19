package guardiansigner

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type GeneratedSigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewGeneratedSigner(key *ecdsa.PrivateKey) (*GeneratedSigner, error) {
	if key == nil {
		privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		return &GeneratedSigner{privateKey: privateKey}, err
	} else {
		return &GeneratedSigner{privateKey: key}, nil
	}

}

func (gs *GeneratedSigner) Sign(hash []byte) (sig []byte, err error) {
	// Sign the hash
	sig, err = crypto.Sign(hash, gs.privateKey)

	if err != nil {
		return nil, fmt.Errorf("failed to sign wormchain address: %w", err)
	}

	return sig, nil
}

func (gs *GeneratedSigner) PublicKey() (pubKey ecdsa.PublicKey) {
	return gs.privateKey.PublicKey
}

func (gs *GeneratedSigner) Verify(sig []byte, hash []byte) (valid bool, err error) {
	recoveredPubKey, err := ethcrypto.SigToPub(hash, sig)

	if err != nil {
		return false, err
	}

	// Need to use gs.privateKey.Public() instead of PublicKey to ensure
	// the returned public key has the right interface for Equal() to work.
	fsPubkey := gs.privateKey.Public()

	return recoveredPubKey.Equal(fsPubkey), nil
}
