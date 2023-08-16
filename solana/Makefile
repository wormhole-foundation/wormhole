SOLANA_CLI="v1.14.22"
ANCHOR_CLI="v0.28.0"

out_mainnet=artifacts-mainnet
out_testnet=artifacts-testnet
out_localnet=artifacts-localnet

.PHONY: all clean check build test lint

all: check

clean:
	anchor clean
	rm -rf node_modules artifacts-mainnet artifacts-testnet artifacts-localnet

node_modules:
	npm ci

check:
	cargo check --features "localnet" --no-default-features
	cargo check --features "mainnet" --no-default-features
	cargo check --features "testnet" --no-default-features

build: $(out_$(NETWORK))
$(out_$(NETWORK)):
ifdef out_$(NETWORK)
	anchor build --arch sbf -- --features "$(NETWORK)" -- --no-default-features
	mkdir -p $(out_$(NETWORK))
	cp target/deploy/*.so $(out_$(NETWORK))/
endif

test: node_modules
	cargo test
	NETWORK=mainnet $(MAKE) build
	anchor test --arch sbf

lint:
	cargo fmt --check
	cargo clippy --no-deps --all-targets -- -D warnings

docker-wip:
ifdef out_$(NETWORK)
	DOCKER_BUILDKIT=1 docker build -f Dockerfile \
		--build-arg SOLANA_CLI=$(SOLANA_CLI) \
		--build-arg ANCHOR_CLI=$(ANCHOR_CLI) \
		--build-arg NETWORK=$(NETWORK) .
endif
