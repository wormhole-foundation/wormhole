use crate::{error::CoreBridgeError, state::PostedMessageV1};
use anchor_lang::prelude::*;
use solana_program::program_memory::sol_memcpy;

#[derive(Accounts)]
pub struct WriteMessageV1<'info> {
    emitter_authority: Signer<'info>,

    /// CHECK: Message account. The payload will be written to and then finalized. This message can
    /// only be published when the message is finalized.
    #[account(
        mut,
        owner = crate::ID,
        constraint = PostedMessageV1::require_draft_message(&draft_message, &emitter_authority)?
    )]
    draft_message: AccountInfo<'info>,
}

impl<'info> WriteMessageV1<'info> {
    fn constraints(ctx: &Context<Self>, args: &WriteMessageV1Args) -> Result<()> {
        require!(
            !args.data.is_empty(),
            CoreBridgeError::InvalidInstructionArgument
        );

        let msg_length =
            PostedMessageV1::payload_size_unsafe(&ctx.accounts.draft_message.data.borrow());

        require!(
            args.index
                .saturating_add(args.data.len().try_into().unwrap())
                <= msg_length,
            CoreBridgeError::DataOverflow
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

#[access_control(WriteMessageV1::constraints(&ctx, &args))]
pub fn write_message_v1(ctx: Context<WriteMessageV1>, args: WriteMessageV1Args) -> Result<()> {
    let WriteMessageV1Args { index, data } = args;

    let acc_data = &mut ctx.accounts.draft_message.data.borrow_mut();
    sol_memcpy(
        &mut acc_data[(PostedMessageV1::PAYLOAD_START + usize::try_from(index).unwrap())..],
        &data,
        data.len(),
    );

    // Done.
    Ok(())
}
