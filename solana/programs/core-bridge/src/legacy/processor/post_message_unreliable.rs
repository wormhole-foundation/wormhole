use crate::{
    error::CoreBridgeError,
    legacy::instruction::{PostMessageArgs, PostMessageUnreliableArgs},
    state::{Config, EmitterSequence, FeeCollector, PostedMessageV1Unreliable},
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{NewAccountSize, SeedPrefix};

use super::handle_post_new_message;

#[derive(Accounts)]
#[instruction(_nonce: u32, payload_len: u32)]
pub struct PostMessageUnreliable<'info> {
    /// Bridge program data. This account is needed to determine whether the core bridge fee has
    /// been paid.
    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    bridge: Account<'info, Config>,

    /// Posted message account data.
    ///
    /// NOTE: This space requirement enforces that the payload length is the same for every call to
    /// this instruction handler.
    #[account(
        init_if_needed,
        payer = payer,
        space = try_compute_size(&message, payload_len)?,
    )]
    message: Account<'info, PostedMessageV1Unreliable>,

    /// The emitter of the core bridge message. This account is typically an integrating program's
    /// PDA which signs for this instruction.
    emitter: Signer<'info>,

    /// Sequence tracker for given emitter. Every core bridge message is tagged with a unique
    /// sequence number.
    #[account(
        init_if_needed,
        payer = payer,
        space = EmitterSequence::INIT_SPACE,
        seeds = [EmitterSequence::SEED_PREFIX, emitter.key().as_ref()],
        bump,
    )]
    emitter_sequence: Account<'info, EmitterSequence>,

    #[account(mut)]
    payer: Signer<'info>,

    /// Collect core bridge message fee when posting a message.
    ///
    /// NOTE: This account is optional because we do not need to pay a fee to post a message if the
    /// fee is zero.
    #[account(
        mut,
        seeds = [FeeCollector::SEED_PREFIX],
        bump,
    )]
    fee_collector: Option<Account<'info, FeeCollector>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    system_program: Program<'info, System>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,
}

fn try_compute_size(message: &AccountInfo, payload_size: u32) -> Result<usize> {
    let payload_size = usize::try_from(payload_size).unwrap();

    if !message.data_is_empty() {
        let expected_size =
            crate::zero_copy::PostedMessageV1Unreliable::parse(&message.data.borrow())?
                .payload_size();
        require_eq!(
            payload_size,
            expected_size,
            CoreBridgeError::PayloadSizeMismatch
        );
    }

    Ok(PostedMessageV1Unreliable::compute_size(payload_size))
}

impl<'info> PostMessageUnreliable<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let msg = &ctx.accounts.message;

        // We are checking if the message has an existing payload. We disallow publishing with zero
        // payload, so this check is sufficient.
        if !msg.payload.is_empty() {
            // The emitter must be identical.
            require_keys_eq!(
                ctx.accounts.emitter.key(),
                msg.emitter,
                CoreBridgeError::EmitterAuthorityMismatch
            );
        }

        Ok(())
    }
}

/// This instruction handler is used to post a new message to the core bridge using an existing
/// message account.
///
/// The constraints for posting a message using this instruction handler are:
/// * Emitter must be the same as the message account's emitter.
/// * The new message must be the same size as the existing message's payload.
#[access_control(PostMessageUnreliable::constraints(&ctx))]
pub fn post_message_unreliable(
    ctx: Context<PostMessageUnreliable>,
    args: PostMessageUnreliableArgs,
) -> Result<()> {
    let PostMessageUnreliableArgs {
        nonce,
        payload,
        commitment,
    } = args;

    handle_post_new_message(
        &mut ctx.accounts.bridge,
        &mut ctx.accounts.message,
        &ctx.accounts.emitter,
        &mut ctx.accounts.emitter_sequence,
        &ctx.accounts.fee_collector,
        PostMessageArgs {
            nonce,
            payload,
            commitment,
        },
    )
}
