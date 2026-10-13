package wormconn

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadWormchainPrivKeyRejectsUnsafePermissions(t *testing.T) {
	keyPath := t.TempDir() + "/wormchain.key"
	require.NoError(t, os.WriteFile(keyPath, []byte("not a valid encrypted key"), 0644)) // #nosec G306 -- This test verifies unsafe key-file permissions are rejected.

	key, err := LoadWormchainPrivKey(keyPath, "passphrase")

	require.Error(t, err)
	assert.Nil(t, key)
	assert.Contains(t, err.Error(), "insecure permissions")
}
