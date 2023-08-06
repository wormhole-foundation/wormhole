#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;

#[cfg(feature = "localnet")]
declare_id!("agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC");

#[cfg(feature = "mainnet")]
declare_id!("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth");

#[cfg(feature = "testnet")]
declare_id!("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5");

#[cfg(feature = "devnet")]
declare_id!("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");

pub mod constants;
pub mod error;
pub mod legacy;
pub mod message;
mod processor;
pub mod state;
pub mod types;
pub mod utils;

pub(crate) use processor::*;

#[cfg(feature = "cpi")]
pub use legacy::cpi::*;

#[derive(Clone)]
pub struct CoreBridge;

impl Id for CoreBridge {
    fn id() -> Pubkey {
        ID
    }
}

#[program]
pub mod solana_wormhole_core_bridge {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>, args: InitializeArgs) -> Result<()> {
        processor::initialize(ctx, args)
    }

    pub fn init_message_v1(ctx: Context<InitMessageV1>, args: InitMessageV1Args) -> Result<()> {
        processor::init_message_v1(ctx, args)
    }

    pub fn process_message_v1(
        ctx: Context<ProcessMessageV1>,
        directive: ProcessMessageV1Directive,
    ) -> Result<()> {
        processor::process_message_v1(ctx, directive)
    }

    pub fn init_encoded_vaa(ctx: Context<InitEncodedVaa>) -> Result<()> {
        processor::init_encoded_vaa(ctx)
    }

    pub fn process_encoded_vaa(
        ctx: Context<ProcessEncodedVaa>,
        directive: ProcessEncodedVaaDirective,
    ) -> Result<()> {
        processor::process_encoded_vaa(ctx, directive)
    }

    pub fn post_vaa_v1(ctx: Context<PostVaaV1>, directive: PostVaaV1Directive) -> Result<()> {
        processor::post_vaa_v1(ctx, directive)
    }

    pub fn close_posted_vaa_v1(
        ctx: Context<ClosePostedVaaV1>,
        directive: ClosePostedVaaV1Directive,
    ) -> Result<()> {
        processor::close_posted_vaa_v1(ctx, directive)
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
