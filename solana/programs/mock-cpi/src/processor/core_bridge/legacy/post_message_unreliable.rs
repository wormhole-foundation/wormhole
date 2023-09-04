use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge_sdk;

const LEGACY_EMITTER_SEED_PREFIX: &[u8] = b"my_unreliable_emitter";
const MESSAGE_SEED_PREFIX: &[u8] = b"my_unreliable_message";

#[derive(Accounts)]
#[instruction(nonce: u32)]
pub struct MockLegacyPostMessageUnreliable<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(
        mut,
        seeds = [
            MESSAGE_SEED_PREFIX,
            payer.key().as_ref(),
            nonce.to_le_bytes().as_ref()
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

impl<'info> core_bridge_sdk::cpi::InvokeCoreBridge<'info>
    for MockLegacyPostMessageUnreliable<'info>
{
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::InvokePostMessageV1Unreliable<'info>
    for MockLegacyPostMessageUnreliable<'info>
{
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
pub struct MockLegacyPostMessageUnreliableArgs {
    pub nonce: u32,
    pub payload: Vec<u8>,
}

pub fn mock_legacy_post_message_unreliable(
    ctx: Context<MockLegacyPostMessageUnreliable>,
    args: MockLegacyPostMessageUnreliableArgs,
) -> Result<()> {
    let MockLegacyPostMessageUnreliableArgs { nonce, payload } = args;

    core_bridge_sdk::cpi::post_message_v1_unreliable_bytes(
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
            nonce.to_le_bytes().as_ref(),
            &[ctx.bumps["core_message"]],
        ]),
    )
}
