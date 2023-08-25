#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;

#[cfg(feature = "localnet")]
declare_id!("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE");

#[cfg(feature = "mainnet")]
declare_id!("wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb");

#[cfg(feature = "testnet")]
declare_id!("DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe");

pub mod constants;

pub mod error;

pub mod legacy;

pub(crate) mod messages;

mod processor;
pub(crate) use processor::*;

pub mod state;

pub mod utils;

// #[cfg(feature = "cpi")]
// pub use legacy::cpi::*;

#[derive(Clone)]
pub struct TokenBridge;

impl Id for TokenBridge {
    fn id() -> Pubkey {
        ID
    }
}

#[program]
pub mod wormhole_token_bridge_solana {
    use super::*;

    pub fn register_chain(ctx: Context<RegisterChain>) -> Result<()> {
        processor::register_chain(ctx)
    }

    pub fn secure_registered_emitter(
        ctx: Context<SecureRegisteredEmitter>,
        directive: SecureRegisteredEmitterDirective,
    ) -> Result<()> {
        processor::secure_registered_emitter(ctx, directive)
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
