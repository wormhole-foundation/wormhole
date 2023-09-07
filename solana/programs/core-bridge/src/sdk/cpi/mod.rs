//! CPI builders. Methods useful for interacting with the Core Bridge program from another program.

/// Sub-module for System program interaction.
pub mod system_program {
    pub use crate::utils::cpi::{create_account, CreateAccount};
}

pub use crate::utils::vaa::claim_vaa;

mod close_encoded_vaa;
pub use close_encoded_vaa::*;

mod publish_message;
pub use publish_message::*;

mod prepare_message;
pub use prepare_message::*;

/// Wormhole Core Bridge Program.
pub type CoreBridge = crate::program::WormholeCoreBridgeSolana;
