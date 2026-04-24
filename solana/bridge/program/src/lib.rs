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

// Canonical base58 program ids for the core bridge on the three Solana
// clusters. These are the same values as in solana/Makefile
// (`bridge_ADDRESS_solana_{mainnet,testnet,devnet}`).
pub const SOLANA_MAINNET_BRIDGE_ADDRESS: &str = "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth";
pub const SOLANA_TESTNET_BRIDGE_ADDRESS: &str = "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5";
pub const SOLANA_DEVNET_BRIDGE_ADDRESS: &str = "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o";

/// Which Solana cluster this bridge program was built for. Determined by
/// comparing the build-time `BRIDGE_ADDRESS` against the known program ids.
/// A return value of `None` from `solana_env` means the build targets a
/// non-Solana SVM (e.g. Fogo) or an unrecognised address.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SolanaEnv {
    Mainnet,
    Testnet,
    Devnet,
}

/// Const-friendly `&str` byte-wise equality.
const fn str_eq(a: &str, b: &str) -> bool {
    let a = a.as_bytes();
    let b = b.as_bytes();
    if a.len() != b.len() {
        return false;
    }
    let mut i = 0;
    while i < a.len() {
        if a[i] != b[i] {
            return false;
        }
        i += 1;
    }
    true
}

/// Classify a bridge program id (base58) into a Solana cluster. Returns
/// `None` for non-Solana SVMs (e.g. Fogo) and any unrecognised address.
///
/// Evaluates fully at compile time, so it can be used in `const` contexts
/// that switch behaviour based on the target cluster (see `api::RETENTION_PERIOD`).
pub const fn solana_env(program_id_b58: &str) -> Option<SolanaEnv> {
    if str_eq(program_id_b58, SOLANA_MAINNET_BRIDGE_ADDRESS) {
        Some(SolanaEnv::Mainnet)
    } else if str_eq(program_id_b58, SOLANA_TESTNET_BRIDGE_ADDRESS) {
        Some(SolanaEnv::Testnet)
    } else if str_eq(program_id_b58, SOLANA_DEVNET_BRIDGE_ADDRESS) {
        Some(SolanaEnv::Devnet)
    } else {
        None
    }
}

#[cfg(test)]
mod solana_env_tests {
    use super::*;

    #[test]
    fn classifies_mainnet() {
        assert_eq!(
            solana_env("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"),
            Some(SolanaEnv::Mainnet),
        );
    }

    #[test]
    fn classifies_testnet() {
        assert_eq!(
            solana_env("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5"),
            Some(SolanaEnv::Testnet),
        );
    }

    #[test]
    fn classifies_devnet() {
        assert_eq!(
            solana_env("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"),
            Some(SolanaEnv::Devnet),
        );
    }

    #[test]
    fn rejects_fogo_addresses() {
        // Fogo is a non-Solana SVM; its bridge addresses must classify as None.
        assert_eq!(
            solana_env("BhnQyKoQQgpuRTRo6D8Emz93PvXCYfVgHhnrR4T3qhw4"),
            None,
            "Fogo testnet should classify as non-Solana",
        );
        assert_eq!(
            solana_env("worm2mrQkG1B1KTz37erMfWN8anHkSK24nzca7UD8BB"),
            None,
            "Fogo mainnet should classify as non-Solana",
        );
    }

    #[test]
    fn rejects_unknown_addresses() {
        assert_eq!(solana_env(""), None);
        assert_eq!(solana_env("not-a-real-address"), None);
        // A near-miss: the mainnet address with a single character changed.
        assert_eq!(
            solana_env("Xorm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"),
            None,
        );
    }

    #[test]
    fn rejects_length_mismatch() {
        // Prefixes / extensions of a valid address must not match.
        assert_eq!(
            solana_env("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMT"),
            None
        );
        assert_eq!(
            solana_env("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMThh"),
            None
        );
    }
}

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
    close_posted_message,
    close_signature_set_and_posted_vaa,
    initialize,
    post_message,
    post_message_unreliable,
    post_vaa,
    set_fees,
    transfer_fees,
    upgrade_contract,
    upgrade_guardian_set,
    verify_signatures,
    ClosePostedMessage,
    ClosePostedMessageData,
    CloseSignatureSetAndPostedVAA,
    CloseSignatureSetAndPostedVAAData,
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
    MESSAGE_ACCOUNT_CLOSED_DISCRIMINATOR,
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
    @event_cpi,
    Initialize                    => initialize,
    PostMessage                   => post_message,
    PostVAA                       => post_vaa,
    SetFees                       => set_fees,
    TransferFees                  => transfer_fees,
    UpgradeContract               => upgrade_contract,
    UpgradeGuardianSet            => upgrade_guardian_set,
    VerifySignatures              => verify_signatures,
    PostMessageUnreliable         => post_message_unreliable,
    ClosePostedMessage            => close_posted_message,
    CloseSignatureSetAndPostedVAA => close_signature_set_and_posted_vaa,
}
