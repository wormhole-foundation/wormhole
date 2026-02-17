# Makefile for production builds. This is not meant, or optimized, for incremental or debug builds. Use the devnet for
# development. For the sake of decentralization, we specifically avoid the use of prebuilt containers wherever possible
# to increase diversity - operators sourcing their compiler binaries from different sources is a good thing.

SHELL = /usr/bin/env bash
MAKEFLAGS += --no-builtin-rules

PREFIX ?= /usr/local
OUT = build
BIN = $(OUT)/bin

-include Makefile.help

# VERSION is the git tag of the current commit if there's a tag, otherwise it's "dev-" plus the git commit sha.
VERSION = $(shell git describe --tags --dirty 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD)")

.PHONY: dirs
dirs: Makefile
	@mkdir -p $(BIN)

.PHONY: install
## Install guardiand binary
install:
	install -m 775 $(BIN)/* $(PREFIX)/bin
	setcap cap_ipc_lock=+ep $(PREFIX)/bin/guardiand

.PHONY: generate
generate: dirs
	cd tools && ./build.sh
	rm -rf bridge
	rm -rf node/pkg/proto
	tools/bin/buf generate

.PHONY: lint
lint:
# Lints spelling and Go
	bash scripts/lint.sh lint

.PHONY: lint-rust
lint-rust:
# Runs clippy for most Rust crates.
	bash scripts/clippy.sh

.PHONY: node
## Build guardiand binary
node: $(BIN)/guardiand

.PHONY: $(BIN)/guardiand
$(BIN)/guardiand: CGO_ENABLED=1
$(BIN)/guardiand: dirs generate
	cd node && go build -ldflags "-X github.com/certusone/wormhole/node/pkg/version.version=${VERSION}" \
	  -mod=readonly -o ../$(BIN)/guardiand \
	  github.com/certusone/wormhole/node

# ── Coverage targets ──────────────────────────────────────────────
# All coverage targets use the coverage-* prefix.
# Tab-complete "make coverage" to discover them.
#
#   coverage         — run tests + check against baseline (main entry point)
#   coverage-init    — create baseline from scratch (first-time setup)
#   coverage-update  — update baseline after improvements
#   coverage-test    — just run tests with coverage, no baseline check
#
# Use VERBOSE=1 for detailed output:  VERBOSE=1 make coverage
COVERAGE_FLAGS := $(if $(VERBOSE),-v,)

# Internal: build the coverage checker binary (not shown in help)
.PHONY: build-coverage-check
build-coverage-check:
	@cd scripts/coverage-check && go build -o ../../coverage-check .

.PHONY: coverage-test
## Run tests with coverage for node/ and sdk/ (no baseline check)
coverage-test:
	@echo "Running tests with coverage for node and sdk..."
	@(cd node && go test -v -timeout 5m -race -cover ./...) 2>&1 | tee coverage.txt
	@(cd sdk && go test -v -timeout 5m -race -cover ./...) 2>&1 | tee -a coverage.txt

.PHONY: coverage
## Run tests and check coverage against baseline
coverage: build-coverage-check coverage-test
	@./coverage-check $(COVERAGE_FLAGS)

.PHONY: coverage-init
## Create coverage baseline from scratch (first-time setup)
coverage-init: build-coverage-check coverage-test
	@./coverage-check -init

.PHONY: coverage-update
## Update coverage baseline with current values
coverage-update: build-coverage-check coverage-test
	@./coverage-check -u
