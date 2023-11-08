use crate::{
    error::CoreBridgeError,
    state::{MessageStatus, PostedMessageV1Info},
    zero_copy::{LoadZeroCopy, PostedMessageV1},
};
use anchor_lang::prelude::*;

#[derive(Accounts)]
pub struct FinalizeMessageV1<'info> {
    emitter_authority: Signer<'info>,

    /// CHECK: Message account. The payload will be written to and then finalized. This message can
    /// only be published when the message is finalized.
    #[account(mut)]
    draft_message: AccountInfo<'info>,
}

impl<'info> FinalizeMessageV1<'info> {
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

#[access_control(FinalizeMessageV1::constraints(&ctx))]
pub fn finalize_message_v1(ctx: Context<FinalizeMessageV1>) -> Result<()> {
    let (nonce, consistency_level, emitter) = {
        let message = PostedMessageV1::load(&ctx.accounts.draft_message).unwrap();

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

    let acc_data: &mut [_] = &mut ctx.accounts.draft_message.data.borrow_mut();
    let mut writer = std::io::Cursor::new(acc_data);

    // Serialize all info for simplicity.
    (
        PostedMessageV1::DISC,
        PostedMessageV1Info {
            consistency_level,
            emitter_authority: ctx.accounts.emitter_authority.key(),
            status: MessageStatus::ReadyForPublishing,
            _gap_0: Default::default(),
            posted_timestamp: Default::default(),
            nonce,
            sequence: Default::default(),
            solana_chain_id: Default::default(),
            emitter,
        },
    )
        .serialize(&mut writer)
        .map_err(Into::into)
}
