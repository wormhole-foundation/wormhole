#![allow(clippy::result_large_err)]

#[cfg(feature = "localnet")]
declare_id!("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");

#[cfg(feature = "mainnet")]
declare_id!("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth");

#[cfg(feature = "testnet")]
declare_id!("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5");

pub mod constants;

pub mod error;

pub mod legacy;

mod processor;
pub(crate) use processor::*;
pub use processor::{
    ClosePostedVaaV1Directive, InitMessageV1Args, PostVaaV1Directive, ProcessEncodedVaaDirective,
    ProcessMessageV1Directive,
};

pub mod sdk;

pub mod state;

pub mod types;

pub mod utils;

pub mod zero_copy;

use anchor_lang::prelude::*;

#[program]
pub mod wormhole_core_bridge_solana {
    use super::*;

    /// Processor used to initialize a created account as [crate::state::PostedMessageV1]. An
    /// authority (the emitter authority) is established with this instruction.
    pub fn init_message_v1(ctx: Context<InitMessageV1>, args: InitMessageV1Args) -> Result<()> {
        processor::init_message_v1(ctx, args)
    }

    /// Processor used to process a draft [crate::state::PostedMessageV1] account. This instruction
    /// requires an authority (the emitter authority) to interact with the message account.
    pub fn process_message_v1(
        ctx: Context<ProcessMessageV1>,
        directive: ProcessMessageV1Directive,
    ) -> Result<()> {
        processor::process_message_v1(ctx, directive)
    }

    /// Processor used to intialize a created account as [crate::state::EncodedVaa]. An authority
    /// (the write authority) is established with this instruction.
    pub fn init_encoded_vaa(ctx: Context<InitEncodedVaa>) -> Result<()> {
        processor::init_encoded_vaa(ctx)
    }

    /// Processor used to process an [crate::state::EncodedVaa] account. This instruction requires
    /// an authority (the write authority) to interact with the encoded VAA account.
    pub fn process_encoded_vaa(
        ctx: Context<ProcessEncodedVaa>,
        directive: ProcessEncodedVaaDirective,
    ) -> Result<()> {
        processor::process_encoded_vaa(ctx, directive)
    }

    /// Processor used to close an [crate::state::EncodedVaa] account to create a
    /// [crate::state::PostedVaaV1] account in its place.
    ///
    /// NOTE: Because the legacy verify signatures instruction was not required for the Posted VAA
    /// account to exist, the encoded [crate::state::SignatureSet] is the default [Pubkey].
    pub fn post_vaa_v1(ctx: Context<PostVaaV1>, directive: PostVaaV1Directive) -> Result<()> {
        processor::post_vaa_v1(ctx, directive)
    }

    /// Processor used to close a [crate::state::PostedVaaV1] account. If a
    /// [crate::state::SignatureSet] were used to verify the VAA, that account will be closed, too.
    pub fn close_posted_vaa_v1(
        ctx: Context<ClosePostedVaaV1>,
        directive: ClosePostedVaaV1Directive,
    ) -> Result<()> {
        processor::close_posted_vaa_v1(ctx, directive)
    }

    /// Process legacy Core Bridge instructions. See [crate::legacy] for more info.
    pub fn process_legacy_instruction(
        program_id: &Pubkey,
        account_infos: &[AccountInfo],
        ix_data: &[u8],
    ) -> Result<()> {
        legacy::process_legacy_instruction(program_id, account_infos, ix_data)
    }
}
