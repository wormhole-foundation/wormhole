use anchor_lang::prelude::*;
use anchor_spl::token;

use crate::constants::CUSTOM_REDEEMER_SEED_PREFIX;

#[derive(Accounts)]
pub struct WithdrawBalance<'info> {
    /// CHECK: Custom redeemer authority used to redeem Token Bridge transfers.
    #[account(
        seeds = [CUSTOM_REDEEMER_SEED_PREFIX],
        bump,
    )]
    custom_redeemer_authority: AccountInfo<'info>,

    #[account(
        mut,
        token::authority = custom_redeemer_authority,
    )]
    program_token: Account<'info, token::TokenAccount>,

    #[account(mut)]
    dst_token: Account<'info, token::TokenAccount>,

    token_program: Program<'info, token::Token>,
}

pub fn withdraw_balance(ctx: Context<WithdrawBalance>) -> Result<()> {
    token::transfer(
        CpiContext::new_with_signer(
            ctx.accounts.token_program.to_account_info(),
            token::Transfer {
                from: ctx.accounts.program_token.to_account_info(),
                to: ctx.accounts.dst_token.to_account_info(),
                authority: ctx.accounts.custom_redeemer_authority.to_account_info(),
            },
            &[&[
                CUSTOM_REDEEMER_SEED_PREFIX,
                &[ctx.bumps["custom_redeemer_authority"]],
            ]],
        ),
        ctx.accounts.program_token.amount,
    )
}
