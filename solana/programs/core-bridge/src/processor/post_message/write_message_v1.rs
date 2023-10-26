use crate::{
    error::CoreBridgeError,
    state::MessageStatus,
    zero_copy::{LoadZeroCopy, PostedMessageV1},
};
use anchor_lang::prelude::*;
use solana_program::program_memory::sol_memcpy;

#[derive(Accounts)]
pub struct WriteMessageV1<'info> {
    emitter_authority: Signer<'info>,

    /// CHECK: Message account. The payload will be written to and then finalized. This message can
    /// only be published when the message is finalized.
    #[account(mut)]
    draft_message: AccountInfo<'info>,
}

impl<'info> WriteMessageV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let message = PostedMessageV1::load(&ctx.accounts.draft_message)?;

        require_keys_eq!(
            ctx.accounts.emitter_authority.key(),
            message.emitter_authority(),
            CoreBridgeError::EmitterAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}

/// Arguments for the [write_message_v1](crate::wormhole_core_bridge_solana::write_message_v1)
/// instruction.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct WriteMessageV1Args {
    /// Index of message buffer.
    pub index: u32,
    /// Data representing subset of message buffer starting at specified index.
    pub data: Vec<u8>,
}

#[access_control(WriteMessageV1::constraints(&ctx))]
pub fn write_message_v1(ctx: Context<WriteMessageV1>, args: WriteMessageV1Args) -> Result<()> {
    let WriteMessageV1Args { index, data } = args;

    require!(
        !data.is_empty(),
        CoreBridgeError::InvalidInstructionArgument
    );

    let msg_length = {
        let message = PostedMessageV1::load(&ctx.accounts.draft_message).unwrap();

        require!(
            message.status() == MessageStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        message.payload_size()
    };

    let index = usize::try_from(index).unwrap();
    require!(
        index.saturating_add(data.len()) <= msg_length,
        CoreBridgeError::DataOverflow
    );

    let acc_data = &mut ctx.accounts.draft_message.data.borrow_mut();
    sol_memcpy(
        &mut acc_data[(PostedMessageV1::PAYLOAD_START + index)..],
        &data,
        data.len(),
    );

    // Done.
    Ok(())
}
