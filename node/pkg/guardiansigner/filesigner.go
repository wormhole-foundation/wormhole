package guardiansigner

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"google.golang.org/protobuf/proto"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"golang.org/x/crypto/openpgp/armor" // nolint
)

type FileSigner struct {
	keyPath    string
	privateKey *ecdsa.PrivateKey
}

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

func NewFileSigner(unsafeDevMode bool, signerKeyPath string) (*FileSigner, error) {
	fileSigner := &FileSigner{
		keyPath: signerKeyPath,
	}

	f, err := os.Open(signerKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	p, err := armor.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored file: %w", err)
	}

	if p.Type != GuardianKeyArmoredBlock {
		return nil, fmt.Errorf("invalid block type: %s", p.Type)
	}

	b, err := io.ReadAll(p.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var m nodev1.GuardianKey
	err = proto.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize protobuf: %w", err)
	}

	if !unsafeDevMode && m.UnsafeDeterministicKey {
		return nil, errors.New("refusing to use deterministic key in production")
	}

	gk, err := ethcrypto.ToECDSA(m.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize raw key data: %w", err)
	}

	fileSigner.privateKey = gk
	return fileSigner, nil
}

func (fs *FileSigner) Sign(hash []byte) ([]byte, error) {

	// Sign the hash
	sig, err := crypto.Sign(hash, fs.privateKey)

	if err != nil {
		return nil, fmt.Errorf("failed to sign wormchain address: %w", err)
	}

	return sig, nil
}

func (fs *FileSigner) PublicKey() ecdsa.PublicKey {
	publicKey := fs.privateKey.PublicKey
	return publicKey
}

func (fs *FileSigner) Verify(sig []byte, hash []byte) (bool, error) {

	recoveredPubKey, err := ethcrypto.SigToPub(hash, sig)

	if err != nil {
		return false, err
	}

	// Need to use fs.privateKey.Public() instead of PublicKey to ensure
	// the returned public key has the right interface for Equal() to work.
	fsPubkey := fs.privateKey.Public()

	return recoveredPubKey.Equal(fsPubkey), nil
}
