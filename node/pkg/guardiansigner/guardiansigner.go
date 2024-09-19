package guardiansigner

import (
	"crypto/ecdsa"
	"fmt"
	"strings"
)

// The types of guardian signers that are supported
type SignerType int

const (
	InvalidSignerType SignerType = iota
	FileSignerType
)

// GuardianSigner interface
type GuardianSigner interface {
	// TODO: document that the signer implementations assume they are recieving a keccack256 hash
	Sign(hash []byte) (sig []byte, err error)
	PublicKey() (pubKey ecdsa.PublicKey)
	Verify(sig []byte, hash []byte) (valid bool, err error)
}

func NewGuardianSignerFromUri(signerUri string, unsafeDevMode bool) (GuardianSigner, error) {

	// Get the signer type
	signerType, signerKeyConfig := ParseSignerUri(signerUri)

	switch signerType {
	case FileSignerType:
		return NewFileSigner(unsafeDevMode, signerKeyConfig)
	case InvalidSignerType:
		return nil, fmt.Errorf("unsupported guardian signer type")
	default:
		return nil, fmt.Errorf("unsupported guardian signer type")
	}
}

func ParseSignerUri(signerUri string) (signerType SignerType, signerKeyConfig string) {
	signerUriSplit := strings.Split(signerUri, "://")

	if len(signerUriSplit) < 2 {
		return InvalidSignerType, ""
	}

	typeStr := signerUriSplit[0]
	keyConfig := strings.Join(signerUriSplit[1:], "")

	switch typeStr {
	case "file":
		return FileSignerType, keyConfig
	default:
		return InvalidSignerType, ""
	}
}

// WARNING: DO NOT USE THIS SIGNER OUTSIDE OF TESTS
//
// This function is meant to be a helper function that returns a guardian signer for tests
// that simply require a private key.
// The caller can specify a private key to be used, or pass nil to have `NewGeneratedSigner`
// generate a random private key.
func GenerateSignerWithPrivatekey(key *ecdsa.PrivateKey) (GuardianSigner, error) {
	return NewGeneratedSigner(key)
}
