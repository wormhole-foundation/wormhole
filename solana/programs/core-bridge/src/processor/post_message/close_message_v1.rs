use crate::state::PostedMessageV1;
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct CloseMessageV1<'info> {
    emitter_authority: Signer<'info>,

    /// CHECK: Message account. The payload will be written to and then finalized. This message can
    /// only be published when the message is finalized.
    #[account(
        mut,
        owner = crate::ID,
        constraint = PostedMessageV1::require_draft_message(&draft_message, &emitter_authority)?
    )]
    draft_message: AccountInfo<'info>,

    /// CHECK: Destination for lamports if the draft message account is closed.
    #[account(mut)]
    close_account_destination: AccountInfo<'info>,
}

pub fn close_message_v1(ctx: Context<CloseMessageV1>) -> Result<()> {
    crate::utils::close_account(
        &ctx.accounts.draft_message,
        &ctx.accounts.close_account_destination,
    )
}
