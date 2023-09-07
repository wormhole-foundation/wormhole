#![doc = include_str!("../README.md")]
#![allow(clippy::result_large_err)]

cfg_if::cfg_if! {
    if #[cfg(feature = "localnet")] {
        declare_id!("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");
    } else if #[cfg(feature = "mainnet")] {
        declare_id!("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth");
    } else if #[cfg(feature = "testnet")] {
        declare_id!("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5");
    }
}

pub mod constants;

pub mod error;

pub mod legacy;

mod processor;
pub(crate) use processor::*;

pub mod sdk;

pub mod state;

pub mod types;

pub(crate) mod utils;

pub(crate) mod zero_copy;

use anchor_lang::prelude::*;

#[program]
pub mod wormhole_core_bridge_solana {
    use super::*;

    /// Processor for initializing a new draft [PostedMessageV1](crate::state::PostedMessageV1)
    /// account for writing. The emitter authority is established at this point and the payload size
    /// is inferred from the size of the created account. This instruction handler also allows an
    /// integrator to publish Wormhole messages using his program's ID as the emitter address
    /// (by passing `Some(crate::ID)` to the [cpi_program_id](InitMessageV1Args::cpi_program_id)
    /// argument). **Be aware that the emitter authority's seeds must only be \[b"emitter"\] in this
    /// case.**
    ///
    /// This instruction should be followed up with `process_message_v1` to write and finalize the
    /// message account (to prepare it for publishing via the
    /// [post message instruction](crate::legacy::instruction::LegacyInstruction)).
    ///
    /// NOTE: If you wish to publish a small message (one where the data does not overflow the
    /// Solana transaction size), it is recommended that you use an [sdk](crate::sdk::cpi) method to
    /// either prepare your message or post a message as a program ID emitter.
    pub fn init_message_v1(ctx: Context<InitMessageV1>, args: InitMessageV1Args) -> Result<()> {
        processor::init_message_v1(ctx, args)
    }

    /// Processor used to write to a draft [PostedMessageV1](crate::state::PostedMessageV1) account.
    /// This instruction requires an authority (the emitter authority) to interact with the message
    /// account.
    pub fn write_message_v1(ctx: Context<WriteMessageV1>, args: WriteMessageV1Args) -> Result<()> {
        processor::write_message_v1(ctx, args)
    }

    /// Processor used to finalize a draft [PostedMessageV1](crate::state::PostedMessageV1) account.
    /// Once finalized, this message account cannot be written to again. A finalized message is the
    /// only state the legacy post message instruction can accept before publishing. This
    /// instruction requires an authority (the emitter authority) to interact with the message
    /// account.
    pub fn finalize_message_v1(ctx: Context<FinalizeMessageV1>) -> Result<()> {
        processor::finalize_message_v1(ctx)
    }

    /// Processor used to process a draft [PostedMessageV1](crate::state::PostedMessageV1) account.
    /// This instruction requires an authority (the emitter authority) to interact with the message
    /// account.
    pub fn close_message_v1(ctx: Context<CloseMessageV1>) -> Result<()> {
        processor::close_message_v1(ctx)
    }

    /// Processor used to intialize a created account as [EncodedVaa](crate::state::EncodedVaa). An
    /// authority (the write authority) is established with this instruction.
    pub fn init_encoded_vaa(ctx: Context<InitEncodedVaa>) -> Result<()> {
        processor::init_encoded_vaa(ctx)
    }

    /// Processor used to close an [EncodedVaa](crate::state::EncodedVaa). This instruction requires
    /// an authority (the write authority) to interact witht he encoded VAA account.
    pub fn close_encoded_vaa(ctx: Context<CloseEncodedVaa>) -> Result<()> {
        processor::close_encoded_vaa(ctx)
    }

    /// Processor used to write to an [EncodedVaa](crate::state::EncodedVaa) account. This
    /// instruction requires an authority (the write authority) to interact with the encoded VAA
    /// account.
    pub fn write_encoded_vaa(
        ctx: Context<WriteEncodedVaa>,
        args: WriteEncodedVaaArgs,
    ) -> Result<()> {
        processor::write_encoded_vaa(ctx, args)
    }

    /// Processor used to verify an [EncodedVaa](crate::state::EncodedVaa) account as a version 1
    /// VAA (guardian signatures attesting to this observation). This instruction requires an
    /// authority (the write authority) to interact with the encoded VAA account.
    pub fn verify_encoded_vaa_v1(ctx: Context<VerifyEncodedVaaV1>) -> Result<()> {
        processor::verify_encoded_vaa_v1(ctx)
    }

    /// Processor used to close an [EncodedVaa](crate::state::EncodedVaa) account to create a
    /// [PostedMessageV1](crate::state::PostedMessageV1) account in its place.
    ///
    /// NOTE: Because the legacy verify signatures instruction was not required for the Posted VAA
    /// account to exist, the encoded [SignatureSet](crate::state::SignatureSet) is the default
    /// [Pubkey].
    pub fn post_vaa_v1(ctx: Context<PostVaaV1>) -> Result<()> {
        processor::post_vaa_v1(ctx)
    }

    /// Processor used to close a [PostedMessageV1](crate::state::PostedMessageV1) account. If a
    /// [SignatureSet](crate::state::SignatureSet) were used to verify the VAA, that account will be
    /// closed, too.
    pub fn close_posted_vaa_v1(ctx: Context<ClosePostedVaaV1>) -> Result<()> {
        processor::close_posted_vaa_v1(ctx)
    }

    /// Process legacy Core Bridge instructions. See [legacy](crate::legacy) for more info.
    pub fn process_legacy_instruction(
        program_id: &Pubkey,
        account_infos: &[AccountInfo],
        ix_data: &[u8],
    ) -> Result<()> {
        legacy::process_legacy_instruction(program_id, account_infos, ix_data)
    }
}
