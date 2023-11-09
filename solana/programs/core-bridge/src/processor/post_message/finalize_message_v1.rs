use crate::state::{MessageStatus, PostedMessageV1};
use anchor_lang::prelude::*;
use solana_program::program_memory::sol_memset;

#[derive(Accounts)]
pub struct FinalizeMessageV1<'info> {
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

pub fn finalize_message_v1(ctx: Context<FinalizeMessageV1>) -> Result<()> {
    let acc_data = &mut ctx.accounts.draft_message.data.borrow_mut();
    sol_memset(
        &mut acc_data[37..],
        MessageStatus::ReadyForPublishing as u8,
        1,
    );

    // Done.
    Ok(())
}
