bridge_SOURCE=wormhole
token_bridge_SOURCE=token_bridge_terra

SOURCE_FILES=$(shell find . -name "*.rs" -or -name "*.lock" -or -name "*.toml" | grep -v target)

PACKAGES=$(shell find . -name "Cargo.toml" | grep -E 'packages|contracts' | xargs cat | grep "name *=" | cut -d' ' -f3 | sed s/\"//g | sed s/-/_/g)
WASMS_terra2=$(patsubst %, artifacts/terra2/%.wasm, $(PACKAGES))
WASMS_injective=$(patsubst %, artifacts/injective/%.wasm, $(PACKAGES))


-include ../Makefile.help

.PHONY: artifacts
## Build contracts.
artifacts/terra2: artifacts/terra2/checksums.txt 

artifacts/injective: artifacts/injective/checksums.txt 

VALID_mainnet=1
VALID_testnet=1
VALID_devnet=1
.PHONY: check-network
check-network:
ifndef VALID_$(NETWORK)
	$(error Invalid or missing NETWORK. Please call with `$(MAKE) $(MAKECMDGOALS) NETWORK=[mainnet | testnet | devnet]`)
endif

$(WASMS_terra2) artifacts/terra2/checksums.txt: $(SOURCE_FILES)
	DOCKER_BUILDKIT=1 docker build --target artifacts --build-arg CHAIN_NAME=terra2 -o artifacts/terra2 .

$(WASMS_injective) artifacts/injective/checksums.txt: $(SOURCE_FILES)
	DOCKER_BUILDKIT=1 docker build --target artifacts --build-arg CHAIN_NAME=injective -o artifacts/injective .

payer-$(NETWORK).json:
	$(error Missing private key in payer-$(NETWORK).json)

.PHONY: deploy/bridge
## Deploy core bridge
deploy/bridge: bridge-code-id-$(NETWORK).txt

.PHONY: deploy/token_bridge
## Deploy token bridge
deploy/token_bridge: token_bridge-code-id-$(NETWORK).txt

%-code-id-$(NETWORK).txt: check-network tools/node_modules payer-$(NETWORK).json artifacts
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

.PHONY: unit-test
## Run unit tests
unit-test:
	CHAIN_NAME=terra2 cargo test -p wormhole-bridge-terra-2
	CHAIN_NAME=terra2 cargo test -p token-bridge-terra-2

.PHONY: test
## Run unit and integration tests
test: artifacts/terra2 test/node_modules LocalTerra unit-test
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
