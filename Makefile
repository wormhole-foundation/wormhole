TEST_RPC = https://ethereum-rpc.publicnode.com

tolib = $(addprefix lib/,$(firstword $(subst @, ,$(notdir $(1)))))

define install_lib_evm
dependencies-evm: $(call tolib,$(1))

$(call tolib,$(1)):
	forge install $(1) --no-git
endef

.DEFAULT_GOAL = build
.PHONY: build* test* clean* dependencies*

build: build-evm build-solana build-solana-ts

test: test-evm test-solana

clean: clean-evm clean-solana

LIB_DEPS = foundry-rs/forge-std
LIB_DEPS += wormhole-foundation/wormhole-solidity-sdk@a7c8786

# dynamically generate install rule for each lib dependency and add to depdenencies
$(foreach dep,$(LIB_DEPS), $(eval $(call install_lib_evm,$(dep))))


build-evm: dependencies-evm
	forge build

test-evm: dependencies-evm
	forge test --fork-url $(TEST_RPC) -vvvv
#--match-test InitiateFullFuzz

clean-evm:
	forge clean
	rm -rf lib

dependencies-solana-ts:
	cd src/solana; npm ci

build-solana:
	cd src/solana; anchor build

build-solana-ts: dependencies-solana-ts
	npm run build --prefix src/solana

test-solana: build-solana-ts
	cd src/solana; anchor test

clean-solana:
	cd src/solana; anchor clean
