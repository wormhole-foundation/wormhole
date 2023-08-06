#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;

#[cfg(feature = "localnet")]
declare_id!("bPPNmBhmHfkEFJmNKKCvwc1tPqBjzPDRwCw3yQYYXQa");

#[cfg(feature = "mainnet")]
declare_id!("wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb");

#[cfg(feature = "testnet")]
declare_id!("DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe");

#[cfg(feature = "devnet")]
declare_id!("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE");

pub mod constants;
pub mod error;
pub mod legacy;
mod processor;
pub mod state;
pub mod utils;

pub(crate) use processor::*;

#[program]
pub mod solana_wormhole_token_bridge {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        processor::initialize(ctx)
    }

    // Fallback to legacy instructions below.

    pub fn process_legacy_instruction(
        program_id: &Pubkey,
        account_infos: &[AccountInfo],
        ix_data: &[u8],
    ) -> Result<()> {
        legacy::process_legacy_instruction(program_id, account_infos, ix_data)
    }
}
