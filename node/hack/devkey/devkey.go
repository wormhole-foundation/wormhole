// Package devkey stages committed devnet keys for the test tools under node/hack.
package devkey

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const ownerOnlyKeyFileMode = 0600

// Stage copies a committed devnet key to a fixed 0600 path in the temp
// directory and returns that path. The key loaders reject group- or
// world-readable files, and git only stores the committed keys as 0644, so this
// avoids a manual chmod after every checkout. Each run replaces the same file,
// so runs that exit through logger.Fatal or Ctrl-C leave at most one copy. The
// staged name includes the parent directory of the key, so tools that stage keys
// with the same file name do not replace each other's copies.
func Stage(path string) (string, error) {
	key, err := os.ReadFile(path) // #nosec G304 -- Path is a committed devnet key chosen by the calling tool.
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	staged := filepath.Join(os.TempDir(), "wormhole-"+filepath.Base(filepath.Dir(abs))+"-"+filepath.Base(abs))
	// Remove the copy from a previous run, then create the file with O_EXCL so
	// that we never write through a symlink planted in a shared temp directory.
	if err = os.Remove(staged); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	f, err := os.OpenFile(staged, os.O_WRONLY|os.O_CREATE|os.O_EXCL, ownerOnlyKeyFileMode)
	if err != nil {
		return "", err
	}
	if _, err = f.Write(key); err != nil {
		_ = f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return staged, nil
}
