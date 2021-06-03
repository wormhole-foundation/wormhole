// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

// Salt contains the framework definition, single file for now but to be extracted into a cargo
// package as soon as possible.
mod api;
mod types;

use solitaire::*;

use api::{
    initialize,
    post_vaa,
    Initialize,
    PostVAA,
    PostVAAData,
};
use types::BridgeConfig;

const VAA_TX_FEE: u64 = 0;
const MAX_LEN_GUARDIAN_KEYS: u64 = 0;

enum Error {
    InvalidSysVar,
    InsufficientFees,
    PostVAAGuardianSetExpired,
    PostVAAGuardianSetMismatch,
    PostVAAConsensusFailed,
}

impl From<Error> for SolitaireError {
    fn from(e: Error) -> SolitaireError {
        SolitaireError::Custom(e as u64)
    }
}

solitaire! {
    Initialize(BridgeConfig) => initialize,
    PostVAA(PostVAAData)     => post_vaa,
}
