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
forge_dependencies: lib/forge-std lib/openzeppelin-contracts

lib/forge-std:
	forge install foundry-rs/forge-std@v1.5.5 --no-git

lib/openzeppelin-contracts:
	forge install openzeppelin/openzeppelin-contracts@0457042d93d9dfd760dbaa06a4d2f1216fdbe297 --no-git

dependencies: node_modules forge_dependencies

build: forge_dependencies node_modules ${SOURCE_FILES}
	npm run build

.env: .env.test
	cp $< $@

test: test-forge

.PHONY:
test-forge: dependencies
	forge test --no-match-test .*_KEVM # ignore KEVM tests (see PROOFS.md)

# Verify that the contracts do not include PUSH0 opcodes
test-push0: dependencies
	forge build --extra-output evm.bytecode.opcodes
	@if grep -qr --include \*.json PUSH0 ./build-forge; then echo "Contract uses PUSH0 instruction" 1>&2; exit 1; fi

clean:
	rm -rf .env node_modules build-forge ethers-contracts lib/forge-std lib/openzeppelin-contracts
