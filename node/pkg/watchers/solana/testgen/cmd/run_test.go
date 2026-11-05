package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/certusone/wormhole/node/pkg/watchers/solana/testgen"
	"github.com/stretchr/testify/require"
)

func shortfallArgs(outDir string) []string {
	return []string{
		"--rpc", "http://unused",
		"--post-message", "1",
		"--shim", "0",
		"--close", "0",
		"--max-pages", "0",
		"--out-dir", outDir,
	}
}

func TestRunLiveDoesNotWritePartialCollection(t *testing.T) {
	outDir := t.TempDir()

	require.ErrorIs(t, runLive(shortfallArgs(outDir)), errShortfall)
	_, err := os.Stat(filepath.Join(outDir, testgen.LiveBundlesFilename))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRunAllDoesNotWriteOnPartialLiveCollection(t *testing.T) {
	outDir := t.TempDir()

	require.ErrorIs(t, runAll(shortfallArgs(outDir)), errShortfall)
	for _, name := range []string{testgen.StaticBundlesFilename, testgen.LiveBundlesFilename} {
		_, err := os.Stat(filepath.Join(outDir, name))
		require.ErrorIs(t, err, os.ErrNotExist)
	}
}
