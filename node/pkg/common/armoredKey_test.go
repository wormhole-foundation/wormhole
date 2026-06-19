package common

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadArmoredKeyRejectsUnsafePermissions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	keyPath := t.TempDir() + "/guardian.key"
	require.NoError(t, WriteArmoredKey(key, "test key", keyPath, GuardianKeyArmoredBlock, false))
	require.NoError(t, os.Chmod(keyPath, 0644))

	loadedKey, err := LoadArmoredKey(keyPath, GuardianKeyArmoredBlock, false)

	require.Error(t, err)
	assert.Nil(t, loadedKey)
	assert.Contains(t, err.Error(), "insecure permissions")
}

func TestLoadArmoredKeyAllowsOwnerOnlyPermissions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	keyPath := t.TempDir() + "/guardian.key"
	require.NoError(t, WriteArmoredKey(key, "test key", keyPath, GuardianKeyArmoredBlock, false))

	loadedKey, err := LoadArmoredKey(keyPath, GuardianKeyArmoredBlock, false)

	require.NoError(t, err)
	assert.Equal(t, key, loadedKey)
}
