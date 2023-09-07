#![allow(clippy::result_large_err)]

use anchor_lang::prelude::*;
use wormhole_core_bridge_solana::sdk as core_bridge_sdk;

declare_id!("CoreBridgeHe11oWor1d11111111111111111111111");

#[derive(Accounts)]
pub struct PublishHelloWorld<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages as
    /// its program ID.
    #[account(
        seeds = [core_bridge_sdk::PROGRAM_EMITTER_SEED_PREFIX],
        bump,
    )]
    core_program_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
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
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for PublishHelloWorld<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for PublishHelloWorld<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.core_program_emitter.to_account_info()
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        Some(self.core_fee_collector.to_account_info())
    }
}

#[program]
pub mod core_bridge_hello_world {
    use super::*;

    pub fn publish_hello_world(ctx: Context<PublishHelloWorld>) -> Result<()> {
        let nonce = 420;
        let payload = b"Hello, world!".to_vec();

        core_bridge_sdk::cpi::publish_message(
            ctx.accounts,
            &ctx.accounts.core_message,
            core_bridge_sdk::cpi::PublishMessageDirective::ProgramMessage {
                program_id: crate::ID,
                nonce,
                payload,
                commitment: core_bridge_sdk::types::Commitment::Finalized,
            },
            Some(&[&[
                core_bridge_sdk::PROGRAM_EMITTER_SEED_PREFIX,
                &[ctx.bumps["core_program_emitter"]],
            ]]),
        )
    }
}
