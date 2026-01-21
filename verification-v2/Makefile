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
	mkdir -p verifiable-evm-build && docker build --file ./evm-build.Dockerfile . --tag verification-v2-evm-build --output type=local,dest=verifiable-evm-build

test-evm: dependencies-evm
	forge test
#--match-test InitiateFullFuzz
#--fork-url $(TEST_RPC)

measure-performance-evm: dependencies-evm
	forge test --match-test test_benchmark -vvvv | \
	awk -f extract-gas.awk

clean-evm:
	forge clean
	rm -rf lib

dependencies-solana-ts:
	yarn install --immutable

# Defaults for Solana mainnet
BRIDGE_ADDRESS ?= worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth

build-solana:
	cd src/solana; anchor build

verifiable-build-solana:
	@if [ -z "$(VERIFICATION_V2_PROGRAM_ID)" ]; then \
		echo "Error: VERIFICATION_V2_PROGRAM_ID must be defined" >&2; \
		exit 1; \
	fi
	rm -rf src/solana/target/
	docker build --tag verification-v2-svm-build --file svm-build.Dockerfile .
	cd src/solana; solana-verify build \
	  --base-image verification-v2-svm-build \
	  --library-name verification_v2 -- \
	  --no-default-features \
	  --features from-env \
	  --config "env.BRIDGE_ADDRESS=\"$$BRIDGE_ADDRESS\"" \
	  --config "env.VERIFICATION_V2_PROGRAM_ID=\"$$VERIFICATION_V2_PROGRAM_ID\""

# depends on build-solana to generate the IDL
# This happens automatically as soon as you run `anchor test` but it's useful to trigger the build
# before typechecking if you're tweaking things that impact the IDL and haven't regenerated it yet.
build-solana-ts: build-solana dependencies-solana-ts
	yarn ./src/solana run build

# cargo test runs the Rust unit tests
# anchor test runs the contract tests
test-solana: build-solana-ts
	cd src/solana; cargo test && anchor test

clean-solana:
	cd src/solana; anchor clean
