package guardiansigner

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSignerUri(t *testing.T) {
	tests := []struct {
		label        string
		path         string
		expectedType SignerType
	}{
		{label: "RandomText", path: "RandomText", expectedType: InvalidSignerType},
		{label: "ArbitraryUriScheme", path: "arb://data", expectedType: InvalidSignerType},
		// File
		{label: "FileURI", path: "file://whatever", expectedType: FileSignerType},
		{label: "FileUriNoSchemeSeparator", path: "filewhatever", expectedType: InvalidSignerType},
		{label: "FileUriMultipleSchemeSeparators", path: "file://testing://this://", expectedType: FileSignerType},
		{label: "FileUriTraversal", path: "file://../../../file", expectedType: FileSignerType},
		// Amazon KMS
		{label: "AmazonKmsURI", path: "amazonkms://some-arn", expectedType: AmazonKmsSignerType},
	}

	for _, testcase := range tests {
		t.Run(testcase.label, func(t *testing.T) {
			signerType, _, err := ParseSignerUri(testcase.path)

			assert.Equal(t, signerType, testcase.expectedType)

			// If the signer type is Invalid, then an error should have been returned.
			if testcase.expectedType == InvalidSignerType {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestFileSignerNonExistentFile(t *testing.T) {
	nonexistentFileUri := "file://somewhere/on/disk.key"

	// Attempt to generate signer using top-level generator
	_, err := NewGuardianSignerFromUri(context.Background(), nonexistentFileUri, true)
	assert.Error(t, err)

	// Attempt to generate signer using NewFileSigner
	_, keyPath, _ := ParseSignerUri(nonexistentFileUri)
	fileSigner, err := NewFileSigner(context.Background(), true, keyPath)
	assert.Nil(t, fileSigner)
	assert.Error(t, err)
}

func TestFileSigner(t *testing.T) {
	ctx := context.Background()
	fileUri := "file://../query/dev.guardian.key"
	expectedEthAddress := "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"

	// For each file signer generation attempt, check:
	//	That the signer returned is not nil
	//	No error is returned
	//	The public key returned by PublicKey(), converted to an eth address,
	//		matches the expected address.

	// Attempt to generate signer using top-level generator
	fileSigner1, err := NewGuardianSignerFromUri(ctx, fileUri, true)
	require.NoError(t, err)
	assert.NotNil(t, fileSigner1)
	assert.Equal(t, ethcrypto.PubkeyToAddress(fileSigner1.PublicKey(ctx)).Hex(), expectedEthAddress)

	// Attempt to generate signer using NewFileSigner
	signerType, keyPath, err := ParseSignerUri(fileUri)
	assert.Equal(t, signerType, FileSignerType)
	require.NoError(t, err)

	fileSigner2, err := NewFileSigner(ctx, true, keyPath)
	require.NoError(t, err)
	assert.NotNil(t, fileSigner2)
	assert.Equal(t, ethcrypto.PubkeyToAddress(fileSigner2.PublicKey(ctx)).Hex(), expectedEthAddress)

	// Sign some arbitrary data
	data := crypto.Keccak256Hash([]byte("data"))
	sig, err := fileSigner1.Sign(ctx, data.Bytes())
	assert.NoError(t, err)

	// Verify the signature
	valid, _ := fileSigner1.Verify(ctx, sig, data.Bytes())
	assert.True(t, valid)

	// Use generated signature with incorrect hash, should fail
	arbitraryHash := crypto.Keccak256Hash([]byte("arbitrary hash data"))
	valid, _ = fileSigner1.Verify(ctx, sig, arbitraryHash.Bytes())
	assert.False(t, valid)

}

func TestAmazonKmsAdjustBufferSize(t *testing.T) {

	bytes_30_null_0102, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000102")
	bytes_33_01, _ := hex.DecodeString("010101010101010101010101010101010101010101010101010101010101010101")
	bytes_32_01, _ := hex.DecodeString("0101010101010101010101010101010101010101010101010101010101010101")

	full_of_null_bytes, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")

	tests := []struct {
		name           string
		input          []byte
		expectedOutput []byte
	}{
		{
			name:           "LeftPadSmallInput",
			input:          []byte{0x1, 0x2},
			expectedOutput: bytes_30_null_0102,
		},
		{
			name:           "TruncateLargeInput",
			input:          bytes_33_01,
			expectedOutput: bytes_32_01,
		},
		{
			name:           "Leave32ByteInputAsIs",
			input:          bytes_32_01,
			expectedOutput: bytes_32_01,
		},
		{
			name:           "Return32NullBytesOnEmptyInput",
			input:          []byte{},
			expectedOutput: full_of_null_bytes,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			output := adjustBufferSize(testcase.input)
			assert.Equal(t, testcase.expectedOutput, output)
		})
	}
}
