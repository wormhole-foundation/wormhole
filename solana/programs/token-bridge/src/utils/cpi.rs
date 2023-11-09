use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge;
use wormhole_io::Writeable;

pub fn publish_token_bridge_message<'info, W>(
    ctx: CpiContext<'_, '_, '_, 'info, core_bridge::PublishMessage<'info>>,
    nonce: u32,
    message: W,
) -> Result<()>
where
    W: Writeable,
{
    core_bridge::publish_message(
        ctx,
        core_bridge::PublishMessageDirective::Message {
            nonce,
            payload: message.to_vec(),
            commitment: core_bridge::Commitment::Finalized,
        },
    )
}
