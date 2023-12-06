use crate::{
    error::CoreBridgeError,
    legacy::{instruction::PostMessageArgs, utils::LegacyAnchorized},
    state::{
        Config, LegacyEmitterSequence, PostedMessageV1Data, PostedMessageV1Info,
        PostedMessageV1Unreliable,
    },
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
#[instruction(_nonce: u32, payload_len: u32)]
pub struct PostMessageUnreliable<'info> {
    /// Core Bridge config account (mut).
    ///
    /// Seeds = \["Bridge"\], seeds::program = core_bridge_program.
    ///
    /// This account is used to determine how many lamports to transfer for Wormhole fee.
    #[account(
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<Config>>,

    /// Core Bridge Message (mut).
    ///
    /// CHECK: This message will be created if it does not exist.
    ///
    /// NOTE: This space requirement enforces that the payload length is the same for every call to
    /// this instruction handler. So for unreliable message accounts already created, this
    /// implicitly prevents the payload size from changing.
    #[account(
        init_if_needed,
        payer = payer,
        space = PostedMessageV1Unreliable::compute_size(payload_len.try_into().unwrap()),
    )]
    message: Account<'info, LegacyAnchorized<PostedMessageV1Unreliable>>,

    /// Core Bridge Emitter (read-only signer).
    ///
    /// This account pubkey will be used as the emitter address.
    emitter: Signer<'info>,

    /// Core Bridge Emitter Sequence (mut).
    ///
    /// Seeds = \["Sequence", emitter.key\], seeds::program = core_bridge_program.
    ///
    /// This account is used to determine the sequence of the Wormhole message for the
    /// provided emitter.
    ///
    /// CHECK: This account will be created in the instruction handler if it does not exist. Because
    /// legacy emitter sequence accounts are 8 bytes, these accounts need to be migrated to the new
    /// schema, which just extends the account size to indicate the type of emitter.
    #[account(
        mut,
        seeds = [
            LegacyEmitterSequence::SEED_PREFIX,
            emitter.key().as_ref()
        ],
        bump,
    )]
    emitter_sequence: AccountInfo<'info>,

    /// Payer (mut signer).
    ///
    /// This account pays for new accounts created and pays for the Wormhole fee.
    #[account(mut)]
    payer: Signer<'info>,

    /// Core Bridge Fee Collector (optional, read-only).
    ///
    /// Seeds = \["fee_collector"\], seeds::program = core_bridge_program.
    ///
    /// CHECK: This account is used to collect fees.
    #[account(
        seeds = [crate::constants::FEE_COLLECTOR_SEED_PREFIX],
        bump,
    )]
    fee_collector: Option<AccountInfo<'info>>,

    /// Previously needed sysvar.
    ///
    /// CHECK: This account is unchecked.
    _clock: UncheckedAccount<'info>,

    /// System Program.
    ///
    /// Required to create accounts and transfer lamports to the fee collector.
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
    // Take the message fee amount from the payer.
    super::handle_message_fee(
        &ctx.accounts.config,
        &ctx.accounts.payer,
        &ctx.accounts.fee_collector,
        &ctx.accounts.system_program,
    )?;

    // Check emitter sequence account. If it does not exist, create it. Otherwise realloc the
    // account if it is a legacy emitter sequence account.
    let mut emitter_sequence = super::create_or_realloc_emitter_sequence(
        &ctx.accounts.emitter_sequence,
        &ctx.accounts.payer,
        &ctx.accounts.system_program,
        &ctx.accounts.emitter.key(),
        ctx.bumps["emitter_sequence"],
    )?;

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

    // Finally set the `message` account with posted data.
    ctx.accounts.message.set_inner(
        PostedMessageV1Unreliable::from(PostedMessageV1Data {
            info: PostedMessageV1Info {
                consistency_level: commitment.into(),
                emitter_authority: Default::default(),
                status: crate::legacy::state::MessageStatus::Published,
                _gap_0: Default::default(),
                posted_timestamp: Clock::get().map(Into::into)?,
                nonce,
                sequence: emitter_sequence.value,
                solana_chain_id: Default::default(),
                emitter: ctx.accounts.emitter.key(),
            },
            payload,
        })
        .into(),
    );

    // Even though integrators should be reading the message account data to determine its sequence
    // number instead of using program logs, we need to keep this log here for backwards
    // compatibility.
    msg!("Sequence: {}", emitter_sequence.value);

    // Update emitter sequence account with incremented value.
    {
        emitter_sequence.value += 1;

        let acc_data: &mut [_] = &mut ctx.accounts.emitter_sequence.data.borrow_mut();
        let mut writer = std::io::Cursor::new(acc_data);
        emitter_sequence.try_serialize(&mut writer)?;
    }

    // Done.
    Ok(())
}
