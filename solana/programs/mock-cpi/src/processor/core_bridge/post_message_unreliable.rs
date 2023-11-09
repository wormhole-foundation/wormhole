use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge;

const LEGACY_EMITTER_SEED_PREFIX: &[u8] = b"my_unreliable_emitter";
const MESSAGE_SEED_PREFIX: &[u8] = b"my_unreliable_message";

#[derive(Accounts)]
#[instruction(nonce: u32)]
pub struct MockPostMessageUnreliable<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
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
    core_fee_collector: Option<AccountInfo<'info>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge::CoreBridge>,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct MockPostMessageUnreliableArgs {
    pub nonce: u32,
    pub payload: Vec<u8>,
}

pub fn mock_post_message_unreliable(
    ctx: Context<MockPostMessageUnreliable>,
    args: MockPostMessageUnreliableArgs,
) -> Result<()> {
    let MockPostMessageUnreliableArgs { nonce, payload } = args;

    core_bridge::post_message_unreliable(
        CpiContext::new_with_signer(
            ctx.accounts.core_bridge_program.to_account_info(),
            core_bridge::PostMessageUnreliable {
                payer: ctx.accounts.payer.to_account_info(),
                message: ctx.accounts.core_message.to_account_info(),
                emitter: ctx.accounts.core_emitter.to_account_info(),
                config: ctx.accounts.core_bridge_config.to_account_info(),
                emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                fee_collector: ctx.accounts.core_fee_collector.clone(),
                system_program: ctx.accounts.system_program.to_account_info(),
            },
            &[
                &[LEGACY_EMITTER_SEED_PREFIX, &[ctx.bumps["core_emitter"]]],
                &[
                    MESSAGE_SEED_PREFIX,
                    ctx.accounts.payer.key().as_ref(),
                    nonce.to_le_bytes().as_ref(),
                    &[ctx.bumps["core_message"]],
                ],
            ],
        ),
        core_bridge::PostMessageArgs {
            nonce,
            payload,
            commitment: core_bridge::Commitment::Finalized,
        },
    )
}
