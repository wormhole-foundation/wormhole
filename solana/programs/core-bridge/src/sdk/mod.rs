#[cfg(feature = "cpi")]
pub mod cpi {
    pub use crate::cpi::accounts::{InitMessageV1, ProcessMessageV1};
    pub use crate::cpi::{init_message_v1, process_message_v1};
    pub use crate::processor::{InitMessageV1Args, ProcessMessageV1Directive};
}

use anchor_lang::prelude::*;

pub static PROGRAM_ID: Pubkey = crate::ID;

pub fn compute_init_message_v1_space(payload_size: usize) -> usize {
    crate::state::PostedMessageV1::BYTES_START + payload_size
}
