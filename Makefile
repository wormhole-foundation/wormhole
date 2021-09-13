# Makefile for production builds. This is not meant, or optimized, for incremental or debug builds. Use the devnet for
# development. For the sake of decentralization, we specifically avoid the use of prebuilt containers wherever possible
# to increase diversity - operators sourcing their compiler binaries from different sources is a good thing.

SHELL = /usr/bin/env bash
MAKEFLAGS += --no-builtin-rules

PREFIX ?= /usr/local
OUT = build
BIN = $(OUT)/bin

VERSION = $(shell git describe --tags --dirty)

.PHONY: dirs
dirs: Makefile
	@mkdir -p $(BIN)

.PHONY: install
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
node: $(BIN)/guardiand

.PHONY: $(BIN)/guardiand
$(BIN)/guardiand: dirs generate
	cd node && go build -ldflags "-X github.com/certusone/wormhole/node/pkg/version.version=${VERSION}" \
	  -mod=readonly -o ../$(BIN)/guardiand \
	  github.com/certusone/wormhole/node
