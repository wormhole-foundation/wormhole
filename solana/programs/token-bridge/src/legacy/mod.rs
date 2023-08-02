mod instruction;
mod processor;
pub mod state;

pub use crate::ID;
pub(crate) use instruction::*;
pub(crate) use processor::*;

#[cfg(feature = "cpi")]
pub use instruction::LegacyAttestTokenArgs;

#[cfg(feature = "cpi")]
pub mod cpi {
    use anchor_lang::prelude::*;
    use solana_program::program::invoke_signed;

    use super::*;

    // TODO
}
