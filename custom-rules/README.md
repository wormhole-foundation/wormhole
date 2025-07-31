# Wormhole Custom Revive Rules

This directory contains custom linting rules for the Wormhole project using the revive framework.

## Custom Rules

### already-locked-checker

Detects functions with "alreadyLocked" in their name (case insensitive) and ensures they are called within proper mutex lock/unlock blocks.

**What it checks:**
- Functions containing "alreadyLocked" in their name
- Verifies the calling function has mutex locking patterns:
  - `mutex.Lock()` + `mutex.Unlock()` 
  - `mutex.Lock()` + `defer mutex.Unlock()`

**Example violations:**
```go
// BAD: No mutex locking
func badFunction() {
    obj.processDataAlreadyLocked("key") // Will be flagged
}

// GOOD: Proper mutex locking
func goodFunction() {
    obj.mu.Lock()
    defer obj.mu.Unlock()
    obj.processDataAlreadyLocked("key") // Will NOT be flagged
}
```

## Usage

### Build the Custom Revive Binary

```bash
cd custom-rules
go build -o wormhole-revive .
```

### Run with Custom Rules

```bash
# Test on a single file
./wormhole-revive -config revive.toml path/to/file.go

# Test on a directory
./wormhole-revive -config revive.toml ../node/pkg/accountant/

# Use different formatter
./wormhole-revive -config revive.toml -formatter stylish ../node/pkg/
```

### Integration with Existing Workflow

#### Add to Makefile
```makefile
lint: lint-standard lint-custom

lint-standard:
	golangci-lint run

lint-custom:
	cd custom-rules && ./wormhole-revive -config revive.toml ../node/pkg/
```

## Configuration

The `revive.toml` file configures which rules are enabled and which files to exclude:

```toml
[rule.already-locked-checker]
  severity = "error"  # Make violations fail the build
  # Exclude files using patterns
  Exclude = ["**/*_test.go", "**/testdata/**", "**/mock*.go"]

[rule.var-naming]
  arguments = [["ID"], ["VM"]]
  severity = "warning"
  # Example: exclude generated files
  Exclude = ["**/*.pb.go", "**/*_generated.go"]
```

### File Exclusion Patterns

You can exclude files from rules using various patterns in the `Exclude` array:

1. **Glob patterns**: `"**/*_test.go"` - exclude all test files
2. **Directory patterns**: `"**/testdata/**"` - exclude testdata directories  
3. **Wildcard patterns**: `"**/mock*.go"` - exclude mock files
4. **Regex patterns**: `"~\.(pb|auto|generated)\.go$"` - exclude generated files
5. **Specific files**: `"path/to/specific/file.go"` - exclude individual files
6. **Well-known patterns**: `"TEST"` - same as `**/*_test.go`

### Common Exclusion Examples

```toml
# Exclude test files, generated code, and vendor directories
[rule.already-locked-checker]
  Exclude = [
    "**/*_test.go",           # Test files
    "**/*.pb.go",             # Protocol buffer generated files
    "**/*_generated.go",      # Generated Go files
    "**/vendor/**",           # Vendor dependencies
    "**/testdata/**",         # Test data directories
    "**/mock*.go",            # Mock files
    "~\.(pb|auto|generated)\.go$"  # Regex for generated files
  ]
```

## Adding More Custom Rules

To add additional custom rules:

1. Create a new rule struct implementing the `lint.Rule` interface
2. Add it to `main.go` using `revivelib.NewExtraRule()`
3. Configure it in `revive.toml`
4. Rebuild the binary

Example:
```go
// In main.go
cli.RunRevive(
    revivelib.NewExtraRule(&AlreadyLockedRule{}, lint.RuleConfig{}),
    revivelib.NewExtraRule(&MyNewRule{}, lint.RuleConfig{}),
)
```

## Why revive?

Revive was selected mainly because we get the following benefits:
- it's well-maintained
- it supports AST-based rule creation
- rules can be done in pure Go

This is in contrast to a few other linters:
- CodeQL requires a heavy indexing step and a steep learning curve
- Semgrep isn't great for complex rules and requires learning its YAML syntax
