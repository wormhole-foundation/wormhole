#![feature(const_generics)]
#![allow(non_upper_case_globals)]
#![allow(incomplete_features)]

pub mod accounts;
pub mod api;
pub mod error;
pub mod types;
pub mod vaa;

#[cfg(feature = "no-entrypoint")]
pub mod instructions;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
extern crate wasm_bindgen;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
pub mod wasm;

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
    DeserializeGovernancePayload,
    DeserializePayload,
    PayloadMessage,
    SerializeGovernancePayload,
    SerializePayload,
};

pub const MAX_LEN_GUARDIAN_KEYS: usize = 19;
pub const CHAIN_ID_SOLANA: u16 = 1;

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
