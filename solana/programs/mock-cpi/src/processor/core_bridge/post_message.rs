use crate::{constants::MESSAGE_SEED_PREFIX, state::SignerSequence};
use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge;

const LEGACY_EMITTER_SEED_PREFIX: &[u8] = b"my_legacy_emitter";

#[derive(Accounts)]
pub struct MockPostMessage<'info> {
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

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages as
    /// its program ID.
    #[account(
        seeds = [core_bridge::PROGRAM_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_program_emitter: Option<AccountInfo<'info>>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages as a
    /// custom emitter address.
    #[account(
        seeds = [LEGACY_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_custom_emitter: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Core Bridge program.
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
pub struct MockPostMessageArgs {
    pub nonce: u32,
    pub payload: Vec<u8>,
}

pub fn mock_post_message(ctx: Context<MockPostMessage>, args: MockPostMessageArgs) -> Result<()> {
    let MockPostMessageArgs { nonce, payload } = args;

    let sequence = ctx.accounts.payer_sequence.take_and_uptick();

    match (
        &ctx.accounts.core_program_emitter,
        &ctx.accounts.core_custom_emitter,
    ) {
        (Some(authority), _) => core_bridge::publish_message(
            CpiContext::new_with_signer(
                ctx.accounts.core_bridge_program.to_account_info(),
                core_bridge::PublishMessage {
                    payer: ctx.accounts.payer.to_account_info(),
                    message: ctx.accounts.core_message.to_account_info(),
                    emitter_authority: authority.to_account_info(),
                    config: ctx.accounts.core_bridge_config.to_account_info(),
                    emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                    fee_collector: ctx.accounts.core_fee_collector.clone(),
                    system_program: ctx.accounts.system_program.to_account_info(),
                },
                &[
                    &[
                        core_bridge::PROGRAM_EMITTER_SEED_PREFIX,
                        &[ctx.bumps["core_program_emitter"]],
                    ],
                    &[
                        MESSAGE_SEED_PREFIX,
                        ctx.accounts.payer.key().as_ref(),
                        sequence.as_ref(),
                        &[ctx.bumps["core_message"]],
                    ],
                ],
            ),
            core_bridge::PublishMessageDirective::ProgramMessage {
                program_id: crate::ID,
                nonce,
                payload,
                commitment: core_bridge::Commitment::Finalized,
            },
        ),
        (None, Some(authority)) => core_bridge::publish_message(
            CpiContext::new_with_signer(
                ctx.accounts.core_bridge_program.to_account_info(),
                core_bridge::PublishMessage {
                    payer: ctx.accounts.payer.to_account_info(),
                    message: ctx.accounts.core_message.to_account_info(),
                    emitter_authority: authority.to_account_info(),
                    config: ctx.accounts.core_bridge_config.to_account_info(),
                    emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                    fee_collector: ctx.accounts.core_fee_collector.clone(),
                    system_program: ctx.accounts.system_program.to_account_info(),
                },
                &[
                    &[
                        LEGACY_EMITTER_SEED_PREFIX,
                        &[ctx.bumps["core_custom_emitter"]],
                    ],
                    &[
                        MESSAGE_SEED_PREFIX,
                        ctx.accounts.payer.key().as_ref(),
                        sequence.as_ref(),
                        &[ctx.bumps["core_message"]],
                    ],
                ],
            ),
            core_bridge::PublishMessageDirective::Message {
                nonce,
                payload,
                commitment: core_bridge::Commitment::Finalized,
            },
        ),
        (None, None) => err!(ErrorCode::AccountNotEnoughKeys),
    }
}
