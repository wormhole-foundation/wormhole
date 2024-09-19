package guardiansigner

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type GeneratedSigner struct {
	pk *ecdsa.PrivateKey
}

func NewGeneratedSigner(key *ecdsa.PrivateKey) (*GeneratedSigner, error) {
	if key == nil {
		pk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		return &GeneratedSigner{pk: pk}, err
	} else {
		return &GeneratedSigner{pk: key}, nil
	}

}

func (gs *GeneratedSigner) Sign(hash []byte) (sig []byte, err error) {
	// Sign the hash
	sig, err = crypto.Sign(hash, gs.pk)

	if err != nil {
		return nil, fmt.Errorf("failed to sign wormchain address: %w", err)
	}

	return sig, nil
}

func (gs *GeneratedSigner) PublicKey() (pubKey ecdsa.PublicKey) {
	publicKey := gs.pk.PublicKey
	return publicKey
}

func (gs *GeneratedSigner) Verify(sig []byte, hash []byte) (valid bool, err error) {
	recoveredPubKey, err := ethcrypto.SigToPub(hash, sig)

	if err != nil {
		return false, err
	}

	fsPubkey := gs.pk.PublicKey

	return recoveredPubKey.Equal(fsPubkey), nil
}
