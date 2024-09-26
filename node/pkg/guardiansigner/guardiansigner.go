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
	// file://<path-to-file>
	FileSignerType
)

// GuardianSigner interface
type GuardianSigner interface {
	// Sign expects a keccak256 hash that needs to be signed.
	Sign(hash []byte) (sig []byte, err error)
	// PublicKey returns the ECDSA public key of the signer. Note that this should not
	// be confused with the EVM address.
	PublicKey() (pubKey ecdsa.PublicKey)
	// Verify is a convenience function that recovers a public key from the sig/hash pair,
	// and checks if the public key matches that of the guardian signer.
	Verify(sig []byte, hash []byte) (valid bool, err error)
}

func NewGuardianSignerFromUri(signerUri string, unsafeDevMode bool) (GuardianSigner, error) {

	// Get the signer type
	signerType, signerKeyConfig := ParseSignerUri(signerUri)

	switch signerType {
	case FileSignerType:
		return NewFileSigner(unsafeDevMode, signerKeyConfig)
	default:
		return nil, fmt.Errorf("unsupported guardian signer type")
	}
}

func ParseSignerUri(signerUri string) (signerType SignerType, signerKeyConfig string) {
	// Split the URI using the standard "://" scheme separator
	signerUriSplit := strings.Split(signerUri, "://")

	// This check is purely for ensuring that there is actually a path separator.
	if len(signerUriSplit) < 2 {
		return InvalidSignerType, ""
	}

	typeStr := signerUriSplit[0]
	// Rejoin the remainder of the split URI as the configuration for the guardian signer
	// implementation. The remainder of the split is joined using the URI scheme separator.
	keyConfig := strings.Join(signerUriSplit[1:], "://")

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
func GenerateSignerWithPrivatekeyUnsafe(key *ecdsa.PrivateKey) (GuardianSigner, error) {
	return NewGeneratedSigner(key)
}
