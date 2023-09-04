use std::io::Write;

use crate::{
    error::CoreBridgeError,
    state::{MessageStatus, PostedMessageV1Info},
    zero_copy::PostedMessageV1,
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct ProcessMessageV1<'info> {
    emitter_authority: Signer<'info>,

    /// CHECK: We do not deserialize this account as `PostedMessageV1` because allocating heap
    /// memory in its deserialization uses significant compute units with every call to this
    /// instruction handler. For large messages, this can be a significant cost.
    #[account(
        mut,
        owner = crate::ID
    )]
    draft_message: AccountInfo<'info>,

    /// CHECK: Destination for lamports if the draft message account is closed.
    #[account(mut)]
    close_account_destination: Option<UncheckedAccount<'info>>,
}

impl<'info> ProcessMessageV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let acc_data = ctx.accounts.draft_message.try_borrow_data()?;
        let message = PostedMessageV1::parse(&acc_data)?;

        // require!(
        //     info.status == MessageStatus::Writing,
        //     CoreBridgeError::MessageAlreadyPublished
        // );
        require_keys_eq!(
            ctx.accounts.emitter_authority.key(),
            message.emitter_authority(),
            CoreBridgeError::EmitterAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum ProcessMessageV1Directive {
    CloseMessageAccount,
    Write { index: u32, data: Vec<u8> },
    Finalize,
}

#[access_control(ProcessMessageV1::constraints(&ctx))]
pub fn process_message_v1(
    ctx: Context<ProcessMessageV1>,
    directive: ProcessMessageV1Directive,
) -> Result<()> {
    match directive {
        ProcessMessageV1Directive::CloseMessageAccount => close_message_account(ctx),
        ProcessMessageV1Directive::Write { index, data } => write(ctx, index, data),
        ProcessMessageV1Directive::Finalize => finalize(ctx),
    }
}

fn close_message_account(ctx: Context<ProcessMessageV1>) -> Result<()> {
    msg!("Directive: CloseMessageAccount");

    match &ctx.accounts.close_account_destination {
        Some(sol_destination) => {
            {
                let acc_data = ctx.accounts.draft_message.data.borrow();
                let message = PostedMessageV1::parse(&acc_data)?;

                require!(
                    message.status() != MessageStatus::Unset,
                    CoreBridgeError::MessageAlreadyPublished
                );
            }

            crate::utils::close_account(
                ctx.accounts.draft_message.to_account_info(),
                sol_destination.to_account_info(),
            )
        }
        None => err!(ErrorCode::AccountNotEnoughKeys),
    }
}

fn write(ctx: Context<ProcessMessageV1>, index: u32, data: Vec<u8>) -> Result<()> {
    msg!("Directive: Write");

    require!(
        !data.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let msg_length = {
        let acc_data = ctx.accounts.draft_message.data.borrow();
        let message = PostedMessageV1::parse(&acc_data)?;

        require!(
            message.status() == MessageStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        message.payload_size()
    };

    let index = usize::try_from(index).unwrap();
    let end = index.saturating_add(data.len());
    require_gte!(msg_length, end, CoreBridgeError::DataOverflow);

    const START: usize = 4 + PostedMessageV1::PAYLOAD_START;
    let acc_data = &mut ctx.accounts.draft_message.data.borrow_mut();
    acc_data[(START + index)..(START + end)].copy_from_slice(&data);

    // Done.
    Ok(())
}

fn finalize(ctx: Context<ProcessMessageV1>) -> Result<()> {
    msg!("Directive: Finalize");

    let (nonce, consistency_level, emitter) = {
        let acc_data = ctx.accounts.draft_message.data.borrow();
        let message = PostedMessageV1::parse(&acc_data)?;

        require!(
            message.status() == MessageStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        (
            message.nonce(),
            message.consistency_level(),
            message.emitter(),
        )
    };

    // Skip the discriminator.
    let acc_data: &mut [u8] = &mut ctx.accounts.draft_message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);

    // Serialize all info for simplicity.
    writer.write_all(&PostedMessageV1::DISCRIMINATOR)?;
    PostedMessageV1Info {
        consistency_level,
        emitter_authority: ctx.accounts.emitter_authority.key(),
        status: MessageStatus::Finalized,
        _gap_0: Default::default(),
        posted_timestamp: Default::default(),
        nonce,
        sequence: Default::default(),
        solana_chain_id: Default::default(),
        emitter,
    }
    .serialize(&mut writer)
    .map_err(Into::into)
}
