use crate::{
    error::CoreBridgeError,
    state::MessageStatus,
    zero_copy::{LoadZeroCopy, PostedMessageV1},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct CloseMessageV1<'info> {
    emitter_authority: Signer<'info>,

    /// CHECK: Message account. The payload will be written to and then finalized. This message can
    /// only be published when the message is finalized.
    #[account(mut)]
    draft_message: AccountInfo<'info>,

    /// CHECK: Destination for lamports if the draft message account is closed.
    #[account(mut)]
    close_account_destination: AccountInfo<'info>,
}

impl<'info> CloseMessageV1<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let message = PostedMessageV1::load(&ctx.accounts.draft_message)?;

        require!(
            message.status() != MessageStatus::Unset,
            CoreBridgeError::MessageAlreadyPublished
        );

        require_keys_eq!(
            ctx.accounts.emitter_authority.key(),
            message.emitter_authority(),
            CoreBridgeError::EmitterAuthorityMismatch
        );

        // Done.
        Ok(())
    }
}
#[access_control(CloseMessageV1::constraints(&ctx))]
pub fn close_message_v1(ctx: Context<CloseMessageV1>) -> Result<()> {
    crate::utils::close_account(
        &ctx.accounts.draft_message,
        &ctx.accounts.close_account_destination,
    )
}
