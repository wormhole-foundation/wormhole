# Chain ids
# ---------
CHAIN_ID_solana=1
CHAIN_ID_fogo=51

# Solana addresses
# ----------------

# Mainnet buffer authority is the "upgrade" PDA
bridge_ADDRESS_solana_mainnet=worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth
bridge_AUTHORITY_solana_mainnet=2rCAC1VKz5YP1jZTHcVfWDhHMs2iEruUaATdeZe5Fjk5
token_bridge_ADDRESS_solana_mainnet=wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb
token_bridge_AUTHORITY_solana_mainnet=DHyAcRbFpRWTkcsAsfwQpbABXvtjs6bQ1dq5ScNhRDoQ
nft_bridge_ADDRESS_solana_mainnet=WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD
nft_bridge_AUTHORITY_solana_mainnet=3cVZHphy4QUYnU1hYFyvHF9joeZJ6ZTxpWx1nzavaUa8

# Testnet buffer authority is the deployer public key
bridge_ADDRESS_solana_testnet=3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5
bridge_AUTHORITY_solana_testnet=J8am6SkUHRTtLPJpnfUd6Uy38U7Yh17fa7ZtiqaLoJcV
token_bridge_ADDRESS_solana_testnet=DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe
token_bridge_AUTHORITY_solana_testnet=FQAHqBcVHiiiLP8qXKPDQGr3mEXLv7RSdvfHJ3ZLugBV
nft_bridge_ADDRESS_solana_testnet=2rHhojZ7hpu1zA91nvZmT8TqWWvMcKmmNBCr2mKTtMq4
nft_bridge_AUTHORITY_solana_testnet=9r6q2iEg4MBevjC8reaLmQUDxueF3vabUoqDkZ2LoAYe

# Devnet buffer authority is the devnet public key
bridge_ADDRESS_solana_devnet=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
bridge_AUTHORITY_solana_devnet=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
token_bridge_ADDRESS_solana_devnet=B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE
token_bridge_AUTHORITY_solana_devnet=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
nft_bridge_ADDRESS_solana_devnet=NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA
nft_bridge_AUTHORITY_solana_devnet=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J

# Fogo SVM addresses
# -------------------

# Testnet
bridge_ADDRESS_fogo_testnet=BhnQyKoQQgpuRTRo6D8Emz93PvXCYfVgHhnrR4T3qhw4
bridge_AUTHORITY_fogo_testnet=34NzRtyigDRthGuWTCcaVRN6sHjkJc86fFNkqjvtghXs
token_bridge_ADDRESS_fogo_testnet=78HdStBqCMioGii9D8mF3zQaWDqDZBQWTUwjjpdmbJKX
token_bridge_AUTHORITY_fogo_testnet=CDZR3RgCs21mJS2uYKFmof24iiryxJAb8R2Wk9Rgg3Pw

# Mainnet
bridge_ADDRESS_fogo_mainnet=worm2mrQkG1B1KTz37erMfWN8anHkSK24nzca7UD8BB
bridge_AUTHORITY_fogo_mainnet=CAb6rWkJAMMX4feJQgJZvBg2sK5Z5Tn2i48njRhSL9sw
token_bridge_ADDRESS_fogo_mainnet=wormQuCVWSSmPdjVmEzAWxAXViVyTSWnLyhff5hVYGS
token_bridge_AUTHORITY_fogo_mainnet=BAaVJ8fzpw8WJJqER9LK5ezTvxmG5VvFRgAWh21jJchz

# Fogo specific RPC URLs.
# for testnet we use the alt endpoint because of solana-cli issues with certs
# at the time of writing.
FOGO_RPC_URL_MAINNET=https://mainnet.fogo.io
FOGO_RPC_URL_TESTNET=https://testnet-alt.fogo.io

SOURCE_FILES:=$(shell find . -name "*.rs" -or -name "*.lock" -or -name "*.toml" | grep -v "target") Dockerfile

.PHONY: clean all help artifacts deploy/bridge deploy/token_bridge deploy/nft_bridge .FORCE fmt check clippy test

-include ../Makefile.help

.FORCE:

# concrete target to parse SVM and NETWORK from the goal name
artifacts-%:
	$(eval SVM := $(word 1,$(subst -, ,$*)))
	$(eval NETWORK := $(word 2,$(subst -, ,$*)))
	echo $(SVM)
	@$(MAKE) NETWORK=$(NETWORK) SVM=$(SVM) artifacts

.PHONY: check-network
check-network:
ifndef bridge_ADDRESS_$(SVM)_$(NETWORK)
	$(error Invalid or missing NETWORK and/or SVM. Please call with `$(MAKE) $(MAKECMDGOALS) SVM=[solana | fogo] NETWORK=[mainnet | testnet | devnet]`)
endif

check-svmchain-name:
	@if [ -z "$(SVM)" ] || [ "$(filter $(SVM), solana fogo)" != "$(SVM)" ]; then \
		echo "Error: Invalid or missing SVM. Valid options are 'solana' and 'fogo'"; \
		exit 1; \
	fi

artifacts: check-svmchain-name check-network $(SOURCE_FILES)
	echo $@
	@echo "Building artifacts for ${SVM} ${NETWORK} (${bridge_ADDRESS_$${SVM}_${NETWORK}})"
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg BRIDGE_ADDRESS=${bridge_ADDRESS_${SVM}_${NETWORK}} --build-arg CHAIN_ID=${CHAIN_ID_${SVM}} -o $@ .
	cd $@ && ls | xargs sha256sum > checksums.txt

payer-$(SVM)-$(NETWORK).json:
	$(error Missing private key in payer-$(SVM)-$(NETWORK).json)

%-buffer-$(SVM)-$(NETWORK).txt: payer-$(SVM)-$(NETWORK).json check-network
	$(eval FLAG := $(shell echo 	$(if $(filter fogo,$(SVM)),\
						$(if $(filter mainnet,$(NETWORK)),	$(FOGO_RPC_URL_MAINNET),\
						$(if $(filter testnet,$(NETWORK)),	$(FOGO_RPC_URL_TESTNET),\
													l)),\
					$(if $(filter mainnet,$(NETWORK)),	m,\
					$(if $(filter testnet,$(NETWORK)),	d,\
									l)))))
	solana -k payer-${SVM}-${NETWORK}.json program write-buffer artifacts-$(SVM)-$(NETWORK)/$*.so -u $(FLAG) | cut -f2 -d' ' > $@
	solana -k payer-${SVM}-${NETWORK}.json program set-buffer-authority $$(cat $@) --new-buffer-authority $($*_AUTHORITY_$(SVM)_$(NETWORK)) -u $(FLAG)

## Deploy core bridge program.
deploy/bridge: bridge-buffer-$(SVM)-$(NETWORK).txt
	@echo Deployed core bridge contract at:
	@cat $<

## Deploy token bridge program.
deploy/token_bridge: token_bridge-buffer-$(SVM)-$(NETWORK).txt
	@echo Deployed token bridge contract at:
	@cat $<

## Deploy nft bridge program.
deploy/nft_bridge: nft_bridge-buffer-$(SVM)-$(NETWORK).txt
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

test: $(SOURCE_FILES) check-svmchain-name
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg BRIDGE_ADDRESS=${bridge_ADDRESS_${SVM}_devnet} \
		--build-arg CHAIN_ID=${CHAIN_ID_${SVM}} \
		--build-arg EMITTER_ADDRESS=CiByUvEcx7w2HA4VHcPCBUAFQ73Won9kB36zW9VjirSr -o target/deploy .
	BPF_OUT_DIR=$(realpath $(dir $(firstword $(MAKEFILE_LIST))))/target/deploy \
		cargo test --workspace \
			--features "nft-bridge/instructions token-bridge/instructions wormhole-bridge-solana/instructions"

.PHONY: build
# Build for local use (not for deployment)
build: check-network check-svmchain-name
	CHAIN_ID=${CHAIN_ID_${SVM}} \
	BRIDGE_ADDRESS=${bridge_ADDRESS_${SVM}_${NETWORK}} \
	EMITTER_ADDRESS=11111111111111111111111111111115 \
	cargo build

clean:
	rm -rf artifacts-* *-buffer-*.txt
