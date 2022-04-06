bridge_SOURCE=wormhole
token_bridge_SOURCE=token_bridge
nft_bridge_SOURCE=nft_bridge

SOURCE_FILES=$(shell find . -name "*.rs" -or -name "*.lock" -or -name "*.toml" | grep -v target)

PACKAGES=$(shell find . -name "Cargo.toml" | grep -E 'packages|contracts' | cut -d/ -f3 | sed s/-/_/g)
WASMS=$(patsubst %, artifacts/%.wasm, $(PACKAGES))

-include ../Makefile.help

.PHONY: artifacts
## Build contracts.
artifacts: artifacts/checksums.txt

VALID_mainnet=1
VALID_testnet=1
VALID_devnet=1
.PHONY: check-network
check-network:
ifndef VALID_$(NETWORK)
	$(error Invalid or missing NETWORK. Please call with `$(MAKE) $(MAKECMDGOALS) NETWORK=[mainnet | testnet | devnet]`)
endif

$(WASMS) artifacts/checksums.txt: $(SOURCE_FILES)
	DOCKER_BUILDKIT=1 docker build --target artifacts -o artifacts .

payer-$(NETWORK).json:
	$(error Missing private key in payer-$(NETWORK).json)

.PHONY: deploy/bridge
## Deploy core bridge
deploy/bridge: bridge-code-id-$(NETWORK).txt

.PHONY: deploy/token_bridge
## Deploy token bridge
deploy/token_bridge: token_bridge-code-id-$(NETWORK).txt

.PHONY: deploy/nft_bridge
## Deploy NFT bridge
deploy/nft_bridge: nft_bridge-code-id-$(NETWORK).txt

%-code-id-$(NETWORK).txt: check-network tools/node_modules payer-$(NETWORK).json
	@echo "Deploying artifacts/$($*_SOURCE).wasm on $(NETWORK)"
	@node tools/deploy_single.js \
		--network $(NETWORK) \
		--artifact artifacts/$($*_SOURCE).wasm \
		--mnemonic "$$(cat payer-$(NETWORK).json)" \
		| grep -i "code id" | sed s/[^0-9]//g \
		> $@
	@echo "Deployed at code id $$(cat $@) (stored in $@)"

tools/node_modules: tools/package-lock.json
	cd tools && npm ci

LocalTerra:
	git clone --depth 1 https://www.github.com/terra-money/LocalTerra

test/node_modules: test/package-lock.json
	cd test && npm ci

.PHONY: test
## Run integration test
test: artifacts test/node_modules LocalTerra
	@if pgrep terrad; then echo "Error: terrad already running. Stop it before running tests"; exit 1; fi
	cd LocalTerra && docker compose up --detach
	sleep 5
	cd test && npm run test || (cd ../LocalTerra && docker compose down && exit 1)
	cd LocalTerra && docker compose down

.PHONY: clean
clean:
	rm -f $(WASMS)
	rm -f artifacts/checksums.txt
	rm -rf tools/node_modules
	rm -rf test/node_modules
