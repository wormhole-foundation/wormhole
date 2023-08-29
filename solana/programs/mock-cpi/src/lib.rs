#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;

declare_id!("MockCP1696969696969696969696969696969696969");

mod processor;
pub(crate) use processor::*;

pub mod state;

#[program]
pub mod wormhole_mock_cpi_solana {
    use super::*;

    pub fn mock_legacy_post_message(
        ctx: Context<MockLegacyPostMessage>,
        args: MockLegacyPostMessageArgs,
    ) -> Result<()> {
        processor::mock_legacy_post_message(ctx, args)
    }

    pub fn mock_legacy_post_message_unreliable(
        ctx: Context<MockLegacyPostMessageUnreliable>,
        args: MockLegacyPostMessageUnreliableArgs,
    ) -> Result<()> {
        processor::mock_legacy_post_message_unreliable(ctx, args)
    }

    pub fn mock_legacy_transfer_tokens_with_payload_mint(
        ctx: Context<MockLegacyTransferTokensWithPayloadNative>,
        args: MockLegacyTransferTokensWithPayloadArgs,
    ) -> Result<()> {
        processor::mock_legacy_transfer_tokens_with_payload_mint(ctx, args)
    }
}
