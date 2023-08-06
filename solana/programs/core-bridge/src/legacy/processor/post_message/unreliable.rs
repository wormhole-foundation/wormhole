use crate::{
    error::CoreBridgeError,
    legacy::{instruction::PostMessageArgs, utils::LegacyAnchorized},
    state::{Config, EmitterSequence, PostedMessageV1Unreliable},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
#[instruction(_nonce: u32, payload_len: u32)]
pub struct PostMessageUnreliable<'info> {
    /// This account is needed to determine whether the Core Bridge fee has been paid.
    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<0, Config>>,

    /// CHECK: This message account is observed by the Guardians.
    ///
    /// NOTE: This space requirement enforces that the payload length is the same for every call to
    /// this instruction handler.
    #[account(
        init_if_needed,
        payer = payer,
        space = try_compute_size(message, payload_len)?,
    )]
    message: Account<'info, LegacyAnchorized<4, PostedMessageV1Unreliable>>,

    /// The emitter of the Core Bridge message. This account is typically an integrating program's
    /// PDA which signs for this instruction.
    emitter: Signer<'info>,

    /// Sequence tracker for given emitter. Every Core Bridge message is tagged with a unique
    /// sequence number.
    #[account(
        init_if_needed,
        payer = payer,
        space = EmitterSequence::INIT_SPACE,
        seeds = [
            EmitterSequence::SEED_PREFIX,
            emitter.key().as_ref()
        ],
        bump,
    )]
    emitter_sequence: Account<'info, LegacyAnchorized<0, EmitterSequence>>,

    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Fee collector, which is used to update the [Config] account with the most up-to-date
    /// last lamports on this account.
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
impl<'info> PostMessageUnreliable<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let msg = &ctx.accounts.message;

        // If the message account already exists, the emitter signing for this instruction must be
        // the same one encoded in this account.
        if !msg.payload.is_empty() {
            require_keys_eq!(
                ctx.accounts.emitter.key(),
                msg.emitter,
                CoreBridgeError::EmitterMismatch
            );
        }

        // Done.
        Ok(())
    }
}

/// Processor to post (publish) a Wormhole message by setting up the message account for Guardian
/// observation. This message account has either been created already or is created in this call. If
/// this message was already created, the emitter must be the same as the one encoded in the message
/// and the payload must be the same size.
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

    let data = super::new_posted_message_data(
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

/// This method is used to compute the size of the message account. Because Anchor's
/// `init_if_needed` checks the size of this account whether it has been created or not, we need to
/// yield either the size determined by the posted message's payload size or the size of the new
/// payload. Instead of reverting with `ConstraintSpace`, we revert with a custom Core Bridge error
/// saying that the payload size does not match the existing one (which is a requirement to reuse
/// this message account).
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
