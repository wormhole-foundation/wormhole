TEST_RPC = https://ethereum-rpc.publicnode.com

tolib = $(addprefix lib/,$(firstword $(subst @, ,$(notdir $(1)))))

define install_lib
dependencies: $(call tolib,$(1))

$(call tolib,$(1)):
	forge install $(1) --no-git --no-commit
endef

.DEFAULT_GOAL = build
.PHONY: build test clean dependencies

build: dependencies
	forge build

test: dependencies
	forge test --fork-url $(TEST_RPC) -vvvv
#--match-test InitiateFullFuzz

LIB_DEPS = foundry-rs/forge-std
LIB_DEPS += wormhole-foundation/wormhole-solidity-sdk@a7c8786

# dynamically generate install rule for each lib dependency and add to depdenencies
$(foreach dep,$(LIB_DEPS), $(eval $(call install_lib,$(dep))))

clean:
	forge clean
	rm -rf lib
