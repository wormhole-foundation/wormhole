TEST_RPC = https://ethereum-rpc.publicnode.com

tolib = $(addprefix lib/,$(firstword $(subst @, ,$(notdir $(1)))))

define install_lib_evm
dependencies-evm: $(call tolib,$(1))

$(call tolib,$(1)):
	forge install $(1) --no-git
endef

.DEFAULT_GOAL = build
.PHONY: build* test* clean* dependencies* verifiable-build*

build: build-evm build-solana build-solana-ts

test: test-evm test-solana

clean: clean-evm clean-solana

LIB_DEPS = foundry-rs/forge-std
LIB_DEPS += wormhole-foundation/wormhole-solidity-sdk@75feab111329628bd649c219d73e465d7c3f5da2

# dynamically generate install rule for each lib dependency and add to depdenencies
$(foreach dep,$(LIB_DEPS), $(eval $(call install_lib_evm,$(dep))))


build-evm: dependencies-evm
	forge build

verifiable-build-evm: dependencies-evm
	mkdir -p verifiable-evm-build && docker build  --file ./Dockerfile.evm . --tag verification-v2-evm-build --output type=local,dest=verifiable-evm-build

test-evm: dependencies-evm
	forge test
#--match-test InitiateFullFuzz
#--fork-url $(TEST_RPC)

clean-evm:
	forge clean
	rm -rf lib

dependencies-solana-ts:
	npm ci --prefix src/solana

build-solana:
	cd src/solana; anchor build

# depends on build-solana to generate the IDL
# This happens automatically as soon as you run `anchor test` but it's useful to trigger the build
# before typechecking if you're tweaking things that impact the IDL and haven't regenerated it yet.
build-solana-ts: build-solana dependencies-solana-ts
	npm run build --prefix src/solana

# cargo test runs the Rust unit tests
# anchor test runs the contract tests
test-solana: build-solana-ts
	cd src/solana; cargo test && anchor test

clean-solana:
	cd src/solana; anchor clean
