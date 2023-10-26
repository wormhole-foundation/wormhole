use crate::{constants::MESSAGE_SEED_PREFIX, state::SignerSequence};
use anchor_lang::prelude::*;
use core_bridge_program::{constants::PROGRAM_EMITTER_SEED_PREFIX, sdk as core_bridge_sdk};

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
        seeds = [PROGRAM_EMITTER_SEED_PREFIX],
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
    core_fee_collector: Option<UncheckedAccount<'info>>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info> for MockPostMessage<'info> {
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for MockPostMessage<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        match &self.core_program_emitter {
            Some(program_emitter) => program_emitter.to_account_info(),
            None => self.core_custom_emitter.as_ref().unwrap().to_account_info(),
        }
    }

    fn core_emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn core_fee_collector(&self) -> Option<AccountInfo<'info>> {
        self.core_fee_collector
            .as_ref()
            .map(|acc| acc.to_account_info())
    }
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
        (Some(_), _) => core_bridge_sdk::cpi::publish_message(
            ctx.accounts,
            &ctx.accounts.core_message,
            core_bridge_sdk::cpi::PublishMessageDirective::ProgramMessage {
                program_id: crate::ID,
                nonce,
                payload,
                commitment: core_bridge_sdk::types::Commitment::Finalized,
            },
            Some(&[
                &[
                    PROGRAM_EMITTER_SEED_PREFIX,
                    &[ctx.bumps["core_program_emitter"]],
                ],
                &[
                    MESSAGE_SEED_PREFIX,
                    ctx.accounts.payer.key().as_ref(),
                    sequence.as_ref(),
                    &[ctx.bumps["core_message"]],
                ],
            ]),
        ),
        (None, Some(_)) => core_bridge_sdk::cpi::publish_message(
            ctx.accounts,
            &ctx.accounts.core_message,
            core_bridge_sdk::cpi::PublishMessageDirective::Message {
                nonce,
                payload,
                commitment: core_bridge_sdk::types::Commitment::Finalized,
            },
            Some(&[
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
            ]),
        ),
        (None, None) => err!(ErrorCode::AccountNotEnoughKeys),
    }
}
