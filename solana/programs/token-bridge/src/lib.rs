#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;

cfg_if::cfg_if! {
    if #[cfg(feature = "localnet")] {
        declare_id!("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE");
    } else if #[cfg(feature = "mainnet")] {
        declare_id!("wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb");
    } else if #[cfg(feature = "testnet")] {
        declare_id!("DZnkkTmCiFWfYTfT41X3Rd1kDgozqzxWaHqsw6W4x2oe");
    }
}

pub mod constants;

pub mod error;

pub mod legacy;

pub(crate) mod messages;

mod processor;
pub(crate) use processor::*;

pub mod state;

pub mod utils;

pub mod zero_copy;

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

    /// Processor for registering a new foreign Token Bridge emitter. This instruction replaces the
    /// legacy register chain instruction (which is now deprecated). This instruction handler
    /// creates two [RegisteredEmitter](crate::legacy::state::RegisteredEmitter) accounts: one with
    /// a PDA address derived using the old way of [emitter_chain, emitter_address] and the more
    /// secure way of [emitter_chain]. By creating both of these accounts, we can consider migrating
    /// to the newly derived account and closing the legacy account in the future.
    pub fn register_chain(ctx: Context<RegisterChain>) -> Result<()> {
        processor::register_chain(ctx)
    }

    /// Processor for securing an existing (legacy)
    /// [RegisteredEmitter](crate::legacy::state::RegisteredEmitter) by creating a new
    /// [RegisteredEmitter](crate::legacy::state::RegisteredEmitter) account with a PDA address with
    /// seeds [emitter_chain]. We can consider migrating to the newly derived account and closing
    /// the legacy account in the future.
    pub fn secure_registered_emitter(
        ctx: Context<SecureRegisteredEmitter>,
        directive: SecureRegisteredEmitterDirective,
    ) -> Result<()> {
        processor::secure_registered_emitter(ctx, directive)
    }

    /// Process legacy Token Bridge instructions. See [legacy](crate::legacy) for more info.
    pub fn process_legacy_instruction(
        program_id: &Pubkey,
        account_infos: &[AccountInfo],
        ix_data: &[u8],
    ) -> Result<()> {
        legacy::process_legacy_instruction(program_id, account_infos, ix_data)
    }
}
