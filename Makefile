# Makefile for production builds. This is not meant, or optimized, for incremental or debug builds. Use the devnet for
# development. For the sake of decentralization, we specifically avoid the use of prebuilt containers wherever possible
# to increase diversity - operators sourcing their compiler binaries from different sources is a good thing.

SHELL = /usr/bin/env bash
MAKEFLAGS += --no-builtin-rules

PREFIX ?= /usr/local
OUT = build
BIN = $(OUT)/bin

-include Makefile.help

VERSION = $(shell git describe --tags --dirty)

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

.PHONY: node
## Build guardiand binary
node: $(BIN)/guardiand

.PHONY: $(BIN)/guardiand
$(BIN)/guardiand: CGO_ENABLED=1
$(BIN)/guardiand: dirs generate
	@# The go-ethereum and celo-blockchain packages both implement secp256k1 using the exact same header, but that causes duplicate symbols.
	cd node && go build -ldflags "-X github.com/certusone/wormhole/node/pkg/version.version=${VERSION} -extldflags -Wl,--allow-multiple-definition" \
	  -mod=readonly -o ../$(BIN)/guardiand \
	  github.com/certusone/wormhole/node
