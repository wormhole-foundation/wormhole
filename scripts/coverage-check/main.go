package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	baselineFile       = ".coverage-baseline"
	coverageOutputFile = "coverage.txt"
	minNewPkgCoverage  = 10.0
	coverageTolerance  = 0.1 // Allow 0.1% tolerance for floating point comparison
)

// Colors for terminal output
const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorReset  = "\033[0m"
)

type packageCoverage struct {
	name     string
	coverage float64
}

type improvement struct {
	name     string
	baseline float64
	current  float64
}

var (
	verbose        bool
	updateBaseline bool
	ciMode         bool
)

func main() {
	flag.BoolVar(&verbose, "v", false, "verbose output (show all checks)")
	flag.BoolVar(&verbose, "verbose", false, "verbose output (show all checks)")
	flag.BoolVar(&updateBaseline, "u", false, "update baseline with current coverage")
	flag.BoolVar(&updateBaseline, "update", false, "update baseline with current coverage")
	flag.BoolVar(&ciMode, "ci", false, "CI mode: improvements are warnings, not failures")
	flag.Parse()

	if !ciMode && (os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true") {
		ciMode = true
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s❌ %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

func run() error {
	if verbose {
		fmt.Println("========================================")
		fmt.Println("Wormhole Coverage Regression Check")
		fmt.Println("========================================")
		fmt.Println()
		fmt.Println("Reading coverage data from", coverageOutputFile, "...")
	}

	// Parse coverage from test output
	currentCoverage, err := parseCoverageOutput()
	if err != nil {
		return fmt.Errorf("failed to parse coverage output: %w", err)
	}

	if verbose {
		fmt.Printf("Found %d packages with coverage data\n", len(currentCoverage))
		fmt.Println()
	}

	// Allow `-u` to bootstrap the baseline on first run.
	if updateBaseline {
		if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
			return writeInitialBaseline(currentCoverage)
		} else if err != nil {
			return fmt.Errorf("failed to stat baseline file %s: %w", baselineFile, err)
		}
	}

	// Load baseline
	baseline, err := loadBaseline()
	if err != nil {
		return fmt.Errorf("failed to load baseline: %w", err)
	}

	// Check baseline packages for regression and track improvements
	passed, regressions, improvements, improvementList := checkBaseline(baseline, currentCoverage)

	// Check new packages
	newPassed, newFailed, newPackages := checkNewPackages(baseline, currentCoverage)

	// Handle update flag
	if updateBaseline {
		if err := writeUpdatedBaseline(currentCoverage); err != nil {
			return fmt.Errorf("failed to update baseline: %w", err)
		}
		fmt.Printf("%s✅ Baseline updated successfully%s\n", colorGreen, colorReset)
		if improvements > 0 {
			fmt.Printf("   %d package(s) improved\n", improvements)
		}
		if len(newPackages) > 0 {
			fmt.Printf("   %d new package(s) added\n", len(newPackages))
		}
		return nil
	}

	// Summary
	if verbose {
		fmt.Println()
		fmt.Println("========================================")
		fmt.Println("Summary")
		fmt.Println("========================================")
		fmt.Printf("Baseline packages checked: %d\n", passed+regressions)
		fmt.Printf("  - Passed: %d\n", passed)
		fmt.Printf("  - Regressions: %d\n", regressions)
		if improvements > 0 {
			fmt.Printf("  - Improvements: %d\n", improvements)
		}
		fmt.Printf("New packages checked: %d\n", newPassed+newFailed)
		if newPassed+newFailed > 0 {
			fmt.Printf("  - Passed: %d\n", newPassed)
			fmt.Printf("  - Failed: %d\n", newFailed)
		}
		fmt.Println()
	}

	// Check for failures (regressions or new packages below threshold)
	if regressions > 0 || newFailed > 0 {
		if !verbose {
			fmt.Printf("%s❌ Coverage check FAILED%s\n", colorRed, colorReset)
			if regressions > 0 {
				fmt.Printf("  %d package(s) regressed below baseline\n", regressions)
			}
			if newFailed > 0 {
				fmt.Printf("  %d new package(s) below minimum coverage (%.1f%%)\n", newFailed, minNewPkgCoverage)
			}
			fmt.Println()
			fmt.Println("Run with -v flag for details")
		} else {
			fmt.Printf("%s❌ Coverage check FAILED%s\n", colorRed, colorReset)
			fmt.Println()
			fmt.Println("To fix:")
			fmt.Println("  1. Add tests to improve coverage for failing packages")
			fmt.Println("  2. If coverage drop is intentional, update baseline:")
			fmt.Println("     - Run: make update-coverage-baseline")
			fmt.Println("     - Or: ./coverage-check -u")
		}
		return fmt.Errorf("coverage check failed")
	}

	// Check for improvements. In CI these are informational so the build stays green.
	if improvements > 0 || len(newPackages) > 0 {
		fmt.Printf("%s💡 Coverage improved!%s\n", colorYellow, colorReset)
		if improvements > 0 {
			fmt.Printf("  %d package(s) have better coverage than baseline:\n", improvements)
			for _, pkg := range improvementList {
				fmt.Printf("    - %s: %.1f%% → %.1f%%\n", pkg.name, pkg.baseline, pkg.current)
			}
		}
		if len(newPackages) > 0 {
			fmt.Printf("  %d new package(s) with coverage:\n", len(newPackages))
			for _, pkg := range newPackages {
				fmt.Printf("    - %s: %.1f%%\n", pkg.name, pkg.coverage)
			}
		}
		fmt.Println()
		if ciMode {
			fmt.Printf("%sRun 'make update-coverage-baseline' locally to lock in these improvements%s\n", colorYellow, colorReset)
			return nil
		}
		fmt.Printf("%sPlease update the baseline to lock in these improvements:%s\n", colorYellow, colorReset)
		fmt.Println("  Run: make update-coverage-baseline")
		fmt.Println("  Or:  ./coverage-check -u")
		return fmt.Errorf("baseline update required")
	}

	// All checks passed, no improvements
	if verbose {
		fmt.Printf("%s✅ All coverage checks PASSED%s\n", colorGreen, colorReset)
	}
	return nil
}

// writeInitialBaseline creates a baseline file from the current coverage run.
func writeInitialBaseline(current map[string]float64) error {
	packages := make([]string, 0, len(current))
	for pkg := range current {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	file, err := os.Create(baselineFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	fmt.Fprintln(writer, "# Wormhole Go Coverage Baseline")
	fmt.Fprintln(writer, "# Auto-generated by coverage-check")
	fmt.Fprintln(writer, "# Format: <package> <coverage-percentage>")
	fmt.Fprintln(writer, "")

	for _, pkg := range packages {
		fmt.Fprintf(writer, "%s %.1f\n", pkg, current[pkg])
	}

	if err := writer.Flush(); err != nil {
		return err
	}

	fmt.Printf("%s✅ Baseline created successfully%s\n", colorGreen, colorReset)
	return nil
}

// parseCoverageOutput reads the coverage output file and extracts package coverage
func parseCoverageOutput() (map[string]float64, error) {
	file, err := os.Open(coverageOutputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open coverage output file %s: %w\nDid you run 'cd node && go test -cover ./... | tee coverage.txt' first?", coverageOutputFile, err)
	}
	defer file.Close()

	// Parse coverage from output
	// Format: "ok  	<package>	<time>	coverage: <percent>% of statements"
	//     or: "	<package>		coverage: <percent>% of statements" (packages with no tests but coverage reported)
	coverage := make(map[string]float64)
	coverageRe := regexp.MustCompile(`^\s*(ok\s+)?([^\s]+)\s+.*coverage:\s+([0-9.]+)%`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := coverageRe.FindStringSubmatch(line)
		if len(matches) >= 4 {
			pkg := matches[2]
			percentStr := matches[3]
			percent, err := strconv.ParseFloat(percentStr, 64)
			if err != nil {
				continue
			}
			coverage[pkg] = percent
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading coverage output: %w", err)
	}

	return coverage, nil
}

// loadBaseline reads the baseline file and returns a map of package -> coverage
func loadBaseline() (map[string]float64, error) {
	file, err := os.Open(baselineFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	baseline := make(map[string]float64)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse: <package> <coverage>
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid baseline format at line %d: %s", lineNum, line)
		}

		pkg := parts[0]
		coverage, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid coverage value at line %d: %s", lineNum, parts[1])
		}

		baseline[pkg] = coverage
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return baseline, nil
}

// checkBaseline compares current coverage against baseline
func checkBaseline(baseline, current map[string]float64) (passed, regressions, improvements int, improvementList []improvement) {
	if verbose {
		fmt.Println("Checking baseline packages for regression...")
		fmt.Println("--------------------------------------------")
	}

	for pkg, baselineCov := range baseline {
		currentCov, exists := current[pkg]

		if !exists {
			if verbose {
				fmt.Printf("%s⚠️  %s: No coverage data found (baseline: %.1f%%)%s\n",
					colorYellow, pkg, baselineCov, colorReset)
				fmt.Printf("%s   Package may have been removed or renamed%s\n",
					colorYellow, colorReset)
			}
			regressions++
			continue
		}

		if currentCov < baselineCov-coverageTolerance {
			// Always print regressions, even in quiet mode
			fmt.Printf("%s❌ REGRESSION: %s%s\n", colorRed, pkg, colorReset)
			fmt.Printf("%s   Coverage dropped from %.1f%% to %.1f%%%s\n",
				colorRed, baselineCov, currentCov, colorReset)
			regressions++
		} else if currentCov > baselineCov+coverageTolerance {
			// Coverage improved
			if verbose {
				fmt.Printf("%s📈 %s: %.1f%% (baseline: %.1f%%, +%.1f%%)%s\n",
					colorGreen, pkg, currentCov, baselineCov, currentCov-baselineCov, colorReset)
			}
			improvements++
			improvementList = append(improvementList, improvement{
				name:     pkg,
				baseline: baselineCov,
				current:  currentCov,
			})
			passed++
		} else {
			if verbose {
				fmt.Printf("%s✅ %s: %.1f%% (baseline: %.1f%%)%s\n",
					colorGreen, pkg, currentCov, baselineCov, colorReset)
			}
			passed++
		}
	}

	if verbose {
		fmt.Println()
		fmt.Printf("Baseline check: %d passed, %d regressions", passed, regressions)
		if improvements > 0 {
			fmt.Printf(", %d improvements", improvements)
		}
		fmt.Println()
		fmt.Println()
	}

	return passed, regressions, improvements, improvementList
}

// checkNewPackages checks that new packages meet minimum coverage requirements
func checkNewPackages(baseline, current map[string]float64) (passed, failed int, newPackages []packageCoverage) {
	if verbose {
		fmt.Println("Checking new packages for minimum coverage...")
		fmt.Println("----------------------------------------------")
	}

	foundNew := false
	for pkg, cov := range current {
		// Skip if in baseline
		if _, inBaseline := baseline[pkg]; inBaseline {
			continue
		}

		// Skip if package should be excluded
		if shouldExclude(pkg) {
			continue
		}

		foundNew = true

		if cov < minNewPkgCoverage-coverageTolerance {
			// Always print failures, even in quiet mode
			fmt.Printf("%s❌ NEW PACKAGE: %s has %.1f%% coverage%s\n",
				colorRed, pkg, cov, colorReset)
			fmt.Printf("%s   Minimum required: %.1f%%%s\n",
				colorRed, minNewPkgCoverage, colorReset)
			failed++
		} else {
			if verbose {
				fmt.Printf("%s✅ NEW PACKAGE: %s has %.1f%% coverage (meets %.1f%% minimum)%s\n",
					colorGreen, pkg, cov, minNewPkgCoverage, colorReset)
			}
			newPackages = append(newPackages, packageCoverage{
				name:     pkg,
				coverage: cov,
			})
			passed++
		}
	}

	if verbose && !foundNew {
		fmt.Println("No new packages found")
	}

	return passed, failed, newPackages
}

// shouldExclude determines if a package should be excluded from new package checks
func shouldExclude(pkg string) bool {
	// Exclude cmd/, hack/, tools, proto/, mock/, *abi packages (generated code), root node package
	excludePatterns := []string{
		"/cmd/",
		"/cmd$",
		"/hack/",
		"/tools$",
		"/proto/",
		"/mock/",
		"/mock$",
		"abi$",
	}

	// Special case: root node package
	if pkg == "github.com/certusone/wormhole/node" {
		return true
	}

	for _, pattern := range excludePatterns {
		matched, _ := regexp.MatchString(pattern, pkg)
		if matched {
			return true
		}
	}

	return false
}

// writeUpdatedBaseline writes the current coverage as a sorted baseline file.
func writeUpdatedBaseline(current map[string]float64) error {
	packages := make([]string, 0, len(current))
	for pkg := range current {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	// Create a temporary file in the same directory as the baseline to avoid cross-device link issues
	tempFile, err := os.CreateTemp(".", ".coverage-baseline-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())

	writer := bufio.NewWriter(tempFile)
	fmt.Fprintln(writer, "# Wormhole Go Coverage Baseline")
	fmt.Fprintln(writer, "# Auto-generated by coverage-check")
	fmt.Fprintln(writer, "# Format: <package> <coverage-percentage>")
	fmt.Fprintln(writer, "")

	for _, pkg := range packages {
		fmt.Fprintf(writer, "%s %.1f\n", pkg, current[pkg])
	}

	if err := writer.Flush(); err != nil {
		return err
	}
	tempFile.Close()

	// Replace original file with updated file
	if err := os.Rename(tempFile.Name(), baselineFile); err != nil {
		return err
	}

	return nil
}
