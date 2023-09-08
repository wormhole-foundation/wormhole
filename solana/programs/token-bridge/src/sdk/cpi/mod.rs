mod transfer_tokens;
pub use transfer_tokens::*;

use anchor_lang::prelude::*;

/// Trait for invoking any Core Bridge instruction via CPI. This trait is used for preparing and
/// posting Core Bridge messages specifically.
pub trait InvokeTokenBridge<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info>;
}

/// Wormhole Token Bridge Program.
pub type TokenBridge = crate::program::WormholeTokenBridgeSolana;
