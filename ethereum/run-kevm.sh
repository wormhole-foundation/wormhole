#!/usr/bin/env bash
set -euxo pipefail

forge_build() {
    # Avoid building Migrator contract (see PROOFS.md for explanation)
    forge build --skip Migrator.sol
}

foundry_kompile() {
    kevm foundry-kompile --verbose      \
        --require wormhole-lemmas.k     \
        --module-import WORMHOLE-LEMMAS \
        ${rekompile}                    \
        ${regen}
}

foundry_prove() {
    kevm foundry-prove                     \
        --max-depth ${max_depth}           \
        --max-iterations ${max_iterations} \
        --workers ${workers}               \
        --verbose                          \
        ${reinit}                          \
        ${debug}                           \
        ${simplify_init}                   \
        ${implication_every_block}         \
        ${break_every_step}                \
        ${break_on_calls}                  \
        ${auto_abstract}                   \
        ${tests[*]}
}

max_depth=5000
max_iterations=5000

# Number of processes run by the prover in parallel
# Should be at most (M - 8) / 8 in a machine with M GB of RAM
workers=1

# Switch the options below to turn them on or off

# Turn on to regenerate K definitions if Solidity code or KEVM version changes
regen=--regen
regen=

# Turn on if new lemmas have been added to wormhole-lemmas.k (subsumed by --regen)
rekompile=--rekompile
rekompile=

# Progress is saved automatically so an unfinished proof can be resumed from where it left off
# Turn on to restart proof from the beginning instead of resuming
reinit=--reinit
reinit=

debug=--debug
debug=

simplify_init=--no-simplify-init
simplify_init=

implication_every_block=--implication-every-block
implication_every_block=

break_every_step=--break-every-step
break_every_step=

# Turn off to save the state before every call to the KCFG
break_on_calls=
break_on_calls=--no-break-on-calls

auto_abstract=--auto-abstract
auto_abstract=

# List of tests to symbolically execute

tests=(
    "--test TestSetters.testUpdateGuardianSetIndex_KEVM "
    "--test TestSetters.testExpireGuardianSet_KEVM "
    "--test TestSetters.testSetMessageFee_KEVM "
    "--test TestSetters.testSetGovernanceContract_KEVM "
    "--test TestSetters.testSetInitialized_KEVM "
    "--test TestSetters.testSetGovernanceActionConsumed_KEVM "
    "--test TestSetters.testSetChainId_KEVM "
    "--test TestSetters.testSetGovernanceChainId_KEVM "
    "--test TestSetters.testSetNextSequence_KEVM "
    "--test TestSetters.testSetEvmChainId_Success_KEVM "
    "--test TestSetters.testSetEvmChainId_Revert_KEVM "
    "--test TestGetters.testGetGuardianSetIndex_KEVM "
    "--test TestGetters.testGetMessageFee_KEVM "
    "--test TestGetters.testGetGovernanceContract_KEVM "
    "--test TestGetters.testIsInitialized_KEVM "
    "--test TestGetters.testGetGovernanceActionConsumed_KEVM "
    "--test TestGetters.testChainId_KEVM "
    "--test TestGetters.testGovernanceChainId_KEVM "
    "--test TestGetters.testNextSequence_KEVM "
    "--test TestGetters.testEvmChainId_KEVM "
    "--test TestGovernanceStructs.testParseContractUpgrade_KEVM "
    "--test TestGovernanceStructs.testParseContractUpgradeWrongAction_KEVM "
    "--test TestGovernanceStructs.testParseSetMessageFee_KEVM "
    "--test TestGovernanceStructs.testParseSetMessageFeeWrongAction_KEVM "
    "--test TestGovernanceStructs.testParseTransferFees_KEVM "
    "--test TestGovernanceStructs.testParseTransferFeesWrongAction_KEVM "
    "--test TestGovernanceStructs.testParseRecoverChainId_KEVM "
    "--test TestGovernanceStructs.testParseRecoverChainIdWrongAction_KEVM "
    "--test TestSetup.testInitialize_after_setup_revert_KEVM "
    "--test TestSetup.testSetup_after_setup_revert_KEVM "
)

# Comment these lines as needed
pkill kore-rpc || true
forge_build
foundry_kompile
foundry_prove
