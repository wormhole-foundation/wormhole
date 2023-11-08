mod unreliable;
pub use unreliable::*;

use crate::{
    error::CoreBridgeError,
    legacy::{
        instruction::PostMessageArgs,
        utils::{LegacyAccount, LegacyAnchorized},
    },
    state::{
        Config, EmitterSequence, MessageStatus, PostedMessageV1, PostedMessageV1Data,
        PostedMessageV1Info,
    },
    utils,
    zero_copy::LoadZeroCopy,
};
use anchor_lang::{prelude::*, system_program};

#[derive(Accounts)]
#[instruction(args: PostMessageArgs)]
pub struct PostMessage<'info> {
    /// This account is needed to determine how many lamports to transfer from the payer for the
    /// message fee (if there is one).
    #[account(
        seeds = [Config::SEED_PREFIX],
        bump,
    )]
    config: Account<'info, LegacyAnchorized<Config>>,

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
    /// using `init_message_v1`, `write_message_v1` and `finalize_message_v1`, then this account is
    /// not checked.
    emitter: Option<AccountInfo<'info>>,

    /// Sequence tracker for given emitter. Every Core Bridge message is tagged with a unique
    /// sequence number.
    ///
    /// NOTE: Because the emitter can either be the emitter defined in this account context (for new
    /// messages) or written to the message account when it was prepared beforehand, we use a custom
    /// function to help determine this PDA's seeds.
    #[account(
        init_if_needed,
        payer = payer,
        space = EmitterSequence::INIT_SPACE,
        seeds = [
            EmitterSequence::SEED_PREFIX,
            find_emitter_for_sequence(&emitter, &message)?.as_ref()
        ],
        bump
    )]
    emitter_sequence: Account<'info, LegacyAnchorized<EmitterSequence>>,

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
}

impl<'info> crate::utils::cpi::CreateAccount<'info> for PostMessage<'info> {
    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }
}

impl<'info> crate::legacy::utils::ProcessLegacyInstruction<'info, PostMessageArgs>
    for PostMessage<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyPostMessage";

    const ANCHOR_IX_FN: fn(Context<Self>, PostMessageArgs) -> Result<()> = post_message;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        order_post_message_account_infos(account_infos)
    }
}

/// The Anchor context orders the accounts as:
///
/// 1. `config`
/// 2. `message`
/// 3. `emitter`
/// 4. `emitter_sequence`
/// 5. `payer`
/// 6. `fee_collector`
/// 7. `clock`
/// 8. `system_program`
///
/// Because the legacy implementation did not require specifying where the System program should be,
/// we ensure that it is account #8 because the Anchor account context requires it to be in this
/// position.
pub(super) fn order_post_message_account_infos<'info>(
    account_infos: &[AccountInfo<'info>],
) -> Result<Vec<AccountInfo<'info>>> {
    const NUM_ACCOUNTS: usize = 8;
    const SYSTEM_PROGRAM_IDX: usize = NUM_ACCOUNTS - 1;

    let mut infos = account_infos.to_vec();

    // We only need to order the account infos if there are more than 8 accounts.
    if infos.len() > NUM_ACCOUNTS {
        // System program needs to exist in these account infos.
        let system_program_idx = SYSTEM_PROGRAM_IDX
            + infos
                .iter()
                .skip(SYSTEM_PROGRAM_IDX)
                .position(|info| info.key() == anchor_lang::system_program::ID)
                .ok_or(error!(ErrorCode::InvalidProgramId))?;

        // Make sure System program is in the right index.
        if system_program_idx != SYSTEM_PROGRAM_IDX {
            infos.swap(SYSTEM_PROGRAM_IDX, system_program_idx);
        }
    }

    Ok(infos)
}

pub(super) struct MessageFeeContext<'ctx, 'info> {
    pub payer: &'ctx AccountInfo<'info>,
    pub fee_collector: &'ctx Option<AccountInfo<'info>>,
    pub system_program: &'ctx Program<'info, System>,
}

/// This method is used by both `post_message` and `post_message_unreliable` instruction handlers.
/// It handles the message fee check on the fee collector, upticks the emitter sequence number and
/// returns the posted message data, which will be serialized to either `PostedMessageV1` or
/// `PostedMessageV1Unreliable` depending on which instruction handler called this method.
pub(super) fn new_posted_message_info<'info>(
    config: &Account<'info, LegacyAnchorized<Config>>,
    message_fee_ctx: MessageFeeContext<'_, 'info>,
    emitter_sequence: &mut Account<'info, LegacyAnchorized<EmitterSequence>>,
    consistency_level: u8,
    nonce: u32,
    emitter: &Pubkey,
) -> Result<PostedMessageV1Info> {
    // Take the message fee amount from the payer.
    handle_message_fee(config, message_fee_ctx)?;

    // Sequence number will be used later on.
    let sequence = emitter_sequence.value;

    // Finally set the `message` account with posted data.
    let info = PostedMessageV1Info {
        consistency_level,
        emitter_authority: Default::default(),
        status: MessageStatus::Published,
        _gap_0: Default::default(),
        posted_timestamp: Clock::get().map(Into::into)?,
        nonce,
        sequence,
        solana_chain_id: Default::default(),
        emitter: *emitter,
    };

    // Increment emitter sequence value.
    emitter_sequence.value += 1;

    // Done.
    Ok(info)
}

/// Processor to post (publish) a Wormhole message by setting up the message account for
/// Guardian observation.
///
/// A message is either created beforehand using the new Anchor instructions `init_message_v1`
/// and `process_message_v1` or is created at this point.
fn post_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
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

    // Create the account.
    {
        utils::cpi::create_account(
            ctx.accounts,
            &ctx.accounts.message,
            PostedMessageV1::compute_size(payload.len()),
            &crate::ID,
            None,
        )?;
    }

    let info = new_posted_message_info(
        &ctx.accounts.config,
        MessageFeeContext {
            payer: &ctx.accounts.payer,
            fee_collector: &ctx.accounts.fee_collector,
            system_program: &ctx.accounts.system_program,
        },
        &mut ctx.accounts.emitter_sequence,
        commitment.into(),
        nonce,
        &ctx.accounts.emitter.as_ref().unwrap().key(),
    )?;

    let msg_acc_data: &mut [_] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    // Finally set the `message` account with posted data.
    LegacyAnchorized::from(PostedMessageV1::from(PostedMessageV1Data { info, payload }))
        .try_serialize(&mut writer)?;

    // Done.
    Ok(())
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

    let (consistency_level, nonce, emitter) = {
        let msg = crate::zero_copy::PostedMessageV1::load(&ctx.accounts.message).unwrap();

        (msg.consistency_level(), msg.nonce(), msg.emitter())
    };

    let info = new_posted_message_info(
        &ctx.accounts.config,
        MessageFeeContext {
            payer: &ctx.accounts.payer,
            fee_collector: &ctx.accounts.fee_collector,
            system_program: &ctx.accounts.system_program,
        },
        &mut ctx.accounts.emitter_sequence,
        consistency_level,
        nonce,
        &emitter,
    )?;

    let msg_acc_data: &mut [_] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    std::io::Write::write_all(&mut writer, PostedMessageV1::DISCRIMINATOR)?;
    info.serialize(&mut writer)?;

    // Done.
    Ok(())
}

/// If there is a fee, check the fee collector account to ensure that the fee has been paid.
fn handle_message_fee<'info>(
    config: &Account<'info, LegacyAnchorized<Config>>,
    message_fee_ctx: MessageFeeContext<'_, 'info>,
) -> Result<()> {
    if config.fee_lamports > 0 {
        let MessageFeeContext {
            payer,
            fee_collector,
            system_program,
        } = message_fee_ctx;

        let fee_collector = fee_collector
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys))?;

        // In the old implementation, integrators were expected to pay the fee outside of this
        // instruction and this instruction handler had to check that the lamports on the fee
        // collector account were at least as much as the last lamports in the config plus the fee
        // amount.
        //
        // Now we just transfer the lamports from the payer to the fee collector for the exact fee
        // amount.
        system_program::transfer(
            CpiContext::new(
                system_program.to_account_info(),
                system_program::Transfer {
                    from: payer.to_account_info(),
                    to: fee_collector.to_account_info(),
                },
            ),
            config.fee_lamports,
        )
    } else {
        // Nothing to do.
        Ok(())
    }
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
fn find_emitter_for_sequence(
    emitter: &Option<AccountInfo>,
    msg_acc_info: &AccountInfo,
) -> Result<Pubkey> {
    if msg_acc_info.data_is_empty() {
        // Message must be a signer in order to be created.
        require!(msg_acc_info.is_signer, ErrorCode::AccountNotSigner);

        // Because this message will be newly created in this instruction, the emitter is required
        // and must be a signer to authorize posting this message.
        let emitter = emitter
            .as_ref()
            .ok_or_else(|| error!(ErrorCode::AccountNotEnoughKeys))?;
        require!(emitter.is_signer, ErrorCode::AccountNotSigner);

        Ok(emitter.key())
    } else {
        let msg = crate::zero_copy::PostedMessageV1::load(msg_acc_info)?;

        match msg.status() {
            MessageStatus::Published => err!(CoreBridgeError::MessageAlreadyPublished),
            MessageStatus::Writing => err!(CoreBridgeError::InWritingStatus),
            MessageStatus::ReadyForPublishing => Ok(msg.emitter()),
        }
    }
}
