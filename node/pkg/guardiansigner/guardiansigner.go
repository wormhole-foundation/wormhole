package guardiansigner

import (
	"context"
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
)

// GuardianSigner interface. Each function in the GuardianSigner interface
// expects a context to be supplied. This is because signers might interact
// with external services that have the potential of introducing unwanted
// behaviour, like timing out or hanging indefinitely. It's up to each signer
// implementation to decide how to handle the context.
type GuardianSigner interface {
	// Sign expects a keccak256 hash that needs to be signed.
	Sign(ctx context.Context, hash []byte) (sig []byte, err error)
	// PublicKey returns the ECDSA public key of the signer.
	PublicKey(ctx context.Context) (pubKey ecdsa.PublicKey)
	// Verify is a convenience function that recovers a public key from the sig/hash pair,
	// and checks if the public key matches that of the guardian signer.
	Verify(ctx context.Context, sig []byte, hash []byte) (valid bool, err error)
	// Return the type of signer as string.
	TypeAsString() string
}

// Create a new GuardianSigner from the given URI. The caller can also specify the
// unsafeDevMode flag, which signals that the signer is running in an unsafe development
// environment. This is used, for example, to signal the file signer that it should check
// whether or not the key is deterministic.
//
// Additionally, a context is expected to be supplied, as the signer might interact with
// external services during construction. For example, the Amazon KMS signer validates that
// the ARN is valid and retrieves the public key from the service.
func NewGuardianSignerFromUri(ctx context.Context, signerUri string, unsafeDevMode bool) (GuardianSigner, error) {
	// Get the signer type and key configuration. The key configuration
	// isn't interpreted as anything in particular here, as each signer
	// implementation requires different configurations; i.e., the file
	// signer requires a path and the amazon kms signer requires an ARN.
	signerType, signerKeyConfig, err := ParseSignerUri(signerUri)

	if err != nil {
		return nil, err
	}

	var guardianSigner GuardianSigner

	// Create the new guardian signer, based on the signerType. If an invalid
	// signer type is supplied, an error is returned; or if the signer creation
	// returns an error, the error is bubbled up.
	// nolint:exhaustive // default is sufficient for handling errors
	switch signerType {
	case FileSignerType:
		guardianSigner, err = NewFileSigner(ctx, unsafeDevMode, signerKeyConfig)
	case AmazonKmsSignerType:
		guardianSigner, err = NewAmazonKmsSigner(ctx, unsafeDevMode, signerKeyConfig)
	default:
		return nil, errors.New("unsupported guardian signer type")
	}

	if err != nil {
		return nil, err
	}

	// Wrap the guardian signer in a benchmark signer, which will record the
	// time taken to sign and verify messages.
	return BenchmarkWrappedSigner(guardianSigner), nil
}

// Parse the signer URI and return the signer type and key configuration. The signer
// URI is expected to be in the format <signer-type>://<key-configuration>.
func ParseSignerUri(signerUri string) (signerType SignerType, signerKeyConfig string, err error) {
	// Split the URI using the standard "://" scheme separator
	signerUriSplit := strings.Split(signerUri, "://")

	// This check ensures that the URI is in the correct format by checking that the split
	// has at least two elements.
	if len(signerUriSplit) < 2 {
		return InvalidSignerType, "", errors.New("no path separator in guardian signer URI")
	}

	typeStr := signerUriSplit[0]

	// Rejoin the remainder of the split URI as the configuration for the guardian signer
	// implementation. The remainder of the split is joined using the URI scheme separator, as
	// the key configuration might require the same separator.
	keyConfig := strings.Join(signerUriSplit[1:], "://")

	// Return the signer type and key configuration. If the signer type is not supported, an
	// error is returned.
	switch typeStr {
	case "file":
		return FileSignerType, keyConfig, nil
	case "amazonkms":
		return AmazonKmsSignerType, keyConfig, nil
	default:
		return InvalidSignerType, "", fmt.Errorf("unsupported guardian signer type: %s", typeStr)
	}
}
