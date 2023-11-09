#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use wormhole_core_bridge_solana::sdk as core_bridge;

declare_id!("CoreBridgeHe11oWor1d11111111111111111111111");

#[derive(Accounts)]
pub struct PublishHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages as
    /// its program ID.
    #[account(
        seeds = [core_bridge::PROGRAM_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_program_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account will be created by the Core Bridge program using a generated keypair.
    #[account(mut)]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge::CoreBridge>,
}

#[program]
pub mod core_bridge_hello_world {
    use super::*;

    pub fn publish_hello_world(ctx: Context<PublishHelloWorld>) -> Result<()> {
        let nonce = 420;
        let payload = b"Hello, world!".to_vec();

        core_bridge::publish_message(
            CpiContext::new_with_signer(
                ctx.accounts.core_bridge_program.to_account_info(),
                core_bridge::PublishMessage {
                    payer: ctx.accounts.payer.to_account_info(),
                    message: ctx.accounts.core_message.to_account_info(),
                    emitter_authority: ctx.accounts.core_program_emitter.to_account_info(),
                    config: ctx.accounts.core_bridge_config.to_account_info(),
                    emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                    fee_collector: Some(ctx.accounts.core_fee_collector.to_account_info()),
                    system_program: ctx.accounts.system_program.to_account_info(),
                },
                &[&[
                    core_bridge::PROGRAM_EMITTER_SEED_PREFIX,
                    &[ctx.bumps["core_program_emitter"]],
                ]],
            ),
            core_bridge::PublishMessageDirective::ProgramMessage {
                program_id: crate::ID,
                nonce,
                payload,
                commitment: core_bridge::Commitment::Finalized,
            },
        )
    }
}
