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
        mut
        // init_if_needed,
        // payer = payer,
        // space = try_compute_size(message, payload_len)?,
    )]
    message: AccountInfo<'info>,

    /// The emitter of the Core Bridge message. This account is typically an integrating program's
    /// PDA which signs for this instruction.
    emitter: Signer<'info>,

    /// CHECK: Sequence tracker for given emitter. Every Core Bridge message is tagged with a unique
    /// sequence number.
    #[account(
        mut,
        seeds = [
            EmitterSequence::SEED_PREFIX,
            emitter.key().as_ref()
        ],
        bump,
    )]
    emitter_sequence: AccountInfo<'info>,

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

    // Below there be dragons....
    //
    // Current integrators of the legacy implementation could have interchanged the System program
    // and Rent sysvar accounts when passing AccountMetas for this instruction. So we need to check
    // which account is the actual System program.
    //
    // ... and here we go!
    //
    /// CHECK: This might be the System program.
    maybe_system_program_1: AccountInfo<'info>,

    /// CHECK: Or this might be, who knows?
    maybe_system_program_2: AccountInfo<'info>,
}

impl<'info> crate::utils::CreateAccount<'info> for PostMessageUnreliable<'info> {
    fn system_program(&self) -> AccountInfo<'info> {
        // Don't look here for any safeties. We will guarantee that one of these account infos is
        // the system program in access control.
        if self.maybe_system_program_1.key() == anchor_lang::system_program::ID {
            self.maybe_system_program_1.to_account_info()
        } else {
            self.maybe_system_program_2.to_account_info()
        }
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, PostMessageArgs>
    for PostMessageUnreliable<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyPostMessageUnreliable";

    const ANCHOR_IX_FN: fn(Context<Self>, PostMessageArgs) -> Result<()> = post_message_unreliable;
}

impl<'info> PostMessageUnreliable<'info> {
    fn constraints(ctx: &Context<Self>, args: &PostMessageArgs) -> Result<()> {
        // One of these account infos must be the system account.
        if ctx.accounts.maybe_system_program_1.key() == anchor_lang::system_program::ID {
            // We're good.
        } else {
            // We revert with a specific error if the last account info is not the system program.
            require_keys_eq!(
                ctx.accounts.maybe_system_program_2.key(),
                anchor_lang::system_program::ID,
                ErrorCode::InvalidProgramId
            );
        }

        if !ctx.accounts.message.data_is_empty() {
            let acc_data = ctx.accounts.message.data.borrow();
            let msg = crate::zero_copy::PostedMessageV1Unreliable::parse(&acc_data)?;

            // The new payload must be the same size as the existing one.
            require_eq!(
                args.payload.len(),
                msg.payload_size(),
                CoreBridgeError::PayloadSizeMismatch
            );

            // The emitter signing for this instruction must be the same one encoded in this
            // account.
            require_keys_eq!(
                ctx.accounts.emitter.key(),
                msg.emitter(),
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
#[access_control(PostMessageUnreliable::constraints(&ctx, &args))]
fn post_message_unreliable(
    ctx: Context<PostMessageUnreliable>,
    args: PostMessageArgs,
) -> Result<()> {
    // Create the emitter sequence account if it doesn't exist.
    if ctx.accounts.emitter_sequence.data_is_empty() {
        super::create_emitter_sequence(
            ctx.accounts,
            ctx.accounts.emitter_sequence.to_account_info(),
            &ctx.accounts.emitter.key(),
            ctx.bumps["emitter_sequence"],
        )?;
    }
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

    // Save for later.
    let payload_size = payload.len();

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

    // Create the message account if it doesn't exist.
    if ctx.accounts.message.data_is_empty() {
        crate::utils::create_account(
            ctx.accounts,
            ctx.accounts.message.to_account_info(),
            PostedMessageV1Unreliable::compute_size(payload_size),
            &crate::ID,
            None,
        )?;
    }

    let msg_acc_data: &mut [u8] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    // Finally set the `message` account with posted data.
    LegacyAnchorized::from(PostedMessageV1Unreliable { data })
        .try_serialize(&mut writer)
        .map_err(Into::into)
}
