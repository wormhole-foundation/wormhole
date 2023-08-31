#[cfg(feature = "cpi")]
pub mod cpi;

pub use crate::state;

pub use crate::types;

use anchor_lang::prelude::*;

pub use crate::constants::SOLANA_CHAIN;

pub static PROGRAM_ID: Pubkey = crate::ID;

pub fn compute_init_message_v1_space(payload_size: usize) -> usize {
    crate::state::PostedMessageV1::BYTES_START + payload_size
}
