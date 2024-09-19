package guardiansigner

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
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
	}

	for _, testcase := range tests {
		t.Run(testcase.label, func(t *testing.T) {
			signerType, _ := ParseSignerUri(testcase.path)

			assert.Equal(t, signerType, testcase.expectedType)
		})
	}
}

func TestFileSignerNonExistentFile(t *testing.T) {
	nonexistentFileUri := "file://somewhere/on/disk.key"

	// Attempt to generate signer using top-level generator
	_, err := NewGuardianSignerFromUri(nonexistentFileUri, true)
	assert.Error(t, err)

	// Attempt to generate signer using NewFileSigner
	_, keyPath := ParseSignerUri(nonexistentFileUri)
	fileSigner, err := NewFileSigner(true, keyPath)
	assert.Nil(t, fileSigner)
	assert.Error(t, err)
}

func TestFileSigner(t *testing.T) {
	fileUri := "file://../query/dev.guardian.key"
	expectedEthAddress := "0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"

	// For each file signer generation attempt, check:
	//	That the signer returned is not nil
	//	No error is returned
	//	The public key returned by PublicKey(), converted to an eth address,
	//		matches the expected address.

	// Attempt to generate signer using top-level generator
	fileSigner1, err := NewGuardianSignerFromUri(fileUri, true)
	assert.NoError(t, err)
	assert.NotNil(t, fileSigner1)
	assert.Equal(t, ethcrypto.PubkeyToAddress(fileSigner1.PublicKey()).Hex(), expectedEthAddress)

	// Attempt to generate signer using NewFileSigner
	signerType, keyPath := ParseSignerUri(fileUri)
	assert.Equal(t, signerType, FileSignerType)

	fileSigner2, err := NewFileSigner(true, keyPath)
	assert.NoError(t, err)
	assert.NotNil(t, fileSigner2)
	assert.Equal(t, ethcrypto.PubkeyToAddress(fileSigner2.PublicKey()).Hex(), expectedEthAddress)

	// Sign some arbitrary data
	data := crypto.Keccak256Hash([]byte("data"))
	sig, err := fileSigner1.Sign(data.Bytes())
	assert.NoError(t, err)

	// Verify the signature
	valid, _ := fileSigner1.Verify(sig, data.Bytes())
	assert.True(t, valid)

	// Use generated signature with incorrect hash, should fail
	arbitraryHash := crypto.Keccak256Hash([]byte("arbitrary hash data"))
	valid, _ = fileSigner1.Verify(sig, arbitraryHash.Bytes())
	assert.False(t, valid)

}
