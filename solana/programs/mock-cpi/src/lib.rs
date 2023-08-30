#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;

declare_id!("MockCpi696969696969696969696969696969696969");

pub mod constants;

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

    pub fn mock_legacy_transfer_tokens_native(
        ctx: Context<MockLegacyTransferTokensNative>,
        args: MockLegacyTransferTokensArgs,
    ) -> Result<()> {
        processor::mock_legacy_transfer_tokens_native(ctx, args)
    }

    pub fn mock_legacy_transfer_tokens_wrapped(
        ctx: Context<MockLegacyTransferTokensWrapped>,
        args: MockLegacyTransferTokensArgs,
    ) -> Result<()> {
        processor::mock_legacy_transfer_tokens_wrapped(ctx, args)
    }

    pub fn mock_legacy_transfer_tokens_with_payload_native(
        ctx: Context<MockLegacyTransferTokensWithPayloadNative>,
        args: MockLegacyTransferTokensWithPayloadArgs,
    ) -> Result<()> {
        processor::mock_legacy_transfer_tokens_with_payload_native(ctx, args)
    }

    pub fn mock_legacy_transfer_tokens_with_payload_wrapped(
        ctx: Context<MockLegacyTransferTokensWithPayloadWrapped>,
        args: MockLegacyTransferTokensWithPayloadArgs,
    ) -> Result<()> {
        processor::mock_legacy_transfer_tokens_with_payload_wrapped(ctx, args)
    }

    pub fn mock_legacy_complete_transfer_native(
        ctx: Context<MockLegacyCompleteTransferNative>,
    ) -> Result<()> {
        processor::mock_legacy_complete_transfer_native(ctx)
    }

    pub fn mock_legacy_complete_transfer_wrapped(
        ctx: Context<MockLegacyCompleteTransferWrapped>,
    ) -> Result<()> {
        processor::mock_legacy_complete_transfer_wrapped(ctx)
    }

    pub fn mock_legacy_complete_transfer_with_payload_native(
        ctx: Context<MockLegacyCompleteTransferWithPayloadNative>,
    ) -> Result<()> {
        processor::mock_legacy_complete_transfer_with_payload_native(ctx)
    }

    pub fn mock_legacy_complete_transfer_with_payload_wrapped(
        ctx: Context<MockLegacyCompleteTransferWithPayloadWrapped>,
    ) -> Result<()> {
        processor::mock_legacy_complete_transfer_with_payload_wrapped(ctx)
    }

    pub fn withdraw_balance(ctx: Context<WithdrawBalance>) -> Result<()> {
        processor::withdraw_balance(ctx)
    }
}
