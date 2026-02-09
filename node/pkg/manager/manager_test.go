package manager

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompressPublicKey_LeadingZeroCoordinate tests that compressPublicKey
// handles public keys whose X or Y coordinate has a leading zero byte.
//
// big.Int.Bytes() strips leading zeros, so a 31-byte coordinate produces a
// 64-byte uncompressed payload (0x04 || 31-byte X || 32-byte Y) instead of the
// required 65 bytes. btcec.ParsePubKey rejects anything that is not exactly 33
// or 65 bytes, causing compressPublicKey to return nil.
//
// With a random key the probability of at least one coordinate being < 32 bytes
// is ~0.8 %. For a set of 7 managers the probability that at least one key
// triggers this bug exceeds 5 %.
func TestCompressPublicKey_LeadingZeroCoordinate(t *testing.T) {
	// secp256k1 point at scalar 122: Y coordinate has a leading zero byte,
	// so big.Int.Bytes() returns only 31 bytes for Y.
	//   X = 139ae46a1133f1f9d23f25efba0f6dd87bf7ddaf568a5fb9e0a3bfda73176237
	//   Y = 00995e555c8aabd263fd238833a12188b8a5ffbeb480ba0e3e6ec481a8991472
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(), // curve field is unused; only X/Y matter
		X: new(big.Int).SetBytes([]byte{
			0x13, 0x9a, 0xe4, 0x6a, 0x11, 0x33, 0xf1, 0xf9,
			0xd2, 0x3f, 0x25, 0xef, 0xba, 0x0f, 0x6d, 0xd8,
			0x7b, 0xf7, 0xdd, 0xaf, 0x56, 0x8a, 0x5f, 0xb9,
			0xe0, 0xa3, 0xbf, 0xda, 0x73, 0x17, 0x62, 0x37,
		}),
		Y: new(big.Int).SetBytes([]byte{
			0x00, 0x99, 0x5e, 0x55, 0x5c, 0x8a, 0xab, 0xd2,
			0x63, 0xfd, 0x23, 0x88, 0x33, 0xa1, 0x21, 0x88,
			0xb8, 0xa5, 0xff, 0xbe, 0xb4, 0x80, 0xba, 0x0e,
			0x3e, 0x6e, 0xc4, 0x81, 0xa8, 0x99, 0x14, 0x72,
		}),
	}

	// Confirm Y is indeed shorter than 32 bytes via Bytes() (the root cause).
	require.Less(t, len(pubKey.Y.Bytes()), 32, "Y.Bytes() should be < 32 bytes due to leading zero")

	// compressPublicKey must return a valid 33-byte compressed key even when
	// a coordinate is shorter than 32 bytes.
	result := compressPublicKey(pubKey)
	require.NotNil(t, result, "compressPublicKey must not return nil for keys with leading-zero coordinates")

	expected, _ := hex.DecodeString("02139ae46a1133f1f9d23f25efba0f6dd87bf7ddaf568a5fb9e0a3bfda73176237")
	assert.Equal(t, expected, result, "compressed key must match expected value")
}
