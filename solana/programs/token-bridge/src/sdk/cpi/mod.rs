//! CPI builders. Methods useful for interacting with the Core Bridge program from another program.

#[doc(inline)]
pub use core_bridge_program::sdk::cpi::system_program;

/// Sub-module for SPL Token program interaction.
pub mod token {
    pub use crate::utils::cpi::{
        burn, burn_from, mint_to, transfer, transfer_from, Burn, MintTo, Transfer,
    };
}

mod complete_transfer;
pub use complete_transfer::*;

mod transfer_tokens;
pub use transfer_tokens::*;

/// Wormhole Token Bridge Program.
pub type TokenBridge = crate::program::WormholeTokenBridgeSolana;
