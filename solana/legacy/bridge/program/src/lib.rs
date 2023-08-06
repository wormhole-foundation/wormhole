#![feature(adt_const_params)]
#![allow(non_upper_case_globals)]
#![allow(incomplete_features)]

use solitaire::*;

pub const MAX_LEN_GUARDIAN_KEYS: usize = 19;
pub const CHAIN_ID_SOLANA: u16 = 1;
pub const CHAIN_ID_GOVERANCE: u16 = 1;

#[cfg(feature = "instructions")]
pub mod instructions;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
extern crate wasm_bindgen;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
pub mod wasm;

pub mod accounts;

pub use accounts::{
    BridgeConfig,
    BridgeData,
    Claim,
    ClaimData,
    ClaimDerivationData,
    FeeCollector,
    GuardianSet,
    GuardianSetData,
    GuardianSetDerivationData,
    MessageData,
    PostedMessage,
    PostedMessageData,
    PostedMessageUnreliable,
    PostedMessageUnreliableData,
    PostedVAA,
    PostedVAAData,
    Sequence,
    SequenceDerivationData,
    SequenceTracker,
    SignatureSet,
    SignatureSetData,
};

pub mod api;

pub use api::{
    initialize,
    post_message,
    post_message_unreliable,
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
    PostMessageUnreliable,
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

pub mod error;
pub mod types;
pub mod vaa;

pub use vaa::{
    DeserializeGovernancePayload,
    DeserializePayload,
    PayloadMessage,
    SerializeGovernancePayload,
    SerializePayload,
};

solitaire! {
    Initialize         => initialize,
    PostMessage        => post_message,
    PostVAA            => post_vaa,
    SetFees            => set_fees,
    TransferFees       => transfer_fees,
    UpgradeContract    => upgrade_contract,
    UpgradeGuardianSet => upgrade_guardian_set,
    VerifySignatures   => verify_signatures,
    PostMessageUnreliable        => post_message_unreliable,
}
