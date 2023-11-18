mod unreliable;
pub use unreliable::*;

use crate::{
    error::CoreBridgeError,
    legacy::{
        instruction::PostMessageArgs,
        utils::{LegacyAccount, LegacyAnchorized},
    },
    state::{
        Config, EmitterSequence, EmitterType, LegacyEmitterSequence, MessageStatus,
        PostedMessageV1, PostedMessageV1Data, PostedMessageV1Info,
    },
    utils,
};
use anchor_lang::{prelude::*, system_program};

#[derive(Accounts)]
#[instruction(args: PostMessageArgs)]
pub struct PostMessage<'info> {
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
    /// NOTE: We do not use the convenient Anchor `init` account macro command here because a
    /// message can either be created at this point or prepared beforehand. If the message has not
    /// been created yet, the instruction handler will create this account (and in this case, the
    /// message account will be required as a signer).
    #[account(mut)]
    message: AccountInfo<'info>,

    /// Core Bridge Emitter (optional, read-only signer).
    ///
    /// This account pubkey will be used as the emitter address. This account is required
    /// if the message account has not been prepared beforehand.
    emitter: Option<Signer<'info>>,

    /// Core Bridge Emitter Sequence (mut).
    ///
    /// Seeds = \["Sequence", emitter.key\], seeds::program = core_bridge_program.
    ///
    /// This account is used to determine the sequence of the Wormhole message for the
    /// provided emitter.
    ///
    /// NOTE: Because the emitter can either be the emitter defined in this account context (for new
    /// messages) or written to the message account when it was prepared beforehand, we use a custom
    /// function to help determine this PDA's seeds.
    ///
    /// CHECK: This account will be created in the instruction handler if it does not exist. Because
    /// legacy emitter sequence accounts are 8 bytes, these accounts need to be migrated to the new
    /// schema, which just extends the account size to indicate the type of emitter.
    #[account(
        mut,
        seeds = [
            EmitterSequence::SEED_PREFIX,
            try_emitter_seed(&emitter, &message)?.as_ref()
        ],
        bump
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
        mut,
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
pub(self) fn order_post_message_account_infos<'info>(
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

/// Processor to post (publish) a Wormhole message by setting up the message account for
/// Guardian observation.
///
/// A message is either created beforehand using the new Anchor instructions `init_message_v1`
/// and `process_message_v1` or is created at this point.
fn post_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
    // Take the message fee amount from the payer.
    handle_message_fee(
        &ctx.accounts.config,
        &ctx.accounts.payer,
        &ctx.accounts.fee_collector,
        &ctx.accounts.system_program,
    )?;

    if ctx.accounts.message.data_is_empty() {
        handle_post_new_message(ctx, args)
    } else {
        handle_post_prepared_message(ctx, args)
    }
}

/// When posting a new message, the message account must first be created. The new message data is
/// then serialized into this account.
fn handle_post_new_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
    let emitter = ctx
        .accounts
        .emitter
        .as_ref()
        .ok_or(error!(ErrorCode::AccountNotEnoughKeys).with_account_name("emitter"))?;

    // Check emitter sequence account. If it does not exist, create it. Otherwise realloc the
    // account if it is a legacy emitter sequence account.
    let mut emitter_sequence = create_or_realloc_emitter_sequence(
        &ctx.accounts.emitter_sequence,
        &ctx.accounts.payer,
        &ctx.accounts.system_program,
        &emitter.key(),
        ctx.bumps["emitter_sequence"],
    )?;

    require!(
        emitter_sequence.emitter_type != EmitterType::Executable,
        CoreBridgeError::ExecutableEmitter
    );

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
    utils::cpi::create_account_safe(
        CpiContext::new(
            ctx.accounts.system_program.to_account_info(),
            utils::cpi::CreateAccountSafe {
                payer: ctx.accounts.payer.to_account_info(),
                new_account: ctx.accounts.message.to_account_info(),
            },
        ),
        PostedMessageV1::compute_size(payload.len()),
        &crate::ID,
    )?;

    // Prepare message data.
    let message = PostedMessageV1::from(PostedMessageV1Data {
        info: PostedMessageV1Info {
            consistency_level: commitment.into(),
            emitter_authority: Default::default(),
            status: crate::legacy::state::MessageStatus::Published,
            _gap_0: Default::default(),
            posted_timestamp: Clock::get().map(Into::into)?,
            nonce,
            sequence: emitter_sequence.value,
            solana_chain_id: Default::default(),
            emitter: emitter.key(),
        },
        payload,
    });

    // Update emitter sequence account with incremented value.
    {
        emitter_sequence.value += 1;

        let acc_data: &mut [_] = &mut ctx.accounts.emitter_sequence.data.borrow_mut();
        let mut writer = std::io::Cursor::new(acc_data);
        emitter_sequence.try_serialize(&mut writer)?;
    }

    let msg_acc_data: &mut [_] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    // Finally set the `message` account with posted data.
    LegacyAnchorized::from(message).try_serialize(&mut writer)?;

    // Done.
    Ok(())
}

/// When posting a prepared message, the `MessageStatus` must be in a `Finalized` state (indicating
/// that the emitter authority has finished writing this message). We disallow a new payload to be
/// used at this point, so we require that this argument be an empty vector. The message data is
/// modified to reflect posting this message (timestamp, sequence number, etc.).
fn handle_post_prepared_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
    msg!("MessageStatus: ReadyForPublishing");

    // Because the message account was prepared by the Core Bridge, we need to double-check that
    // the Core Bridge is the owner.
    require_keys_eq!(
        *ctx.accounts.message.owner,
        crate::ID,
        ErrorCode::ConstraintOwner
    );

    // Check message header. This is mutable because we will rewrite to the mesasge account with
    // some modified values later on.
    let mut info = PostedMessageV1::try_deserialize_info(&ctx.accounts.message)?;

    // Make sure the message is ready to be published.
    require!(
        info.status == MessageStatus::ReadyForPublishing,
        CoreBridgeError::NotReadyForPublishing
    );

    // Check emitter sequence account. If it does not exist, create it. Otherwise realloc the
    // account if it is a legacy emitter sequence account.
    let mut emitter_sequence = create_or_realloc_emitter_sequence(
        &ctx.accounts.emitter_sequence,
        &ctx.accounts.payer,
        &ctx.accounts.system_program,
        &info.emitter,
        ctx.bumps["emitter_sequence"],
    )?;

    // If the emitter is the same as the emitter authority, this message's emitter is a legacy
    // emitter. Otherwise it is an executable.
    if info.emitter == info.emitter_authority {
        require!(
            emitter_sequence.emitter_type != EmitterType::Executable,
            CoreBridgeError::ExecutableEmitter
        );
        emitter_sequence.emitter_type = EmitterType::Legacy;
    } else {
        require!(
            emitter_sequence.emitter_type != EmitterType::Legacy,
            CoreBridgeError::LegacyEmitter
        );
        emitter_sequence.emitter_type = EmitterType::Executable;
    }

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

    // If the emitter is passed into the account context, revert if it does not agree with the one
    // encoded in the message account. This check is important for deriving the correct emitter
    // sequence PDA address, which is determined by the account passed into this context.
    if let Some(emitter) = ctx.accounts.emitter.as_ref() {
        require_keys_eq!(
            emitter.key(),
            info.emitter,
            CoreBridgeError::EmitterMismatch
        );
    }

    // Set other values to reflect published state.
    info.emitter_authority = Default::default();
    info.status = MessageStatus::Published;
    info.posted_timestamp = Clock::get().map(Into::into)?;
    info.sequence = emitter_sequence.value;

    // Update emitter sequence account with incremented value.
    {
        emitter_sequence.value += 1;

        let acc_data: &mut [_] = &mut ctx.accounts.emitter_sequence.data.borrow_mut();
        let mut writer = std::io::Cursor::new(acc_data);
        emitter_sequence.try_serialize(&mut writer)?;
    }

    let msg_acc_data: &mut [_] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    std::io::Write::write_all(&mut writer, PostedMessageV1::DISCRIMINATOR)?;
    info.serialize(&mut writer)?;

    // Done.
    Ok(())
}

/// If there is a fee, check the fee collector account to ensure that the fee has been paid.
pub(self) fn handle_message_fee<'info>(
    config: &Account<'info, LegacyAnchorized<Config>>,
    payer: &AccountInfo<'info>,
    fee_collector: &Option<AccountInfo<'info>>,
    system_program: &Program<'info, System>,
) -> Result<()> {
    if config.fee_lamports > 0 {
        let fee_collector = fee_collector
            .as_ref()
            .ok_or(error!(ErrorCode::AccountNotEnoughKeys).with_account_name("fee_collector"))?;

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

pub(self) fn create_or_realloc_emitter_sequence<'info>(
    emitter_sequence: &AccountInfo<'info>,
    payer: &AccountInfo<'info>,
    system_program: &Program<'info, System>,
    emitter: &Pubkey,
    emitter_sequence_bump: u8,
) -> Result<LegacyAnchorized<EmitterSequence>> {
    if emitter_sequence.data_is_empty() {
        // Create the emitter sequence account.
        utils::cpi::create_account_safe(
            CpiContext::new_with_signer(
                system_program.to_account_info(),
                utils::cpi::CreateAccountSafe {
                    payer: payer.to_account_info(),
                    new_account: emitter_sequence.to_account_info(),
                },
                &[&[
                    EmitterSequence::SEED_PREFIX,
                    emitter.as_ref(),
                    &[emitter_sequence_bump],
                ]],
            ),
            EmitterSequence::INIT_SPACE,
            &crate::ID,
        )?;

        Ok(EmitterSequence {
            legacy: LegacyEmitterSequence { value: 0 },
            bump: emitter_sequence_bump,
            emitter_type: EmitterType::Unset,
        }
        .into())
    } else if emitter_sequence.data_len() == LegacyEmitterSequence::INIT_SPACE {
        let legacy = LegacyAnchorized::<LegacyEmitterSequence>::try_deserialize(
            &mut emitter_sequence.data.borrow().as_ref(),
        )?;

        // This account is the legacy emitter sequence size. To migrate to the new schema, we must
        // transfer more lamports to the account and then realloc.
        let lamports_diff = Rent::get().map(|rent| {
            rent.minimum_balance(EmitterSequence::INIT_SPACE)
                .saturating_sub(emitter_sequence.lamports())
        })?;

        system_program::transfer(
            CpiContext::new(
                system_program.to_account_info(),
                system_program::Transfer {
                    from: payer.to_account_info(),
                    to: emitter_sequence.to_account_info(),
                },
            ),
            lamports_diff,
        )?;

        emitter_sequence.realloc(EmitterSequence::INIT_SPACE, false)?;

        // Because this account already existed, this account must have been created with the old
        // implementation. Program emitters were not possible with the old implementation, so we
        // will serialize the emitter type as legacy.
        Ok(EmitterSequence {
            legacy: legacy.0,
            bump: emitter_sequence_bump,
            emitter_type: EmitterType::Legacy,
        }
        .into())
    } else {
        // Nothing to do.
        LegacyAnchorized::<EmitterSequence>::try_deserialize(
            &mut emitter_sequence.data.borrow().as_ref(),
        )
    }
}

/// Determine the emitter seed for the emitter sequence account. This emitter will either come from
/// the message account if it was prepared beforehand or will be the signer passed into the account
/// context.
///
/// For posting a message, either a message has been prepared beforehand or this account is created
/// at this point in time. With respect to the emitter passed into the account context, it is only
/// a required signer if the message will be created in this instruction handler.
fn try_emitter_seed(emitter: &Option<Signer>, message: &AccountInfo) -> Result<Pubkey> {
    match emitter {
        Some(emitter) => Ok(emitter.key()),
        None => {
            // Message account must exist. We are making an assumption at this point whether this
            // account is actually a Core Bridge owned account with discriminator "msg\0". We will
            // verify the integrity of this message in the prepared message handler.
            require!(
                message.data_len() > 91,
                CoreBridgeError::InvalidPreparedMessage
            );

            // Return the emitter encoded in this account.
            message
                .try_borrow_data()
                .map(|data| PostedMessageV1::emitter_unsafe(&data))
                .map_err(Into::into)
        }
    }
}
