package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/certusone/wormhole/node/pkg/watchers/solana/testgen"
)

// errAborted is returned when the user declines an overwrite.
var errAborted = errors.New("aborted")

// writeBundles confirms (if path exists) then writes. Use writeBundlesFile directly when the
// overwrite has already been confirmed (e.g. the `all` pre-flight).
func writeBundles(path string, bundles []*testgen.Bundle) error {
	if fileExists(path) && !confirmOverwrite(path) {
		return errAborted
	}
	return writeBundlesFile(path, bundles)
}

// writeBundlesFile minifies and atomically writes bundles to path without prompting. The
// caller controls ordering (static keeps scenario order; live is sorted by collectLiveBundles).
func writeBundlesFile(path string, bundles []*testgen.Bundle) error {
	buf, err := json.Marshal(bundles)
	if err != nil {
		return fmt.Errorf("marshal bundles: %w", err)
	}
	buf = append(buf, '\n')

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil { //nolint:gosec // fixture file
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	fmt.Fprintf(os.Stderr, "wrote %d bundles to %s\n", len(bundles), path)
	return nil
}

// fileExists reports whether path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// countRecordedHashes returns how many bundles in the file carry a non-empty `expected` block.
func countRecordedHashes(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var bundles []struct {
		Expected json.RawMessage `json:"expected"`
	}
	if json.Unmarshal(raw, &bundles) != nil {
		return 0
	}
	n := 0
	for _, b := range bundles {
		if len(b.Expected) > 0 && string(b.Expected) != "null" {
			n++
		}
	}
	return n
}

// confirmOverwrite prints the destructive-overwrite warning and reads a y/N answer from stdin.
// A non-interactive / EOF stdin returns false, so automation can never overwrite.
func confirmOverwrite(path string) bool {
	printOverwriteWarning(path)
	fmt.Fprint(os.Stderr, "Overwrite? [y/N]: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nno confirmation on stdin; aborting.")
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}

func printOverwriteWarning(path string) {
	const rule = "========================================================================"
	fmt.Fprintf(os.Stderr, "\n%s\n", rule)
	fmt.Fprintf(os.Stderr, "                              *** WARNING ***\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", rule)
	fmt.Fprintf(os.Stderr, "  %s\n", path)
	fmt.Fprintf(os.Stderr, "  already exists with %d bundle(s) carrying recorded `expected` hashes.\n\n", countRecordedHashes(path))
	fmt.Fprintf(os.Stderr, "  Overwriting DESTROYS those recorded regression hashes. Only proceed if\n")
	fmt.Fprintf(os.Stderr, "  you intend to re-record them (re-run the replay test afterwards).\n\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", rule)
}
