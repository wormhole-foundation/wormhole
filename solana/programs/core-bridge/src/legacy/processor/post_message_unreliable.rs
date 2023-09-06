use crate::{
    error::CoreBridgeError,
    legacy::{instruction::PostMessageArgs, utils::LegacyAnchorized},
    state::{Config, EmitterSequence, PostedMessageV1Unreliable},
};
use anchor_lang::prelude::*;

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
    config: Account<'info, LegacyAnchorized<0, Config>>,

    /// CHECK: Posted message account data.
    ///
    /// NOTE: This space requirement enforces that the payload length is the same for every call to
    /// this instruction handler.
    #[account(
        init_if_needed,
        payer = payer,
        space = try_compute_size(message, payload_len)?,
    )]
    message: Account<'info, LegacyAnchorized<4, PostedMessageV1Unreliable>>,

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
    emitter_sequence: Account<'info, LegacyAnchorized<0, EmitterSequence>>,

    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Fee collector.
    #[account(
        mut,
        seeds = [crate::constants::FEE_COLLECTOR_SEED_PREFIX],
        bump,
    )]
    fee_collector: Option<AccountInfo<'info>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    system_program: Program<'info, System>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, PostMessageArgs>
    for PostMessageUnreliable<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyPostMessageUnreliable";

    const ANCHOR_IX_FN: fn(Context<Self>, PostMessageArgs) -> Result<()> = post_message_unreliable;
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
                CoreBridgeError::EmitterMismatch
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
fn post_message_unreliable(
    ctx: Context<PostMessageUnreliable>,
    args: PostMessageArgs,
) -> Result<()> {
    let PostMessageArgs {
        nonce,
        payload,
        commitment,
    } = args;

    // Should we require the payload not be empty?
    require!(
        !payload.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let data = crate::legacy::new_posted_message_data(
        &mut ctx.accounts.config,
        &ctx.accounts.fee_collector,
        &mut ctx.accounts.emitter_sequence,
        commitment.into(),
        nonce,
        &ctx.accounts.emitter.key(),
        payload,
    )?;

    // NOTE: The legacy instruction had the note "DO NOT REMOVE - CRITICAL OUTPUT". But we may be
    // able to remove this to save on compute units.
    msg!("Sequence: {}", data.sequence);

    // Finally set the `message` account with posted data.
    ctx.accounts
        .message
        .set_inner(PostedMessageV1Unreliable { data }.into());

    // Done.
    Ok(())
}
