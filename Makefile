# Makefile for production builds. This is not meant, or optimized, for incremental or debug builds. Use the devnet for
# development. For the sake of decentralization, we specifically avoid the use of prebuilt containers wherever possible
# to increase diversity - operators sourcing their compiler binaries from different sources is a good thing.

SHELL = /usr/bin/env bash
MAKEFLAGS += --no-builtin-rules

PREFIX ?= /usr/local
OUT = build
BIN = $(OUT)/bin
UNIT_DOCKER_IMAGE ?= wormhole-unit-test
UNIT_DOCKER_RUN = docker run --rm --workdir /app --user "$$(id -u):$$(id -g)" --mount type=bind,source=$(CURDIR),target=/app $(UNIT_DOCKER_IMAGE)

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

.PHONY: lint-docker
## Run Go lints in Docker
lint-docker:
	bash scripts/lint.sh -c lint

.PHONY: lint-rust
lint-rust:
# Runs clippy for most Rust crates.
	bash scripts/clippy.sh

.PHONY: build-unit-docker
## Build the unit test container used for node/sdk Go tests
build-unit-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.unit -t $(UNIT_DOCKER_IMAGE) .

.PHONY: test-unit
## Run node and sdk Go unit tests
test-unit:
	@echo "Running unit tests for node and sdk..."
	@cd node && go test -count=1 -v -timeout 5m -race ./...
	@cd sdk && go test -count=1 -v -timeout 5m -race ./...

.PHONY: test-docker
## Run node and sdk Go unit tests in Docker
test-docker: build-unit-docker
	$(UNIT_DOCKER_RUN) make test-unit

.PHONY: node
## Build guardiand binary
node: $(BIN)/guardiand

.PHONY: $(BIN)/guardiand
$(BIN)/guardiand: CGO_ENABLED=1
$(BIN)/guardiand: dirs generate
	cd node && go build -ldflags "-X github.com/certusone/wormhole/node/pkg/version.version=${VERSION}" \
	  -mod=readonly -o ../$(BIN)/guardiand \
	  github.com/certusone/wormhole/node

.PHONY: test-coverage
## Run tests with coverage for node and sdk (matches CI)
test-coverage:
	@echo "Running tests with coverage for node and sdk..."
	@set -o pipefail && (cd node && go test -count=1 -v -timeout 5m -race -cover ./...) 2>&1 | tee coverage.txt
	@set -o pipefail && (cd sdk && go test -count=1 -v -timeout 5m -race -cover ./...) 2>&1 | tee -a coverage.txt

.PHONY: test-coverage-docker
## Run test-coverage in Docker
test-coverage-docker: build-unit-docker
	$(UNIT_DOCKER_RUN) make test-coverage

.PHONY: check-coverage
## Check coverage against baseline (run tests first)
check-coverage: build-coverage-check test-coverage
	@./coverage-check

.PHONY: check-coverage-verbose
## Check coverage against baseline (run tests first)
check-coverage-verbose: build-coverage-check test-coverage
	@./coverage-check -v

.PHONY: build-coverage-check
## Build the coverage checker tool
build-coverage-check:
	@cd scripts/coverage-check && go build -o ../../coverage-check .

.PHONY: update-coverage-baseline
## Update coverage baseline with current coverage
update-coverage-baseline: build-coverage-check test-coverage
	@./coverage-check -u
