mod unreliable;
pub use unreliable::*;

use crate::{
    error::CoreBridgeError,
    legacy::{instruction::PostMessageArgs, utils::LegacyAnchorized},
    state::{
        Config, EmitterSequence, MessageStatus, PostedMessageV1, PostedMessageV1Data,
        PostedMessageV1Info,
    },
    utils::CreateAccount,
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
#[instruction(args: PostMessageArgs)]
pub struct PostMessage<'info> {
    /// This account is needed to determine whether the Core Bridge fee has been paid.
    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<0, Config>>,

    /// CHECK: This message account is observed by the Guardians.
    ///
    /// NOTE: We do not use the convenient Anchor `init` account macro command here because a
    /// message can either be created at this point or prepared beforehand. If the message has not
    /// been created yet, the instruction handler will create this account (and in this case, the
    /// message account will be required as a signer).
    #[account(mut)]
    message: AccountInfo<'info>,

    /// The emitter of the Core Bridge message. This account is typically an integrating program's
    /// PDA which signs for this instruction. But if a message is already prepared by this point
    /// using `init_message_v1` and `process_message_v1`, then this account is not checked.
    emitter: Option<AccountInfo<'info>>,

    /// CHECK: Sequence tracker for given emitter. Every Core Bridge message is tagged with a unique
    /// sequence number.
    ///
    /// NOTE: Because the emitter can either be the emitter defined in this account context (for new
    /// messages) or written to the message account when it was prepared beforehand, we use a custom
    /// function to help determine this PDA's seeds.
    #[account(
        mut,
        seeds = [
            EmitterSequence::SEED_PREFIX,
            find_emitter_for_sequence(&emitter, &message)?.as_ref()
        ],
        bump
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
    // ...
    //
    /// CHECK: This might be the System program.
    maybe_system_program_1: AccountInfo<'info>,

    /// CHECK: Or this might be, who knows?
    maybe_system_program_2: AccountInfo<'info>,
}

impl<'info> crate::utils::CreateAccount<'info> for PostMessage<'info> {
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

impl<'info> PostMessage<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
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

        // Done.
        Ok(())
    }
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, PostMessageArgs>
    for PostMessage<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyPostMessage";

    const ANCHOR_IX_FN: fn(Context<Self>, PostMessageArgs) -> Result<()> = post_message;
}

/// This method is used by both `post_message` and `post_message_unreliable` instruction handlers.
/// It handles the message fee check on the fee collector, upticks the emitter sequence number and
/// returns the posted message data, which will be serialized to either `PostedMessageV1` or
/// `PostedMessageV1Unreliable` depending on which instruction handler called this method.
pub(super) fn new_posted_message_data(
    config: &mut Account<LegacyAnchorized<0, Config>>,
    fee_collector: &Option<AccountInfo>,
    emitter_sequence: &mut AccountInfo,
    consistency_level: u8,
    nonce: u32,
    emitter: &Pubkey,
    payload: Vec<u8>,
) -> Result<PostedMessageV1Data> {
    // Determine whether fee has been paid. Update core bridge config account if so.
    //
    // NOTE: This is inconsistent with other Core Bridge implementations, where we would check that
    // the change would equal exactly the fee amount.
    handle_message_fee(config, fee_collector)?;

    // Sequence number will be used later on.
    let sequence = take_and_uptick_sequence(emitter_sequence)?;

    // Finally set the `message` account with posted data.
    let data = PostedMessageV1Data {
        info: PostedMessageV1Info {
            consistency_level,
            emitter_authority: Default::default(),
            status: MessageStatus::Unset,
            _gap_0: Default::default(),
            posted_timestamp: Clock::get().map(Into::into)?,
            nonce,
            sequence,
            solana_chain_id: Default::default(),
            emitter: *emitter,
        },
        payload,
    };

    // Done.
    Ok(data)
}

pub(super) fn create_emitter_sequence<'info, A>(
    accounts: &A,
    seq_acc_info: AccountInfo<'info>,
    emitter: &Pubkey,
    emitter_bump: u8,
) -> Result<()>
where
    A: CreateAccount<'info>,
{
    crate::utils::create_account(
        accounts,
        seq_acc_info.to_account_info(),
        EmitterSequence::INIT_SPACE,
        &crate::ID,
        Some(&[&[
            EmitterSequence::SEED_PREFIX,
            emitter.as_ref(),
            &[emitter_bump],
        ]]),
    )?;

    let acc_data: &mut [u8] = &mut seq_acc_info.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);
    LegacyAnchorized::from(EmitterSequence { value: 0 })
        .try_serialize(&mut writer)
        .map_err(Into::into)
}

/// Processor to post (publish) a Wormhole message by setting up the message account for
/// Guardian observation.
///
/// A message is either created beforehand using the new Anchor instruction to process a message
/// or is created at this point.
#[access_control(PostMessage::constraints(&ctx))]
fn post_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
    // Create the emitter sequence account if it doesn't exist.
    if ctx.accounts.emitter_sequence.data_is_empty() {
        create_emitter_sequence(
            ctx.accounts,
            ctx.accounts.emitter_sequence.to_account_info(),
            &find_emitter_for_sequence(&ctx.accounts.emitter, &ctx.accounts.message).unwrap(),
            ctx.bumps["emitter_sequence"],
        )?;
    }

    if ctx.accounts.message.data_is_empty() {
        handle_post_new_message(ctx, args)
    } else {
        handle_post_prepared_message(ctx, args)
    }
}

/// When posting a new message, the message account must first be created. The new message data is
/// then serialized into this account.
fn handle_post_new_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
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

    let payload_size = payload.len();

    // Generate message data.
    let data = new_posted_message_data(
        &mut ctx.accounts.config,
        &ctx.accounts.fee_collector,
        &mut ctx.accounts.emitter_sequence,
        commitment.into(),
        nonce,
        &ctx.accounts.emitter.as_ref().unwrap().key(),
        payload,
    )?;

    // NOTE: The legacy instruction had the note "DO NOT REMOVE - CRITICAL OUTPUT". But we may be
    // able to remove this to save on compute units.
    msg!("Sequence: {}", data.sequence);

    // Finally create account and set the message data.
    crate::utils::create_account(
        ctx.accounts,
        ctx.accounts.message.to_account_info(),
        PostedMessageV1::compute_size(payload_size),
        &crate::ID,
        None,
    )?;

    let msg_acc_data: &mut [u8] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    // Finally set the `message` account with posted data.
    LegacyAnchorized::from(PostedMessageV1 { data })
        .try_serialize(&mut writer)
        .map_err(Into::into)
}

/// When posting a prepared message, the `MessageStatus` must be in a `Finalized` state (indicating
/// that the emitter authority has finished writing this message). We disallow a new payload to be
/// used at this point, so we require that this argument be an empty vector. The message data is
/// modified to reflect posting this message (timestamp, sequence number, etc.).
fn handle_post_prepared_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
    msg!("MessageStatus: Finalized");

    let PostMessageArgs {
        nonce: _,
        payload: unnecessary_payload,
        commitment: _,
    } = args;

    // The payload argument is not allowed if the message has been prepared beforehand.
    require!(
        unnecessary_payload.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let (consistency_level, nonce, emitter, payload) = {
        let acc_data = ctx.accounts.message.data.borrow();
        let msg = crate::zero_copy::PostedMessageV1::parse(&acc_data).unwrap();

        (
            msg.consistency_level(),
            msg.nonce(),
            msg.emitter(),
            msg.payload().to_vec(),
        )
    };

    let data = new_posted_message_data(
        &mut ctx.accounts.config,
        &ctx.accounts.fee_collector,
        &mut ctx.accounts.emitter_sequence,
        consistency_level,
        nonce,
        &emitter,
        payload,
    )?;

    let msg_acc_data: &mut [u8] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    // Finally set the `message` account with posted data.
    LegacyAnchorized::from(PostedMessageV1 { data })
        .try_serialize(&mut writer)
        .map_err(Into::into)
}

/// If there is a fee, check the fee collector account to ensure that the fee has been paid.
fn handle_message_fee(
    config: &mut Account<LegacyAnchorized<0, Config>>,
    fee_collector: &Option<AccountInfo>,
) -> Result<()> {
    if config.fee_lamports > 0 {
        let fee_collector = fee_collector
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))?;

        let collector_lamports = fee_collector.lamports();
        require_eq!(
            collector_lamports,
            config.last_lamports.saturating_add(config.fee_lamports),
            CoreBridgeError::InsufficientFees
        );

        // Update core bridge config to reflect paid fees.
        config.last_lamports = collector_lamports;
    }

    // Done.
    Ok(())
}

/// For posting a message, either a message has been prepared beforehand or this account is created
/// at this point in time. We make the assumption that if the status is unset, it is a message
/// account created at this point, which is the way the legacy post message instruction handler
/// worked.
///
/// The legacy post message instruction handler did not allow posting a message as a program,
/// which `init_message_v1` now enables integrators to do. So the emitter sequence account, whose
/// PDA address is derived using the emitter, is assigned to the emitter signer (now called the
/// emitter authority). Whereas with the new prepared message, this emitter can be taken from the
/// message account to re-derive the emitter sequence PDA address.
fn find_emitter_for_sequence(emitter: &Option<AccountInfo>, msg: &AccountInfo) -> Result<Pubkey> {
    if msg.data_is_empty() {
        // Message must be a signer in order to be created.
        require!(msg.is_signer, ErrorCode::AccountNotSigner);

        // Because this message will be newly created in this instruction, the emitter is required
        // and must be a signer to authorize posting this message.
        let emitter = emitter
            .as_ref()
            .ok_or_else(|| error!(ErrorCode::AccountNotEnoughKeys))?;
        require!(emitter.is_signer, ErrorCode::AccountNotSigner);

        Ok(emitter.key())
    } else {
        let msg_acc_data = msg.data.borrow();
        let msg = crate::zero_copy::PostedMessageV1::parse(&msg_acc_data)?;

        match msg.status() {
            MessageStatus::Unset => err!(CoreBridgeError::MessageAlreadyPublished),
            MessageStatus::Writing => err!(CoreBridgeError::InWritingStatus),
            MessageStatus::Finalized => Ok(msg.emitter()),
        }
    }
}

fn take_and_uptick_sequence(seq_acc_info: &mut AccountInfo) -> Result<u64> {
    // Take.
    let sequence = {
        let acc_data = seq_acc_info.data.borrow();
        let emitter_sequence =
            LegacyAnchorized::<0, EmitterSequence>::try_deserialize(&mut acc_data.as_ref())?;

        emitter_sequence.value
    };

    // And uptick.
    let acc_data: &mut [u8] = &mut seq_acc_info.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);
    LegacyAnchorized::from(EmitterSequence {
        value: sequence + 1,
    })
    .try_serialize(&mut writer)?;

    // Done.
    Ok(sequence)
}
