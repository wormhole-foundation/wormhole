use crate::{
    error::CoreBridgeError,
    legacy::{instruction::PostMessageArgs, utils::LegacyAnchorized},
    state::{Config, EmitterSequence, PostedMessageV1Data, PostedMessageV1Unreliable},
    zero_copy::LoadZeroCopy,
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
#[instruction(_nonce: u32, payload_len: u32)]
pub struct PostMessageUnreliable<'info> {
    /// This account is needed to determine how many lamports to transfer from the payer for the
    /// message fee (if there is one).
    #[account(
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<Config>>,

    /// CHECK: This message account is observed by the Guardians.
    ///
    /// NOTE: This space requirement enforces that the payload length is the same for every call to
    /// this instruction handler.
    #[account(
        init_if_needed,
        payer = payer,
        space = try_compute_size(message, payload_len)?,
    )]
    message: Account<'info, LegacyAnchorized<PostedMessageV1Unreliable>>,

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
    emitter_sequence: Account<'info, LegacyAnchorized<EmitterSequence>>,

    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Fee collector, which is used to update the [Config] account with the most up-to-date
    /// last lamports on this account.
    #[account(
        seeds = [crate::constants::FEE_COLLECTOR_SEED_PREFIX],
        bump,
    )]
    fee_collector: Option<AccountInfo<'info>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, PostMessageArgs>
    for PostMessageUnreliable<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyPostMessageUnreliable";

    const ANCHOR_IX_FN: fn(Context<Self>, PostMessageArgs) -> Result<()> = post_message_unreliable;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_post_message_account_infos(account_infos)
    }
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

/// Processor to post (publish) a Wormhole message by setting up the message account for
/// Guardian observation. This message account has either been created already or is created in
/// this call.
///
/// If this message account already exists, the emitter must be the same as the one encoded in
/// the message and the payload must be the same size.
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

    let info = super::new_posted_message_info(
        &ctx.accounts.config,
        super::MessageFeeContext {
            payer: &ctx.accounts.payer,
            fee_collector: &ctx.accounts.fee_collector,
            system_program: &ctx.accounts.system_program,
        },
        &mut ctx.accounts.emitter_sequence,
        commitment.into(),
        nonce,
        &ctx.accounts.emitter.key(),
    )?;

    // Finally set the `message` account with posted data.
    ctx.accounts
        .message
        .set_inner(PostedMessageV1Unreliable::from(PostedMessageV1Data { info, payload }).into());

    // Done.
    Ok(())
}

/// This method is used to compute the size of the message account. Because Anchor's
/// `init_if_needed` checks the size of this account whether it has been created or not, we need to
/// yield either the size determined by the posted message's payload size or the size of the new
/// payload. Instead of reverting with `ConstraintSpace`, we revert with a custom Core Bridge error
/// saying that the payload size does not match the existing one (which is a requirement to reuse
/// this message account).
fn try_compute_size(msg_acc_info: &AccountInfo, payload_size: u32) -> Result<usize> {
    let payload_size = usize::try_from(payload_size).unwrap();

    if !msg_acc_info.data_is_empty() {
        let msg = crate::zero_copy::MessageAccount::load(msg_acc_info)?;

        match msg.v1_unreliable() {
            Some(inner) => {
                require_eq!(
                    payload_size,
                    inner.payload_size(),
                    CoreBridgeError::PayloadSizeMismatch
                )
            }
            _ => return err!(ErrorCode::AccountDidNotDeserialize),
        }
    }

    Ok(PostedMessageV1Unreliable::compute_size(payload_size))
}
