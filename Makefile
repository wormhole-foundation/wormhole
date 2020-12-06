# Makefile for production builds. This is not meant, or optimized, for incremental or debug builds. Use the devnet for
# development. For the sake of decentralization, we specifically avoid the use of prebuilt containers wherever possible
# to increase diversity - operators sourcing their compiler binaries from different sources is a good thing.

SHELL = /usr/bin/env bash
MAKEFLAGS += --no-builtin-rules

PREFIX ?= /usr/local
OUT = build
BIN = $(OUT)/bin

.PHONY: dirs
dirs: Makefile
	@mkdir -p $(BIN)

.PHONY: install
install:
	install -m 775 $(BIN)/* $(PREFIX)/bin
	setcap cap_ipc_lock=+ep $(PREFIX)/bin/guardiand

.PHONY: generate
generate: dirs
	./generate-protos.sh

.PHONY: bridge
bridge: $(BIN)/guardiand

.PHONY: $(BIN)/guardiand
$(BIN)/guardiand: dirs
	cd bridge && go build -o mod=readonly -o ../$(BIN)/guardiand github.com/certusone/wormhole/bridge

.PHONY: agent
agent: $(BIN)/guardiand-solana-agent

.PHONY: $(BIN)/guardiand-solana-agent
$(BIN)/guardiand-solana-agent: dirs
	cd solana/agent && cargo build --release
	cp solana/target/release/agent $(BIN)/guardiand-solana-agent
