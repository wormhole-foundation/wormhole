// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]
#![feature(const_generics)]
#![allow(warnings)]

use solana_program::msg;

pub mod accounts;
pub mod api;
pub mod types;
pub mod vaa;

#[cfg(feature = "no-entrypoint")]
pub mod instructions;

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
    Signature,
    UninitializedMessage,
    UpgradeContract,
    UpgradeContractData,
    UpgradeGuardianSet,
    UpgradeGuardianSetData,
    VerifySignatures,
    VerifySignaturesData,
};
use types::BridgeConfig;

const MAX_LEN_GUARDIAN_KEYS: usize = 19;

#[derive(Debug)]
enum Error {
    GuardianSetMismatch,
    InstructionAtWrongIndex,
    InsufficientFees,
    InvalidFeeRecipient,
    InvalidGovernanceAction,
    InvalidGovernanceChain,
    InvalidGovernanceModule,
    InvalidGuardianSetUpgrade,
    InvalidHash,
    InvalidSecpInstruction,
    MathOverflow,
    PostVAAConsensusFailed,
    PostVAAGuardianSetExpired,
    VAAAlreadyExecuted,
}

impl From<Error> for SolitaireError {
    fn from(e: Error) -> SolitaireError {
        msg!("ProgramError: {:?}", e);
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
