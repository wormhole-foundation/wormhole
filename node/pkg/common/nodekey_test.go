package common

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetOrCreateNodeKeyWithNewPath(t *testing.T) {
	// Get a non-existing temp file path to write auto-generated privKey to
	path := "/tmp/node_key_test_" + fmt.Sprint(rand.Int()) //#nosec G404 no CSPRNG needed here
	defer os.Remove(path)

	logger := zap.NewNop()
	privKey1, _ := GetOrCreateNodeKey(logger, path)
	assert.NotNil(t, privKey1)

	// Re-read the generated privKey file back into memory
	b, _ := os.ReadFile(path)
	privKey2, _ := crypto.UnmarshalPrivateKey(b)

	// Make sure we got the same key
	assert.Equal(t, privKey1, privKey2)
}

func TestGetOrCreateNodeKeyWithPreExistingPath(t *testing.T) {
	// Get a temp file to write our test private key to
	file, err := os.CreateTemp("", "tmpfile1-")
	assert.Nil(t, err)
	defer os.Remove(file.Name())

	// Generate a test private key
	privKey1, _, _ := crypto.GenerateKeyPair(crypto.Ed25519, -1)

	// Marshall the private key to bytes
	marshalledPrivKey, _ := crypto.MarshalPrivateKey(privKey1)

	// Write the private key bytes to temp file
	_, _ = file.Write(marshalledPrivKey)

	logger := zap.NewNop()
	privKey2, _ := GetOrCreateNodeKey(logger, file.Name())

	// Make sure we got the same key
	assert.Equal(t, privKey1, privKey2)
}

func TestGetOrCreateNodeKeyRejectsUnsafePermissions(t *testing.T) {
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	require.NoError(t, err)
	marshalledPrivKey, err := crypto.MarshalPrivateKey(privKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "node.key")
	require.NoError(t, os.WriteFile(path, marshalledPrivKey, 0644)) // #nosec G306 -- This test verifies unsafe key-file permissions are rejected.

	loadedKey, err := GetOrCreateNodeKey(zap.NewNop(), path)

	require.Error(t, err)
	assert.Nil(t, loadedKey)
	assert.Contains(t, err.Error(), "insecure permissions")
}
