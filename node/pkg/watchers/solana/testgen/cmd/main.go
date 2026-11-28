// Command testgen rebuilds synthetic Solana watcher fixtures and collects fresh live artifacts.
//
//	testgen static           build the synthetic matrix -> <out-dir>/static_bundles.json
//	testgen live  --rpc URL  collect on-chain transactions -> <out-dir>/live_bundles.json
//	testgen all   --rpc URL  rebuild synthetic fixtures and collect fresh live artifacts
//
// Overwriting an existing fixture prompts for confirmation because it removes the recorded
// `expected` VAA signing digests.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/solana/testgen"
)

const (
	defaultOutDir = "./pkg/watchers/solana/testdata"
	usageExitCode = 2

	defaultSleepSeconds  = 0.1
	defaultPostMessages  = 50
	defaultShimMessages  = 50
	defaultCloseMessages = 20
	defaultPageSize      = 100
	defaultMaxPages      = 200
)

func main() {
	if len(os.Args) < usageExitCode {
		usage()
		os.Exit(usageExitCode)
	}

	var err error
	switch os.Args[1] {
	case "static":
		err = runStatic(os.Args[2:])
	case "live":
		err = runLive(os.Args[2:])
	case "all":
		err = runAll(os.Args[2:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n", os.Args[1])
		usage()
		os.Exit(usageExitCode)
	}

	switch {
	case errors.Is(err, errShortfall):
		os.Exit(usageExitCode)
	case err != nil:
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `testgen rebuilds synthetic Solana watcher fixtures and collects fresh live artifacts.

Usage:
  testgen static [--out-dir DIR]
  testgen live  --rpc URL [flags] [--out-dir DIR]
  testgen all   --rpc URL [flags] [--out-dir DIR]

  static  deterministically rebuild the synthetic corpus -> <out-dir>/%s
  live    collect a fresh corpus from on-chain history    -> <out-dir>/%s
  all     rebuild the synthetic corpus and collect a fresh live corpus (requires --rpc)

Overwriting an existing fixture prompts for confirmation (it removes recorded VAA signing digests).
`, testgen.StaticBundlesFilename, testgen.LiveBundlesFilename)
}

func runStatic(args []string) error {
	fs := flag.NewFlagSet("static", flag.ExitOnError)
	outDir := fs.String("out-dir", defaultOutDir, "output directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	bundles, err := buildStaticBundles()
	if err != nil {
		return err
	}
	return writeBundles(filepath.Join(*outDir, testgen.StaticBundlesFilename), bundles)
}

func runLive(args []string) error {
	fs := flag.NewFlagSet("live", flag.ExitOnError)
	outDir := fs.String("out-dir", defaultOutDir, "output directory")
	cfg, sleepSecs := liveFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cfg.rpc == "" {
		return errors.New("--rpc is required")
	}
	cfg.sleep = time.Duration(*sleepSecs * float64(time.Second))

	bundles, err := collectLiveBundles(context.Background(), *cfg)
	if err != nil {
		return err
	}
	return writeBundles(filepath.Join(*outDir, testgen.LiveBundlesFilename), bundles)
}

func runAll(args []string) error {
	fs := flag.NewFlagSet("all", flag.ExitOnError)
	outDir := fs.String("out-dir", defaultOutDir, "output directory")
	cfg, sleepSecs := liveFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if cfg.rpc == "" {
		return errors.New("all requires --rpc")
	}
	cfg.sleep = time.Duration(*sleepSecs * float64(time.Second))

	staticPath := filepath.Join(*outDir, testgen.StaticBundlesFilename)
	livePath := filepath.Join(*outDir, testgen.LiveBundlesFilename)

	// Confirm both overwrites before any (slow) network I/O so we fail fast.
	for _, p := range []string{staticPath, livePath} {
		if fileExists(p) && !confirmOverwrite(p) {
			return errAborted
		}
	}

	staticBundles, err := buildStaticBundles()
	if err != nil {
		return err
	}
	liveBundles, err := collectLiveBundles(context.Background(), *cfg)
	if err != nil {
		return err
	}

	// Do not replace either corpus until both have been built successfully.
	if err := writeBundlesFile(staticPath, staticBundles); err != nil {
		return err
	}
	if err := writeBundlesFile(livePath, liveBundles); err != nil {
		return err
	}
	return nil
}

// liveFlags registers the live-collection flags on fs and returns the config plus a pointer to
// the raw --sleep seconds (converted to cfg.sleep by the caller after Parse).
func liveFlags(fs *flag.FlagSet) (*liveConfig, *float64) {
	cfg := &liveConfig{}
	sleepSecs := fs.Float64("sleep", defaultSleepSeconds, "delay between RPC calls in seconds (rate limiting)")
	fs.StringVar(&cfg.rpc, "rpc", "", "Solana JSON-RPC URL (required)")
	fs.IntVar(&cfg.postMessage, "post-message", defaultPostMessages, "number of post_message transactions")
	fs.IntVar(&cfg.shim, "shim", defaultShimMessages, "number of shim transactions")
	fs.IntVar(&cfg.close, "close", defaultCloseMessages, "number of close_posted_message transactions")
	fs.IntVar(&cfg.pageSize, "page-size", defaultPageSize, "getSignaturesForAddress page size (max 1000)")
	fs.IntVar(&cfg.maxPages, "max-pages", defaultMaxPages, "max signature pages per program scan")
	fs.StringVar(&cfg.wormholescanBase, "wormholescan-base", defaultWormholescanBase, "WormholeScan API base URL")
	fs.BoolVar(&cfg.verifyWormholescan, "verify-wormholescan", false, "verify each signature against WormholeScan (off by default)")
	fs.BoolVar(&cfg.keepMissingAccount, "keep-missing-account", false, "keep post_message records whose message account can no longer be fetched")
	fs.StringVar(&cfg.closeAnchor, "close-anchor", defaultCloseAnchor, "signature to anchor the close search near; empty to search from the tip")
	return cfg, sleepSecs
}
