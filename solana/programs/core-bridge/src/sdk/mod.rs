//! **ATTENTION INTEGRATORS!** Core Bridge Program developer kit. It is recommended to use
//! [sdk::cpi](mod@crate::sdk::cpi) for invoking Core Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](mod@crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](mod@crate::legacy::cpi)).

pub use crate::{
    constants::{PROGRAM_EMITTER_SEED_PREFIX, SOLANA_CHAIN},
    state, types,
};

/// Methods useful for interacting with the Core Bridge program via CPI if your program composes
/// with this program.
#[cfg(feature = "cpi")]
pub mod cpi;

/// The program ID of the Core Bridge program.
pub static PROGRAM_ID: anchor_lang::prelude::Pubkey = crate::ID;

/// Convenient method to determine the space required for a `PostedMessageV1` account when it is
/// being prepared via `init_message_v1` and `process_message_v1`.
pub fn compute_init_message_v1_space(payload_size: usize) -> usize {
    crate::state::PostedMessageV1::BYTES_START + payload_size
}
