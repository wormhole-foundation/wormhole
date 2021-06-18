// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]
#![feature(const_generics)]
#![allow(warnings)]

// Salt contains the framework definition, single file for now but to be extracted into a cargo
// package as soon as possible.
pub mod accounts;
pub mod api;
pub mod types;
pub mod vaa;

use solitaire::*;

pub use api::{
    initialize,
    post_message,
    post_vaa,
    upgrade_contract,
    upgrade_guardian_set,
    verify_signatures,
    Initialize,
    PostMessage,
    PostMessageData,
    PostVAA,
    PostVAAData,
    UpgradeContract,
    UpgradeContractData,
    UpgradeGuardianSet,
    UpgradeGuardianSetData,
    VerifySignatures,
    VerifySignaturesData,
};
use types::BridgeConfig;

const MAX_LEN_GUARDIAN_KEYS: u64 = 19;

enum Error {
    InsufficientFees,
    PostVAAGuardianSetExpired,
    PostVAAConsensusFailed,
    VAAAlreadyExecuted,
    InstructionAtWrongIndex,
    InvalidSecpInstruction,
    InvalidHash,
    GuardianSetMismatch,
    InvalidGovernanceModule,
    InvalidGovernanceChain,
    InvalidGovernanceAction,
    InvalidGuardianSetUpgrade,
    MathOverflow,
    InvalidFeeRecipient,
}

impl From<Error> for SolitaireError {
    fn from(e: Error) -> SolitaireError {
        SolitaireError::Custom(e as u64)
    }
}

solitaire! {
    Initialize(BridgeConfig)                    => initialize,
    PostVAA(PostVAAData)                        => post_vaa,
    PostMessage(PostMessageData)                => post_message,
    VerifySignatures(VerifySignaturesData)      => verify_signatures,
    UpgradeContract(UpgradeContractData)        => upgrade_contract,
    UpgradeGuardianSet(UpgradeGuardianSetData)  => upgrade_guardian_set,
}
