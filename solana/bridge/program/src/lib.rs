#![feature(const_generics)]
#![allow(non_upper_case_globals)]
#![allow(incomplete_features)]

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
    set_fees,
    transfer_fees,
    upgrade_contract,
    upgrade_guardian_set,
    verify_signatures,
    Initialize,
    InitializeData,
    PostMessage,
    PostMessageData,
    PostVAA,
    PostVAAData,
    SetFees,
    SetFeesData,
    Signature,
    TransferFees,
    TransferFeesData,
    UninitializedMessage,
    UpgradeContract,
    UpgradeContractData,
    UpgradeGuardianSet,
    UpgradeGuardianSetData,
    VerifySignatures,
    VerifySignaturesData,
};

pub use vaa::{
    DeserializePayload,
    PayloadMessage,
    SerializePayload,
};

const MAX_LEN_GUARDIAN_KEYS: usize = 19;

const CHAIN_ID_SOLANA: u16 = 1;

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
    TooManyGuardians,
}

/// Translate from program specific errors to Solitaire framework errors. Log the error on the way
/// out of the program for debugging.
impl From<Error> for SolitaireError {
    fn from(e: Error) -> SolitaireError {
        msg!("ProgramError: {:?}", e);
        SolitaireError::Custom(e as u64)
    }
}

solitaire! {
    Initialize(InitializeData)                  => initialize,
    PostMessage(PostMessageData)                => post_message,
    PostVAA(PostVAAData)                        => post_vaa,
    SetFees(SetFeesData)                        => set_fees,
    TransferFees(TransferFeesData)              => transfer_fees,
    UpgradeContract(UpgradeContractData)        => upgrade_contract,
    UpgradeGuardianSet(UpgradeGuardianSetData)  => upgrade_guardian_set,
    VerifySignatures(VerifySignaturesData)      => verify_signatures,
}
