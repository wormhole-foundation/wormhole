use anchor_lang::{prelude::*, system_program};
use core_bridge_program::{state::Config, CoreBridge, SeedPrefix};

const LEGACY_EMITTER_SEED_PREFIX: &[u8] = b"my_unreliable_emitter";
const MESSAGE_SEED_PREFIX: &[u8] = b"my_unreliable_message";

#[derive(Accounts)]
pub struct MockLegacyPostMessageUnreliable<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// We need to deserialize this account to determine the Wormhole message fee.
    #[account(
        mut,
        seeds = [Config::SEED_PREFIX],
        bump,
        seeds::program = core_bridge_program
    )]
    core_bridge_config: Account<'info, Config>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(
        mut,
        seeds = [MESSAGE_SEED_PREFIX],
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
    core_bridge_program: Program<'info, CoreBridge>,
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
    let fee_collector = match (
        ctx.accounts.core_bridge_config.fee_lamports,
        ctx.accounts.core_fee_collector.as_ref(),
    ) {
        (0, _) => None,
        (lamports, Some(core_fee_collector)) => {
            system_program::transfer(
                CpiContext::new(
                    ctx.accounts.system_program.to_account_info(),
                    system_program::Transfer {
                        from: ctx.accounts.payer.to_account_info(),
                        to: core_fee_collector.to_account_info(),
                    },
                ),
                lamports,
            )?;

            Some(core_fee_collector.to_account_info())
        }
        _ => return err!(ErrorCode::AccountNotEnoughKeys),
    };

    let MockLegacyPostMessageUnreliableArgs { nonce, payload } = args;

    core_bridge_program::legacy::cpi::post_message_unreliable(
        CpiContext::new_with_signer(
            ctx.accounts.core_bridge_program.to_account_info(),
            core_bridge_program::legacy::cpi::PostMessageUnreliable {
                config: ctx.accounts.core_bridge_config.to_account_info(),
                message: ctx.accounts.core_message.to_account_info(),
                emitter: ctx.accounts.core_emitter.to_account_info(),
                emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
                fee_collector,
                system_program: ctx.accounts.system_program.to_account_info(),
            },
            &[
                &[LEGACY_EMITTER_SEED_PREFIX, &[ctx.bumps["core_emitter"]]],
                &[MESSAGE_SEED_PREFIX, &[ctx.bumps["core_message"]]],
            ],
        ),
        core_bridge_program::legacy::cpi::PostMessageUnreliableArgs {
            nonce,
            payload,
            commitment: core_bridge_program::types::Commitment::Finalized,
        },
    )
}
