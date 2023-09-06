use crate::{
    error::CoreBridgeError,
    legacy::{instruction::PostMessageArgs, utils::LegacyAnchorized},
    state::{
        Config, EmitterSequence, MessageStatus, PostedMessageV1, PostedMessageV1Data,
        PostedMessageV1Info,
    },
};
use anchor_lang::prelude::*;
#[derive(Accounts)]
#[instruction(args: PostMessageArgs)]
pub struct PostMessage<'info> {
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
    #[account(mut)]
    message: AccountInfo<'info>,

    /// CHECK: The emitter of the core bridge message. This account is typically an integrating
    /// program's PDA which signs for this instruction. This account must be a signer if the message
    /// is created in this instruction.
    emitter: Option<AccountInfo<'info>>,

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
            EmitterSequence::SEED_PREFIX,
            find_emitter_for_sequence(&emitter, &message)?.as_ref()
        ],
        bump
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
    for PostMessage<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyPostMessage";

    const ANCHOR_IX_FN: fn(Context<Self>, PostMessageArgs) -> Result<()> = post_message;
}

fn post_message(ctx: Context<PostMessage>, args: PostMessageArgs) -> Result<()> {
    if ctx.accounts.message.data_is_empty() {
        handle_post_new_message(ctx, args)
    } else {
        handle_post_prepared_message(ctx, args)
    }
}

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
        let data_len = PostedMessageV1::compute_size(payload.len());
        anchor_lang::system_program::create_account(
            CpiContext::new(
                ctx.accounts.system_program.to_account_info(),
                anchor_lang::system_program::CreateAccount {
                    from: ctx.accounts.payer.to_account_info(),
                    to: ctx.accounts.message.to_account_info(),
                },
            ),
            Rent::get().map(|rent| rent.minimum_balance(data_len))?,
            data_len.try_into().unwrap(),
            &crate::ID,
        )?;
    }

    let data = crate::legacy::new_posted_message_data(
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

    let msg_acc_data: &mut [u8] = &mut ctx.accounts.message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(msg_acc_data);

    // Finally set the `message` account with posted data.
    LegacyAnchorized::from(PostedMessageV1 { data }).try_serialize(&mut writer)?;

    // Done.
    Ok(())
}

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
    LegacyAnchorized::from(PostedMessageV1 { data }).try_serialize(&mut writer)?;

    // Done.
    Ok(())
}

pub(in crate::legacy) fn new_posted_message_data(
    config: &mut Account<LegacyAnchorized<0, Config>>,
    fee_collector: &Option<AccountInfo>,
    emitter_sequence: &mut Account<LegacyAnchorized<0, EmitterSequence>>,
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
    let sequence = emitter_sequence.value;

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

    // Increment emitter sequence value.
    emitter_sequence.value += 1;

    // Done.
    Ok(data)
}

fn handle_message_fee(
    config: &mut Account<LegacyAnchorized<0, Config>>,
    fee_collector: &Option<AccountInfo>,
) -> Result<()> {
    match (config.fee_lamports, fee_collector) {
        (0, _) => Ok(()), // Nothing to do.
        (lamports, Some(fee_collector)) => {
            let collector_lamports = fee_collector.to_account_info().lamports();
            require_eq!(
                collector_lamports,
                config.last_lamports.saturating_add(lamports),
                CoreBridgeError::InsufficientFees
            );

            // Update core bridge config to reflect paid fees.
            config.last_lamports = collector_lamports;

            // Done.
            Ok(())
        }
        _ => err!(ErrorCode::AccountNotEnoughKeys),
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
