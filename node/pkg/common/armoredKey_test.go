package common

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadArmoredKeyRejectsUnsafePermissions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "guardian.key")
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
	keyPath := filepath.Join(t.TempDir(), "guardian.key")
	require.NoError(t, WriteArmoredKey(key, "test key", keyPath, GuardianKeyArmoredBlock, false))

	loadedKey, err := LoadArmoredKey(keyPath, GuardianKeyArmoredBlock, false)

	require.NoError(t, err)
	assert.Equal(t, key, loadedKey)
}

func openTestFIFO(t *testing.T, perm os.FileMode) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "key.fifo")
	require.NoError(t, syscall.Mkfifo(path, 0600))
	// Set the mode explicitly because Mkfifo applies the umask.
	require.NoError(t, os.Chmod(path, perm))
	// O_RDWR opens a FIFO without a blocking wait for a writer on Linux and macOS.
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestValidatePrivateKeyFilePermissionsRejectsUnsafeFIFO(t *testing.T) {
	f := openTestFIFO(t, 0644)

	err := ValidatePrivateKeyFilePermissions(f)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insecure permissions")
}

func TestValidatePrivateKeyFilePermissionsAllowsOwnerOnlyFIFO(t *testing.T) {
	f := openTestFIFO(t, 0600)

	require.NoError(t, ValidatePrivateKeyFilePermissions(f))
}

func TestValidatePrivateKeyFilePermissionsAllowsAnonymousPipeOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("only Linux reports anonymous pipes as 0600")
	}
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})

	require.NoError(t, ValidatePrivateKeyFilePermissions(r))
}
