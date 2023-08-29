use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::state::EmitterSequence;
use token_bridge_program::{self, TokenBridge};

use super::MockLegacyTransferTokensWithPayloadArgs;

const MESSAGE_SEED_PREFIX: &[u8] = b"my_transfer";

#[derive(Accounts)]
pub struct MockLegacyTransferTokensWithPayloadNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        token::mint = mint,
        token::authority = payer
    )]
    src_token: Account<'info, token::TokenAccount>,

    /// CHECK: Mint.
    #[account(owner = token::Token::id())]
    mint: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    custody_token: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    transfer_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    custody_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        mut,
        seeds = [MESSAGE_SEED_PREFIX, core_emitter_sequence.to_be_bytes().as_ref()],
        bump,
    )]
    core_message: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_emitter_sequence: Account<'info, EmitterSequence>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_fee_collector: Option<UncheckedAccount<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        seeds = [b"sender"],
        bump,
    )]
    sender_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, TokenBridge>,
    token_program: Program<'info, token::Token>,
}

pub fn mock_legacy_transfer_tokens_with_payload_mint(
    ctx: Context<MockLegacyTransferTokensWithPayloadNative>,
    args: MockLegacyTransferTokensWithPayloadArgs,
) -> Result<()> {
    let MockLegacyTransferTokensWithPayloadArgs {
        nonce,
        amount,
        redeemer,
        redeemer_chain,
        payload,
    } = args;

    token_bridge_program::legacy::cpi::transfer_tokens_with_payload_native(
        CpiContext::new_with_signer(
            ctx.accounts.token_bridge_program.to_account_info(),
            token_bridge_program::legacy::cpi::TransferTokensWithPayloadNative {
                payer: ctx.accounts.payer.to_account_info(),
                src_token: ctx.accounts.src_token.to_account_info(),
                mint: ctx.accounts.mint.to_account_info(),
                custody_token: ctx.accounts.custody_token.to_account_info(),
                transfer_authority: ctx.accounts.transfer_authority.to_account_info(),
                custody_authority: ctx.accounts.custody_authority.to_account_info(),
                core_bridge_config: ctx.accounts.core_bridge_config.to_account_info(),
                core_message: ctx.accounts.core_message.to_account_info(),
                core_emitter: ctx.accounts.core_emitter.to_account_info(),
                core_emitter_sequence: ctx.accounts.core_emitter_sequence.to_account_info(),
                core_fee_collector: ctx
                    .accounts
                    .core_fee_collector
                    .as_ref()
                    .map(|acc| acc.to_account_info()),
                sender_authority: ctx.accounts.sender_authority.to_account_info(),
                system_program: ctx.accounts.system_program.to_account_info(),
                core_bridge_program: ctx.accounts.core_bridge_program.to_account_info(),
                token_program: ctx.accounts.token_program.to_account_info(),
            },
            &[
                &[
                    token_bridge_program::constants::EMITTER_SEED_PREFIX,
                    &[ctx.bumps["core_emitter"]],
                ],
                &[MESSAGE_SEED_PREFIX, &[ctx.bumps["core_message"]]],
            ],
        ),
        token_bridge_program::legacy::cpi::TransferTokensWithPayloadArgs {
            nonce,
            amount,
            redeemer,
            redeemer_chain,
            payload,
            cpi_program_id: Some(crate::ID),
        },
    )
}
