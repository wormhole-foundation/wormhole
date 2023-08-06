mod publish_message;
pub use publish_message::*;

mod prepare_message_v1;
pub use prepare_message_v1::*;

use anchor_lang::prelude::*;

/// Trait for invoking any Core Bridge instruction via CPI. This trait is used for preparing and
/// posting Core Bridge messages specifically.
pub trait InvokeCoreBridge<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info>;
}

/// Trait for invoking any program instruction that requires account creation.
pub trait CreateAccount<'info> {
    fn payer(&self) -> AccountInfo<'info>;

    fn system_program(&self) -> AccountInfo<'info>;
}

/// Wormhole Core Bridge Program.
pub type CoreBridge = crate::program::WormholeCoreBridgeSolana;
