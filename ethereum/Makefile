SOURCE_FILES:=$(shell find contracts -name "*.sol")

# List of files to flatten. These are typically the 'leaves' of the
# dependendency tree, i.e. the contracts that actually get deployed on-chain.
# Proxies and implementations.
FLATTEN_FILES:= contracts/Implementation.sol \
                contracts/Wormhole.sol \
                contracts/bridge/BridgeImplementation.sol \
                contracts/bridge/TokenBridge.sol \
                contracts/bridge/token/TokenImplementation.sol \
                contracts/bridge/token/Token.sol \
                contracts/nft/NFTBridgeImplementation.sol \
                contracts/nft/NFTBridgeEntrypoint.sol \
                contracts/nft/token/NFT.sol \
                contracts/nft/token/NFTImplementation.sol \
                contracts/nft/token/NFT.sol

.PHONY: dependencies test clean all

all: build

node_modules: package-lock.json
	touch -m node_modules
	npm ci

# Note: Forge really wants to manage dependencies via submodules, but that
# workflow is a little awkward. There's currently no support for a more
# traditional package manager workflow (with a package manifest file and
# installation into a subdirectory that can be gitignored).
# Instead, we just specify the dependencies here. make will then take care of
# installing them if they are not yet present.
# When adding a new dependency, make sure to specify the exact commit hash, and
# the --no-git and --no-commit flags (see lib/forge-std below)
.PHONY: forge_dependencies
forge_dependencies: lib/forge-std lib/openzeppelin-contracts

lib/forge-std:
	forge install foundry-rs/forge-std@v1.5.5 --no-git --no-commit

lib/openzeppelin-contracts:
	forge install openzeppelin/openzeppelin-contracts@0457042d93d9dfd760dbaa06a4d2f1216fdbe297 --no-git --no-commit

dependencies: node_modules forge_dependencies

build: forge_dependencies node_modules ${SOURCE_FILES}
	mkdir -p build
	touch -m build
	npm run build
	

flattened/%.sol: contracts/%.sol node_modules
	@mkdir -p $(dir $@)
    # We remove the license (SPDX) lines and ABIEncoderV2 lines because they
    # break compilation in the flattened file if there are multiple conflicting
    # declarations.
	npx truffle-flattener $< | grep -v SPDX | grep -v ABIEncoderV2 > $@

.PHONY: flattened
flattened: $(patsubst contracts/%, flattened/%, $(FLATTEN_FILES))

.env: .env.test
	cp $< $@

test: test-forge test-ganache

.PHONY: test-ganache
test-ganache: build .env dependencies
	@if pgrep ganache-cli; then echo "Error: ganache-cli already running. Stop it before running tests"; exit 1; fi
	. ./.env && npx ganache-cli --chain.vmErrorsOnRPCResponse --chain.chainId $$INIT_EVM_CHAIN_ID --wallet.defaultBalance 10000 --wallet.deterministic --chain.time="1970-01-01T00:00:00+00:00" --chain.asyncRequestProcessing=false > ganache.log &
	sleep 5
	npm test || (pkill ganache-cli && exit 1)
	pkill ganache-cli || true

.PHONY: test-upgrade
test-upgrade: build .env node_modules
	./simulate_upgrades

.PHONY:
test-forge: dependencies
	./compare-method-identifiers.sh contracts/Implementation.sol:Implementation contracts/interfaces/IWormhole.sol:IWormhole
	./compare-method-identifiers.sh contracts/bridge/BridgeImplementation.sol:BridgeImplementation contracts/bridge/interfaces/ITokenBridge.sol:ITokenBridge
	./compare-method-identifiers.sh contracts/nft/NFTBridgeImplementation.sol:NFTBridgeImplementation contracts/nft/interfaces/INFTBridge.sol:INFTBridge
	forge test

clean:
	rm -rf ganache.log .env node_modules build flattened build-forge ethers-contracts
