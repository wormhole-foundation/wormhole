mod transfer_tokens;
pub use transfer_tokens::*;

use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge_sdk;

/// Trait for invoking any Core Bridge instruction via CPI. This trait is used for preparing and
/// posting Core Bridge messages specifically.
pub trait InvokeTokenBridge<'info>: core_bridge_sdk::cpi::InvokeCoreBridge<'info> {
    fn token_bridge_program(&self) -> AccountInfo<'info>;

    fn token_program(&self) -> AccountInfo<'info>;
}

/// Wormhole Token Bridge Program.
pub type TokenBridge = crate::program::WormholeTokenBridgeSolana;
