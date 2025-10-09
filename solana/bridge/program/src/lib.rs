#![feature(adt_const_params)]
#![allow(non_upper_case_globals)]
#![allow(incomplete_features)]

use solitaire::*;

pub const MAX_LEN_GUARDIAN_KEYS: usize = 19;
pub const OUR_CHAIN_ID: u16 = parse_u16_const(env!("CHAIN_ID"));
pub const CHAIN_ID_GOVERNANCE: u16 = 1;

/// A const fn to parse a u16 from a string at compile time.
/// A newer version of std includes a const parser function, but we depend on an
/// old version, which doesn't.
/// For good measure, we include a couple of static asserts below to make sure it works.
pub const fn parse_u16_const(s: &str) -> u16 {
    let b = s.as_bytes();
    let mut i = 0;

    let mut acc: u16 = 0;
    while i < b.len() {
        let c = b[i];
        if c >= b'0' && c <= b'9' {
            let d = (c - b'0') as u16;
            acc = acc * 10 + d;
            i += 1;
        } else {
            break;
        }
    }

    if acc == 0 {
        panic!("Invalid chain id");
    }

    acc
}

const _: () = {
    assert!(parse_u16_const("1") == 1);
    assert!(parse_u16_const("65535") == 65535);
};

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
