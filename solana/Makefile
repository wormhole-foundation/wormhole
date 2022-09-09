# Mainnet buffer authority is the "upgrade" PDA
bridge_ADDRESS_mainnet=worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth
bridge_AUTHORITY_mainnet=2rCAC1VKz5YP1jZTHcVfWDhHMs2iEruUaATdeZe5Fjk5
token_bridge_ADDRESS_mainnet=wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb
token_bridge_AUTHORITY_mainnet=DHyAcRbFpRWTkcsAsfwQpbABXvtjs6bQ1dq5ScNhRDoQ
nft_bridge_ADDRESS_mainnet=WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD
nft_bridge_AUTHORITY_mainnet=3cVZHphy4QUYnU1hYFyvHF9joeZJ6ZTxpWx1nzavaUa8

# Testnet buffer authority is the deployer public key
bridge_ADDRESS_testnet=3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5
bridge_AUTHORITY_testnet=9r6q2iEg4MBevjC8reaLmQUDxueF3vabUoqDkZ2LoAYe
token_bridge_ADDRESS_testnet=DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe
token_bridge_AUTHORITY_testnet=9r6q2iEg4MBevjC8reaLmQUDxueF3vabUoqDkZ2LoAYe
nft_bridge_ADDRESS_testnet=2rHhojZ7hpu1zA91nvZmT8TqWWvMcKmmNBCr2mKTtMq4
nft_bridge_AUTHORITY_testnet=9r6q2iEg4MBevjC8reaLmQUDxueF3vabUoqDkZ2LoAYe

# Devnet buffer authority is the devnet public key
bridge_ADDRESS_devnet=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
bridge_AUTHORITY_devnet=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
token_bridge_ADDRESS_devnet=B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE
token_bridge_AUTHORITY_devnet=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
nft_bridge_ADDRESS_devnet=NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA
nft_bridge_AUTHORITY_devnet=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J

SOURCE_FILES=$(shell find . -name "*.rs" -or -name "*.lock" -or -name "*.toml" | grep -v "target") Dockerfile

.PHONY: clean all help artifacts deploy/bridge deploy/token_bridge deploy/nft_bridge .FORCE fmt check clippy test

-include ../Makefile.help

.FORCE:

## Build contracts.
artifacts: check-network artifacts-$(NETWORK)

.PHONY: check-network
check-network:
ifndef bridge_ADDRESS_$(NETWORK)
	$(error Invalid or missing NETWORK. Please call with `$(MAKE) $(MAKECMDGOALS) NETWORK=[mainnet | testnet | devnet]`)
endif

artifacts-$(NETWORK): $(SOURCE_FILES)
	echo $@
	@echo "Building artifacts for ${NETWORK} (${bridge_ADDRESS_${NETWORK}})"
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg BRIDGE_ADDRESS=${bridge_ADDRESS_${NETWORK}} -o $@ .
	cd $@ && ls | xargs sha256sum > checksums.txt

payer-$(NETWORK).json:
	$(error Missing private key in payer-$(NETWORK).json)

%-buffer-$(NETWORK).txt: payer-$(NETWORK).json check-network
	$(eval FLAG := $(shell echo 	$(if $(filter mainnet,$(NETWORK)),	m,\
				   	$(if $(filter testnet,$(NETWORK)),	d,\
										l\
			))))
	solana -k payer-${NETWORK}.json program write-buffer artifacts-$(NETWORK)/$*.so -u $(FLAG) | cut -f2 -d' ' > $@
	solana -k payer-${NETWORK}.json program set-buffer-authority $$(cat $@) --new-buffer-authority $($*_AUTHORITY_$(NETWORK)) -u $(FLAG)

## Deploy core bridge program.
deploy/bridge: bridge-buffer-$(NETWORK).txt
	@echo Deployed core bridge contract at:
	@cat $<

## Deploy token bridge program.
deploy/token_bridge: token_bridge-buffer-$(NETWORK).txt
	@echo Deployed token bridge contract at:
	@cat $<

## Deploy nft bridge program.
deploy/nft_bridge: nft_bridge-buffer-$(NETWORK).txt
	@echo Deployed nft bridge contract at:
	@cat $<

.PHONY: wasm
## Build wasm
wasm: $(SOURCE_FILES)
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.wasm -o type=local,dest=$@ .
	cp -r $@/* ..

fmt: $(SOURCE_FILES)
	cargo fmt --check --all --manifest-path Cargo.toml

check: $(SOURCE_FILES)
	cargo check --workspace --tests --manifest-path Cargo.toml \
		--features "nft-bridge/instructions token-bridge/instructions wormhole-bridge-solana/instructions"

clippy: $(SOURCE_FILES)
	cargo clippy --workspace --tests --manifest-path Cargo.toml \
		--features "nft-bridge/instructions token-bridge/instructions wormhole-bridge-solana/instructions"

test: $(SOURCE_FILES)
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg BRIDGE_ADDRESS=${bridge_ADDRESS_devnet} \
		--build-arg EMITTER_ADDRESS=CiByUvEcx7w2HA4VHcPCBUAFQ73Won9kB36zW9VjirSr -o target/deploy .
	BPF_OUT_DIR=$(realpath $(dir $(firstword $(MAKEFILE_LIST))))/target/deploy \
		cargo test --workspace \
			--features "nft-bridge/instructions token-bridge/instructions wormhole-bridge-solana/instructions"

clean:
	rm -rf artifacts-mainnet artifacts-testnet artifacts-devnet *-buffer-*.txt

