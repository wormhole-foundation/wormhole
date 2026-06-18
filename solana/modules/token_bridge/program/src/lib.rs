#![feature(adt_const_params)]
#![allow(incomplete_features)]
#![deny(unused_must_use)]

// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

#[cfg(feature = "instructions")]
pub mod instructions;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
extern crate wasm_bindgen;

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
pub mod wasm;

pub mod accounts;
pub mod api;
pub mod messages;
pub mod types;

pub use api::{
    attest_token,
    complete_native,
    complete_native_with_payload,
    complete_wrapped,
    complete_wrapped_with_payload,
    create_wrapped,
    freeze,
    initialize,
    pause,
    register_chain,
    set_pauser_addresses,
    transfer_native,
    transfer_native_with_payload,
    transfer_wrapped,
    transfer_wrapped_with_payload,
    unpause,
    unpause_expired,
    upgrade_contract,
    AttestToken,
    AttestTokenData,
    CompleteNative,
    CompleteNativeData,
    CompleteNativeWithPayload,
    CompleteNativeWithPayloadData,
    CompleteWrapped,
    CompleteWrappedData,
    CompleteWrappedWithPayload,
    CompleteWrappedWithPayloadData,
    CreateWrapped,
    CreateWrappedData,
    Freeze,
    FreezeData,
    Initialize,
    InitializeData,
    Pause,
    PauseData,
    RegisterChain,
    RegisterChainData,
    SetPauserAddresses,
    SetPauserAddressesData,
    TransferNative,
    TransferNativeData,
    TransferNativeWithPayload,
    TransferNativeWithPayloadData,
    TransferWrapped,
    TransferWrappedData,
    TransferWrappedWithPayload,
    TransferWrappedWithPayloadData,
    Unpause,
    UnpauseData,
    UnpauseExpired,
    UnpauseExpiredData,
    UpgradeContract,
    UpgradeContractData,
};

use solitaire::*;

// Static list of invalid VAA Message accounts.
pub(crate) static INVALID_VAAS: &[&str; 7] = &[
    "28Tx7c3W8rggVNyUQEAL9Uq6pUng4xJLAeLA6V8nLH1Z",
    "32YEuzLCvSyHoV6NFpaTXfiAB8sHiAnYcvP2BBeLeGWq",
    "427N2RrDHYooLvyWCiEiNR4KtGsGFTMuXiGwtuChWRSd",
    "56Vf4Y2SCxJBf4TSR24fPF8qLHhC8ZuTJvHS6mLGWieD",
    "7SzK4pmh9fM9SWLTCKmbjQC8EvDgPmtwdaBeTRztkM98",
    "G2VJNjmQsz6wfVZkTUzYAB8ZzRS2hZbpUd5Cr4DTpz6t",
    "GvAarWUV8khMLrTRouzBh3xSr8AeLDXxoKNJ6FgxGyg5",
];

pub enum TokenBridgeError {
    AlreadyExecuted,
    InvalidChain,
    InvalidGovernanceKey,
    InvalidMetadata,
    InvalidMint,
    InvalidPayload,
    InvalidUTF8String,
    TokenNotNative,
    UninitializedMint,
    WrongAccountOwner,
    InvalidFee,
    InvalidRecipient,
    InvalidVAA,
    NonexistentTokenMetadataAccount,
    NotMetadataV1Account,
    WrongMetadataAccount,
    /// `paused == true`. Lifted only by a successful `unpause` / `unpause_expired` call.
    Paused,
    /// Caller of `pause` / `unpause` is not the configured pauser / unpauser.
    InvalidPauser,
    /// Configured pauser / unpauser is the zero pubkey, so that side of the pair is disabled.
    PauserNotConfigured,
    /// `self_program` account passed by the caller does not match this program's `program_id`.
    /// Only used to defensively guard the self-CPI that emits Anchor-style events.
    InvalidSelfProgram,
    // NOTE: new variants are APPENDED below — `From<TokenBridgeError>` maps each variant to its
    // numeric position (`t as u64`), so inserting in the middle would renumber existing error
    // codes. Always append.
    /// Caller of `freeze` is not the configured freezer (or the freezer role is unassigned).
    NotFreezer,
    /// `unpause` / `unpause_expired` was called while the bridge is not paused.
    NotPaused,
    /// `unpause_expired` was called before the current pause has expired (`now < pause_expiry`).
    NotExpired,
}

impl From<TokenBridgeError> for SolitaireError {
    fn from(t: TokenBridgeError) -> SolitaireError {
        SolitaireError::Custom(t as u64)
    }
}

solitaire! {
    @event_cpi,
    Initialize => initialize,
    AttestToken => attest_token,
    CompleteNative => complete_native,
    CompleteWrapped => complete_wrapped,
    TransferWrapped => transfer_wrapped,
    TransferNative => transfer_native,
    RegisterChain => register_chain,
    CreateWrapped => create_wrapped,
    UpgradeContract => upgrade_contract,
    CompleteNativeWithPayload => complete_native_with_payload,
    CompleteWrappedWithPayload => complete_wrapped_with_payload,
    TransferWrappedWithPayload => transfer_wrapped_with_payload,
    TransferNativeWithPayload => transfer_native_with_payload,
    SetPauserAddresses => set_pauser_addresses,
    Pause => pause,
    Unpause => unpause,
    // Appended after the existing instructions so their dispatch indices (the on-chain
    // instruction discriminators) stay stable.
    Freeze => freeze,
    UnpauseExpired => unpause_expired,
}
