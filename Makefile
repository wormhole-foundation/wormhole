TEST_RPC = https://rpc.ankr.com/eth

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
LIB_DEPS += openzeppelin/openzeppelin-contracts@0457042d93d9dfd760dbaa06a4d2f1216fdbe297 #for gscd's optimizations
LIB_DEPS += wormhole-foundation/wormhole-solidity-sdk@b0590b6

# dynamically generate install rule for each lib dependency and add to depdenencies
$(foreach dep,$(LIB_DEPS), $(eval $(call install_lib,$(dep))))

clean:
	forge clean
	rm -rf lib
