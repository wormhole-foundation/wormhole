out_mainnet=artifacts-mainnet
out_testnet=artifacts-testnet
out_localnet=artifacts-localnet

.PHONY: all clean build test lint

all: test

clean:
	anchor clean
	rm -rf node_modules artifacts-mainnet artifacts-testnet artifacts-localnet

node_modules:
	npm ci

build: $(out_$(NETWORK))

$(out_$(NETWORK)):
ifdef out_$(NETWORK)
	anchor build --arch sbf -- --features "$(NETWORK)" -- --no-default-features
	mkdir -p $(out_$(NETWORK))
	cp target/deploy/*.so $(out_$(NETWORK))/
endif

test: node_modules
	anchor test --arch sbf

lint:
	cargo fmt --check
	cargo check --features "localnet" --no-default-features
	cargo check --features "mainnet" --no-default-features
	cargo check --features "testnet" --no-default-features
	cargo clippy --no-deps --all-targets -- -D warnings