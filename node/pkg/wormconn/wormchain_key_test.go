package wormconn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWormchainPrivKeyRejectsUnsafePermissions(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "wormchain.key")
	require.NoError(t, os.WriteFile(keyPath, []byte("not a valid encrypted key"), 0644)) // #nosec G306 -- This test verifies unsafe key-file permissions are rejected.

	key, err := LoadWormchainPrivKey(keyPath, "passphrase")

	require.Error(t, err)
	assert.Nil(t, key)
	assert.Contains(t, err.Error(), "insecure permissions")
}

func TestLoadWormchainPrivKeyAllowsOwnerOnlyPermissions(t *testing.T) {
	privKey := secp256k1.GenPrivKey()
	armored := crypto.EncryptArmorPrivKey(privKey, "passphrase", "secp256k1")
	keyPath := filepath.Join(t.TempDir(), "wormchain.key")
	require.NoError(t, os.WriteFile(keyPath, []byte(armored), 0600))

	key, err := LoadWormchainPrivKey(keyPath, "passphrase")

	require.NoError(t, err)
	assert.True(t, privKey.Equals(key))
}
