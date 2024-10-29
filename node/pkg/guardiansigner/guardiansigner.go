package guardiansigner

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
)

// The types of guardian signers that are supported
type SignerType int

const (
	InvalidSignerType SignerType = iota
	// file://<path-to-file>
	FileSignerType
	// amazonkms://<arn>
	AmazonKmsSignerType
	// benchmark://<uri>://<config>
	BenchmarkSignerType
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
	signerType, signerKeyConfig, err := ParseSignerUri(signerUri)

	if err != nil {
		return nil, err
	}

	switch signerType {
	case FileSignerType:
		return NewFileSigner(unsafeDevMode, signerKeyConfig)
	case AmazonKmsSignerType:
		return NewAmazonKmsSigner(unsafeDevMode, signerKeyConfig)
	case BenchmarkSignerType:
		return NewBenchmarkSigner(unsafeDevMode, signerKeyConfig)
	default:
		return nil, errors.New("unsupported guardian signer type")
	}
}

func ParseSignerUri(signerUri string) (signerType SignerType, signerKeyConfig string, err error) {
	// Split the URI using the standard "://" scheme separator
	signerUriSplit := strings.Split(signerUri, "://")

	// This check is purely for ensuring that there is actually a path separator.
	if len(signerUriSplit) < 2 {
		return InvalidSignerType, "", errors.New("no path separator in guardian signer URI")
	}

	typeStr := signerUriSplit[0]
	// Rejoin the remainder of the split URI as the configuration for the guardian signer
	// implementation. The remainder of the split is joined using the URI scheme separator.
	keyConfig := strings.Join(signerUriSplit[1:], "://")

	switch typeStr {
	case "file":
		return FileSignerType, keyConfig, nil
	case "amazonkms":
		return AmazonKmsSignerType, keyConfig, nil
	case "benchmark":
		return BenchmarkSignerType, keyConfig, nil
	default:
		return InvalidSignerType, "", fmt.Errorf("unsupported guardian signer type: %s", typeStr)
	}
}
