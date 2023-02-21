SOURCE_FILES:=$(shell find contracts -name "*.sol")

.PHONY: dependencies forge_dependencies wormhole_dependencies unit-test integration-test clean all

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
	forge install foundry-rs/forge-std@2c7cbfc6fbede6d7c9e6b17afe997e3fdfe22fef --no-git --no-commit

lib/openzeppelin-contracts:
	forge install openzeppelin/openzeppelin-contracts@0457042d93d9dfd760dbaa06a4d2f1216fdbe297 --no-git --no-commit

.PHONY: wormhole_dependencies

dependencies: node_modules forge_dependencies

build: dependencies
	npm run build
	npm run typechain

.env: .env.test
	cp $< $@

.PHONY: test
test: unit-test integration-test

.PHONY: unit-test
unit-test: dependencies build
	npm run unit-test

.PHONY: integration-test
integration-test: dependencies build
	npm run integration-test

.PHONY: clean
clean:
	rm -rf anvil.log .env node_modules build lib wormhole
