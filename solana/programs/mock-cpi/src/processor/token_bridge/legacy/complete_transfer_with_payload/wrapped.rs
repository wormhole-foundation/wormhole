use anchor_lang::prelude::*;
use anchor_spl::token;
use token_bridge_program::{self, constants::PROGRAM_REDEEMER_SEED_PREFIX, TokenBridge};

use crate::constants::CUSTOM_REDEEMER_SEED_PREFIX;

#[derive(Accounts)]
pub struct MockLegacyCompleteTransferWithPayloadWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        seeds = [PROGRAM_REDEEMER_SEED_PREFIX],
        bump,
    )]
    token_bridge_program_redeemer_authority: Option<AccountInfo<'info>>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        seeds = [CUSTOM_REDEEMER_SEED_PREFIX],
        bump,
    )]
    token_bridge_custom_redeemer_authority: Option<AccountInfo<'info>>,

    #[account(
        mut,
        token::mint = token_bridge_wrapped_mint,
    )]
    dst_token: Account<'info, token::TokenAccount>,

    /// CHECK: This account is needed for the Token Bridge program.
    posted_vaa: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_claim: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_registered_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_wrapped_mint: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_wrapped_asset: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_mint_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, TokenBridge>,
    token_program: Program<'info, token::Token>,
}

pub fn mock_legacy_complete_transfer_with_payload_wrapped(
    ctx: Context<MockLegacyCompleteTransferWithPayloadWrapped>,
) -> Result<()> {
    let (redeemer_authority, redeemer_seed_prefix, redeemer_bump) = match (
        &ctx.accounts.token_bridge_program_redeemer_authority,
        &ctx.accounts.token_bridge_custom_redeemer_authority,
    ) {
        (Some(sender_authority), _) => (
            sender_authority.to_account_info(),
            PROGRAM_REDEEMER_SEED_PREFIX,
            ctx.bumps["token_bridge_program_redeemer_authority"],
        ),
        (None, Some(sender_authority)) => (
            sender_authority.to_account_info(),
            CUSTOM_REDEEMER_SEED_PREFIX,
            ctx.bumps["token_bridge_custom_redeemer_authority"],
        ),
        (None, None) => return err!(ErrorCode::AccountNotEnoughKeys),
    };

    token_bridge_program::legacy::cpi::complete_transfer_with_payload_wrapped(
        CpiContext::new_with_signer(
            ctx.accounts.token_bridge_program.to_account_info(),
            token_bridge_program::legacy::cpi::CompleteTransferWithPayloadWrapped {
                payer: ctx.accounts.payer.to_account_info(),
                posted_vaa: ctx.accounts.posted_vaa.to_account_info(),
                claim: ctx.accounts.token_bridge_claim.to_account_info(),
                registered_emitter: ctx
                    .accounts
                    .token_bridge_registered_emitter
                    .to_account_info(),
                dst_token: ctx.accounts.dst_token.to_account_info(),
                redeemer_authority,
                wrapped_mint: ctx.accounts.token_bridge_wrapped_mint.to_account_info(),
                wrapped_asset: ctx.accounts.token_bridge_wrapped_asset.to_account_info(),
                mint_authority: ctx.accounts.token_bridge_mint_authority.to_account_info(),
                system_program: ctx.accounts.system_program.to_account_info(),
                core_bridge_program: ctx.accounts.core_bridge_program.to_account_info(),
                token_program: ctx.accounts.token_program.to_account_info(),
            },
            &[&[redeemer_seed_prefix, &[redeemer_bump]]],
        ),
    )
}
