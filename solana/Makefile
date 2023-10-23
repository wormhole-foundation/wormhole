SOLANA_CLI="v1.16.16"
ANCHOR_CLI="v0.28.0"

out_mainnet=artifacts-mainnet
out_testnet=artifacts-testnet
out_localnet=artifacts-localnet

.PHONY: all clean check build test lint prune_idl ci

all: check

check:
	cargo check --all-features

clean:
	anchor clean
	rm -rf node_modules artifacts-mainnet artifacts-testnet artifacts-localnet tests/artifacts

node_modules:
	npm ci

prune_idl: node_modules scripts/prune_idl_types.ts
	cd scripts && npx ts-node prune_idl_types.ts

build: $(out_$(NETWORK))
$(out_$(NETWORK)):
ifdef out_$(NETWORK)
	anchor build -p wormhole_core_bridge_solana --arch sbf -- --features "$(NETWORK),no-idl" -- --no-default-features
	anchor build -p wormhole_token_bridge_solana --arch sbf -- --features "$(NETWORK),no-idl" -- --no-default-features
	$(MAKE) prune_idl
	mkdir -p $(out_$(NETWORK))
	cp target/deploy/*.so $(out_$(NETWORK))/
endif

test: node_modules
	cargo test --all-features
	anchor build -p wormhole_core_bridge_solana --arch sbf -- --features mainnet -- --no-default-features
	anchor build -p wormhole_token_bridge_solana --arch sbf -- --features mainnet -- --no-default-features
	mkdir -p tests/artifacts && cp target/deploy/*.so tests/artifacts/
	anchor build --arch sbf -- --features localnet -- --no-default-features
	$(MAKE) prune_idl
	anchor test --skip-build

lint:
	cargo fmt --check
	cargo clippy --no-deps --all-targets --all-features -- -D warnings

ci:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.ci \
		--build-arg SOLANA_CLI=$(SOLANA_CLI) \
		--build-arg ANCHOR_CLI=$(ANCHOR_CLI) \
		.
