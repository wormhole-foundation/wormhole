use crate::{
    error::CoreBridgeError,
    legacy::instruction::LegacyPostMessageArgs,
    state::{
        BridgeProgramData, EmitterSequence, FeeCollector, MessageStatus, PostedMessageV1,
        PostedMessageV1Data, PostedMessageV1Info,
    },
};
use anchor_lang::prelude::*;
use wormhole_solana_common::{NewAccountSize, SeedPrefix};

#[derive(Accounts)]
#[instruction(args: LegacyPostMessageArgs)]
pub struct PostMessage<'info> {
    /// Bridge program data. This account is needed to determine whether the core bridge fee has
    /// been paid.
    #[account(
        mut,
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
    )]
    bridge: Account<'info, BridgeProgramData>,

    /// Posted message account data.
    ///
    /// NOTE: This is `init_if_needed` not because the original implementation allowed it (before if
    /// this message account were created again, this instruction handler would be a no-op). This
    /// account macro argument exists because we leverage publishing a Wormhole message using the
    /// same instruction handler, but for larger messages prepared with `init_message_v1` and
    /// `process_message_v1` instruction handlers.
    ///
    /// The unfortunate side effect of this is `init_if_needed` requires a message signer whether or
    /// not the account was already created. Is this a bug? Not the end of the world because the
    /// message signer used to create the message via `init_message_v1` should still have this
    /// signer by the time he wishes to post this message.
    #[account(
        init_if_needed,
        payer = payer,
        space = compute_size_if_needed(message, &args.payload)
    )]
    message: Account<'info, PostedMessageV1>,

    /// The emitter of the core bridge message. This account is typically an integrating program's
    /// PDA which signs for this instruction.
    emitter_authority: Signer<'info>,

    /// Sequence tracker for given emitter. Every core bridge message is tagged with a unique
    /// sequence number.
    ///
    /// NOTE: Because the emitter can either be the emitter authority in this account context or
    /// an address derived from an integrator's program, we use a custom function to help determine
    /// which seeds to use.
    #[account(
        init_if_needed,
        payer = payer,
        space = EmitterSequence::INIT_SPACE,
        seeds = [
            EmitterSequence::seed_prefix(),
            find_emitter_for_sequence(&emitter_authority, &message).as_ref()
        ],
        bump
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
        seeds = [FeeCollector::seed_prefix()],
        bump,
    )]
    fee_collector: Option<Account<'info, FeeCollector>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
}

pub fn post_message(ctx: Context<PostMessage>, args: LegacyPostMessageArgs) -> Result<()> {
    match ctx.accounts.message.data.status {
        MessageStatus::Unset => {
            // If the message status is unset, we need to make sure that this
            // message account has not been used already. The emitter will be
            // unset as well if the account was just created with
            // `init_if_needed`.
            require_keys_eq!(ctx.accounts.message.emitter, Default::default());

            handle_post_new_message(
                &mut ctx.accounts.bridge,
                &mut ctx.accounts.message,
                &ctx.accounts.emitter_authority,
                &mut ctx.accounts.emitter_sequence,
                &ctx.accounts.fee_collector,
                args,
            )
        }
        MessageStatus::Writing => {
            msg!("MessageStatus: Writing");
            handle_post_prepared_message(
                &mut ctx.accounts.bridge,
                &mut ctx.accounts.message,
                &ctx.accounts.emitter_authority,
                &mut ctx.accounts.emitter_sequence,
                &ctx.accounts.fee_collector,
                args,
            )
        }
    }
}

pub(in crate::legacy) fn handle_post_new_message(
    bridge: &mut Account<BridgeProgramData>,
    msg: &mut PostedMessageV1Data,
    emitter: &Signer,
    emitter_sequence: &mut Account<EmitterSequence>,
    fee_collector: &Option<Account<FeeCollector>>,
    args: LegacyPostMessageArgs,
) -> Result<()> {
    let LegacyPostMessageArgs {
        nonce,
        payload,
        commitment,
    } = args;

    // Should we require the payload not be empty?
    require!(
        !payload.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    // Determine whether fee has been paid. Update bridge program data account if so.
    //
    // NOTE: This is inconsistent with other Core Bridge implementations, where we would check that
    // the change would equal exactly the fee amount.
    handle_message_fee(bridge, fee_collector)?;

    // Sequence number will be used later on.
    let sequence = emitter_sequence.value;

    // NOTE: The legacy instruction had the note "DO NOT REMOVE - CRITICAL OUTPUT". But we may be
    // able to remove this to save on compute units.
    msg!("Sequence: {}", sequence);

    // Finally set the `message` account with posted data.
    *msg = PostedMessageV1Data {
        info: PostedMessageV1Info {
            consistency_level: commitment.into(),
            emitter_authority: Default::default(),
            status: MessageStatus::Unset,
            _gap_0: Default::default(),
            posted_timestamp: Clock::get().map(Into::into)?,
            nonce,
            sequence,
            solana_chain_id: Default::default(),
            emitter: emitter.key(),
        },
        payload,
    };

    // Increment emitter sequence value.
    emitter_sequence.value += 1;

    // Done.
    Ok(())
}

fn handle_post_prepared_message(
    bridge: &mut Account<BridgeProgramData>,
    msg: &mut PostedMessageV1Data,
    emitter_authority: &Signer,
    emitter_sequence: &mut Account<EmitterSequence>,
    fee_collector: &Option<Account<FeeCollector>>,
    args: LegacyPostMessageArgs,
) -> Result<()> {
    let LegacyPostMessageArgs {
        nonce,
        payload,
        commitment,
    } = args;

    // The payload argument is not allowed if the message has been prepared beforehand.
    require!(
        payload.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    // The emitter authority passed into the instruction handler must be the same one that drafted
    // this Core Bridge message.
    require_keys_eq!(msg.emitter_authority, emitter_authority.key());

    // Determine whether fee has been paid. Update bridge program data account if so.
    //
    // NOTE: This is inconsistent with other Core Bridge implementations, where we would check that
    // the change would equal exactly the fee amount.
    handle_message_fee(bridge, fee_collector)?;

    // Now indicate that this message will be observed by the guardians.
    msg.status = MessageStatus::Unset;
    msg.consistency_level = commitment.into();
    msg.emitter_authority = Default::default();
    msg.posted_timestamp = Clock::get().map(Into::into)?;
    msg.nonce = nonce;
    msg.sequence = emitter_sequence.value;

    // Increment emitter sequence value.
    emitter_sequence.value += 1;

    // Done.
    Ok(())
}

fn handle_message_fee(
    bridge: &mut Account<BridgeProgramData>,
    fee_collector: &Option<Account<FeeCollector>>,
) -> Result<()> {
    if bridge.fee_lamports == 0 {
        // Nothing to do.
        Ok(())
    } else {
        match fee_collector {
            Some(fee_collector) => {
                let collector_lamports = fee_collector.to_account_info().lamports();
                require!(
                    collector_lamports >= bridge.last_lamports.saturating_add(bridge.fee_lamports),
                    CoreBridgeError::InsufficientMessageFee
                );

                // Update bridge program data to reflect paid fees.
                bridge.last_lamports = collector_lamports;

                // Done.
                Ok(())
            }
            None => err!(ErrorCode::AccountNotEnoughKeys),
        }
    }
}

fn compute_size_if_needed(message_acct_info: &AccountInfo<'_>, payload: &Vec<u8>) -> usize {
    if message_acct_info.data_is_empty() {
        PostedMessageV1::compute_size(payload.len())
    } else {
        message_acct_info.data_len()
    }
}

/// For posting a message, either a message has been prepared beforehand or this account is
/// created at this point in time. We make the assumption that if the status is unset, it is
/// a message account created at this point, which is the way the legacy post message
/// instruction handler worked.
///
/// The legacy post message instruction handler did not allow posting a message as a program,
/// which `init_message_v1` allows for. So the emitter sequence account, whose PDA address is
/// derived using the emitter, is assigned to the emitter signer (now called the emitter
/// authority). Whereas with the new prepared message, this emitter can be taken from the
/// message account to re-derive the emitter sequence PDA address.
fn find_emitter_for_sequence(
    emitter_authority: &Signer<'_>,
    message: &Account<'_, PostedMessageV1>,
) -> Pubkey {
    match message.data.status {
        MessageStatus::Unset => emitter_authority.key(),
        MessageStatus::Writing => message.emitter,
    }
}
