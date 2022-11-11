SOURCE_FILES:=$(shell find contracts -name "*.sol")

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
forge_dependencies: lib/forge-std

lib/forge-std:
	forge install foundry-rs/forge-std@2c7cbfc6fbede6d7c9e6b17afe997e3fdfe22fef --no-git --no-commit

dependencies: node_modules forge_dependencies

build: node_modules ${SOURCE_FILES}
	mkdir -p build
	touch -m build
	npm run build

.env: .env.test
	cp $< $@

test: test-forge test-ganache

.PHONY: test-ganache
test-ganache: build .env dependencies
	@if pgrep ganache-cli; then echo "Error: ganache-cli already running. Stop it before running tests"; exit 1; fi
	npx ganache-cli -e 10000 --deterministic --time="1970-01-01T00:00:00+00:00" > ganache.log &
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
	rm -rf ganache.log .env node_modules build
