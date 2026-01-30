# Go Coverage Check Tool

A minimal Go tool to enforce incremental test coverage improvements without blocking development.

## What It Does

This tool enforces two simple rules:

1. **No regression**: Packages in the baseline file must maintain their current coverage (within 0.1% tolerance)
2. **New packages need tests**: Any new package in `pkg/` must have at least 10% test coverage

## How It Works

### Baseline File

The baseline file (`node/.coverage-baseline`) lists packages and their minimum allowed coverage:

```
# Format: <package> <coverage-percentage>
github.com/certusone/wormhole/node/pkg/db 73.8
github.com/certusone/wormhole/node/pkg/query 69.9
...
```

### What Gets Checked

**Baseline packages**: Coverage must not drop below the baseline threshold
**New packages**: Any package in `pkg/` that's not in the baseline must have ≥10% coverage

### What's Excluded

The following are **excluded** from new package checks (but can still be in the baseline):
- `cmd/` - Command-line binaries
- `hack/` - Utility scripts
- `tools/` - Tooling packages
- `proto/` - Generated protobuf code
- `mock/` - Mock packages
- `*abi` - Generated ABI bindings

## Usage

### Run Locally

**Easy way (recommended):**
```bash
# Build the tool once
make build-coverage-check

# Run tests and check coverage
make check-coverage

# Or with verbose output
make test-coverage && ./coverage-check -v
```

**Manual way:**
```bash
# Step 1: Run tests with coverage and save output (both node and sdk)
(cd node && go test -cover ./...; cd ../sdk && go test -cover ./...) 2>&1 | tee coverage.txt

# Step 2: Build the coverage checker (one-time)
cd scripts/coverage-check && go build -o ../../coverage-check . && cd ../..

# Step 3: Check coverage against baseline
./coverage-check          # Quiet mode (only shows failures)
./coverage-check -v       # Verbose mode (shows all checks)
```

### CI Integration

The tool runs automatically in GitHub Actions (`.github/workflows/build.yml`):

```yaml
- name: Run golang tests with coverage (node)
  run: cd node && go test -v -timeout 5m -race -cover ./... 2>&1 | tee ../coverage-node.tmp
- name: Run golang tests with coverage (sdk)
  run: cd sdk && go test -v -timeout 5m -cover ./... 2>&1 | tee ../coverage-sdk.tmp
- name: Combine coverage output
  run: cat coverage-node.tmp coverage-sdk.tmp > coverage.txt && rm coverage-node.tmp coverage-sdk.tmp
- name: Build coverage check tool
  run: cd scripts/coverage-check && go build -o ../../coverage-check .
- name: Check coverage against baseline
  run: ./coverage-check
```

**Note**: The tool reads from `coverage.txt` at repo root, which must be generated first by `go test -cover` on both `node/` and `sdk/`.

## Common Scenarios

### 💡 Scenario 1: Adding tests to an existing package

```
# Before: pkg/foo has 50% coverage in baseline
# You add tests and coverage goes to 55%
Result: 💡 Coverage improved! Update baseline with: make update-coverage-baseline
Exit code: 1 (forces you to update)
```

### ❌ Scenario 2: Coverage drops accidentally

```
# Before: pkg/foo has 50% coverage in baseline
# You refactor and coverage drops to 45%
Result: ❌ FAIL - coverage regression detected
Fix: Add tests to restore coverage to ≥50%
```

### ✅ Scenario 3: Intentional coverage drop

```
# Before: pkg/foo has 50% coverage in baseline
# You remove dead code, coverage drops to 48%
Result: ❌ FAIL - coverage regression
Fix: Run `make update-coverage-baseline` and commit with explanation
```

### ❌ Scenario 4: New package without tests

```
# You add pkg/feature/ with code but no tests
Result: ❌ FAIL - new package has 0% coverage (minimum: 10%)
Fix: Add basic tests to reach ≥10% coverage
```

### 💡 Scenario 5: New package with basic tests

```
# You add pkg/feature/ with 15% test coverage
Result: 💡 New package detected! Update baseline with: make update-coverage-baseline
Exit code: 1 (forces you to add to baseline)
Note: After update, 15% becomes the baseline for this package
```

### ✅ Scenario 6: Adding cmd/ or hack/ code

```
# You add cmd/tool/ with no tests
Result: ✅ PASS - cmd/ packages are excluded from checks
```

## Updating the Baseline

### When to Update

The tool will **require** you to update the baseline when:
1. You **improve** coverage (tool exits with code 1, forces update to lock in gains)
2. You add a new package with tests (tool exits with code 1, forces adding to baseline)
3. You **intentionally** reduce coverage (e.g., removing dead code)

### How to Update

**Automatic way (recommended):**
```bash
make update-coverage-baseline
# Or: ./coverage-check -u
```

This updates the baseline file automatically while preserving comments and structure.

**Manual way (if needed):**
Edit `.coverage-baseline` directly:

```bash
# Lower the threshold for a specific package
# Before:
github.com/certusone/wormhole/node/pkg/foo 50.0

# After (with justification in commit message):
github.com/certusone/wormhole/node/pkg/foo 45.0
```

### Process

1. Edit `node/.coverage-baseline`
2. Run `./coverage-check` locally to verify
3. Commit with explanation: `"coverage: lower pkg/foo baseline to 45% due to removing deprecated code"`

## Adding New Packages to Baseline

When you add substantial tests to a previously untested package:

```bash
# If pkg/supervisor currently has 0% in baseline and you add tests:
# 1. Add tests, achieving say 25% coverage
# 2. Run ./coverage-check - it will pass (improvement over 0%)
# 3. Update baseline to lock in the new coverage:

# In node/.coverage-baseline:
github.com/certusone/wormhole/node/pkg/supervisor 25.0  # Was 0.0
```

## Configuration

Edit constants in `scripts/coverage-check/main.go`:

```go
const (
    baselineFile       = ".coverage-baseline"        // Baseline file location (repo root)
    coverageOutputFile = "coverage.txt"              // Where to read test coverage from (repo root)
    minNewPkgCoverage  = 10.0                        // Minimum for new packages
    coverageTolerance  = 0.1                         // Floating point tolerance
)
```

### Command-line Flags

```bash
./coverage-check          # Quiet mode (only shows failures and improvements)
./coverage-check -v       # Verbose mode (shows all checks)
./coverage-check --verbose # Same as -v
./coverage-check -u       # Update baseline with current coverage
./coverage-check --update # Same as -u
```

**Important**: The tool will exit with code 1 if coverage improved, prompting you to update the baseline. This is intentional - it forces you to lock in improvements!

**Note**: The tool expects `coverage.txt` at repo root. Generate it with:
```bash
make test-coverage
# Or manually:
(cd node && go test -cover ./...; cd ../sdk && go test -cover ./...) 2>&1 | tee coverage.txt
```

## FAQ

**Q: Why 10% minimum for new packages?**
A: It's low enough to not block development, but high enough to ensure at least basic test coverage exists.

**Q: What if I need to ship urgently without tests?**
A: Add the package to the baseline with 0% coverage. File a follow-up issue to add tests.

**Q: Can I increase the minimum coverage over time?**
A: Yes! Edit `minNewPkgCoverage` in `main.go`. Existing packages are grandfathered in via the baseline.

**Q: What about integration tests?**
A: This tool only measures unit test coverage (`go test ./...`). Integration tests are valuable but not tracked here.

**Q: Why not enforce total coverage percentage?**
A: Total coverage can drop as you add new code, which would block development. Per-package baselines are more granular and fair.

## Troubleshooting

### "No coverage data found" warning

This usually means:
- Package was renamed → Update baseline with new name
- Package was deleted → Remove from baseline
- Tests are skipped in CI → Check test build tags

### Coverage check fails locally but passes in CI

Ensure you're running from repo root:
```bash
cd /path/to/wormhole2-lol
./coverage-check
```

### "Minimum required: 10.0%" for generated code

If a package contains only generated code (proto, abi), add it to the exclusion list in `shouldExclude()`:

```go
excludePatterns := []string{
    "/cmd/",
    "/proto/",
}
```

## How It Works

The tool is intentionally simple:

1. **Input**: Reads `node/coverage.txt` (generated by `go test -cover ./... | tee coverage.txt`)
2. **Parse**: Extracts package coverage percentages using regex
3. **Compare**: Checks against baseline file (`node/.coverage-baseline`)
4. **Report**: Prints pass/fail for each package with clear error messages

**No side effects**: The tool only reads files, never runs tests or modifies anything.

## Development

The tool is a single Go file with no dependencies beyond the standard library:

```bash
scripts/coverage-check/
├── main.go      # All logic here (~250 lines)
├── go.mod       # Module declaration
└── README.md    # This file
```

To modify behavior, edit `main.go` and rebuild:

```bash
cd scripts/coverage-check
go build -o ../../coverage-check .
```

**Key functions:**
- `parseCoverageOutput()` - Reads `node/coverage.txt` and extracts percentages
- `loadBaseline()` - Parses `node/.coverage-baseline`
- `checkBaseline()` - Compares current vs baseline
- `checkNewPackages()` - Enforces minimum coverage for new packages
- `shouldExclude()` - Determines what to skip (cmd/, proto/, etc.)

## Design Principles

1. **No external dependencies** - Uses only Go stdlib
2. **Fail fast, fail clear** - Shows exactly what failed and how to fix it
3. **Grandfathering** - Existing code keeps current coverage, new code must meet minimum
4. **Developer-friendly** - Low friction, clear error messages, easy to override when needed
5. **Maintainable** - Small, simple Go code the team can easily modify
