use crate::{constants::MESSAGE_SEED_PREFIX, state::SignerSequence};
use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge_sdk;

const LEGACY_EMITTER_SEED_PREFIX: &[u8] = b"my_legacy_emitter";

#[derive(Accounts)]
pub struct MockLegacyPostMessage<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        space = 8 + SignerSequence::INIT_SPACE,
        seeds = [SignerSequence::SEED_PREFIX, payer.key().as_ref()],
        bump,
    )]
    payer_sequence: Account<'info, SignerSequence>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(
        mut,
        seeds = [
            MESSAGE_SEED_PREFIX,
            payer.key().as_ref(),
            payer_sequence.to_le_bytes().as_ref()
        ],
        bump
    )]
    core_message: AccountInfo<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    #[account(
        seeds = [LEGACY_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<UncheckedAccount<'info>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::InvokeCoreBridge<'info> for MockLegacyPostMessage<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::InvokePostMessageV1<'info> for MockLegacyPostMessage<'info> {
    fn config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn emitter(&self) -> AccountInfo<'info> {
        self.core_emitter.to_account_info()
    }

    fn emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn fee_collector(&self) -> Option<AccountInfo<'info>> {
        self.core_fee_collector
            .as_ref()
            .map(|acc| acc.to_account_info())
    }

    fn message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct MockLegacyPostMessageArgs {
    pub nonce: u32,
    pub payload: Vec<u8>,
}

pub fn mock_legacy_post_message(
    ctx: Context<MockLegacyPostMessage>,
    args: MockLegacyPostMessageArgs,
) -> Result<()> {
    let MockLegacyPostMessageArgs { nonce, payload } = args;

    let sequence = ctx.accounts.payer_sequence.take_and_uptick();

    core_bridge_sdk::cpi::post_new_message_v1_bytes(
        ctx.accounts,
        core_bridge_sdk::cpi::PostMessageArgs {
            nonce,
            payload,
            commitment: core_bridge_sdk::types::Commitment::Finalized,
        },
        &[LEGACY_EMITTER_SEED_PREFIX, &[ctx.bumps["core_emitter"]]],
        Some(&[
            MESSAGE_SEED_PREFIX,
            ctx.accounts.payer.key().as_ref(),
            sequence.as_ref(),
            &[ctx.bumps["core_message"]],
        ]),
    )
}
