use anchor_lang::prelude::*;
use anchor_spl::{associated_token, token};
use token_bridge_program::{self, TokenBridge};

#[derive(Accounts)]
pub struct MockLegacyCompleteTransferNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        token::mint = mint,
        token::authority = recipient,
    )]
    recipient_token: Account<'info, token::TokenAccount>,

    /// CHECK: VAA recipient (i.e. recipient token owner).
    recipient: AccountInfo<'info>,

    #[account(
        mut,
        token::mint = mint,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: This account is needed for the Token Bridge program.
    posted_vaa: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_claim: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_registered_emitter: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(mut)]
    token_bridge_custody_token: UncheckedAccount<'info>,

    mint: Account<'info, token::Mint>,

    /// CHECK: This account is needed for the Token Bridge program.
    token_bridge_custody_authority: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    core_bridge_program: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_bridge_program: Program<'info, TokenBridge>,
    token_program: Program<'info, token::Token>,
    associated_token_program: Program<'info, associated_token::AssociatedToken>,
}

pub fn mock_legacy_complete_transfer_native(
    ctx: Context<MockLegacyCompleteTransferNative>,
) -> Result<()> {
    token_bridge_program::legacy::cpi::complete_transfer_native(CpiContext::new(
        ctx.accounts.token_bridge_program.to_account_info(),
        token_bridge_program::legacy::cpi::CompleteTransferNative {
            payer: ctx.accounts.payer.to_account_info(),
            posted_vaa: ctx.accounts.posted_vaa.to_account_info(),
            claim: ctx.accounts.token_bridge_claim.to_account_info(),
            registered_emitter: ctx
                .accounts
                .token_bridge_registered_emitter
                .to_account_info(),
            recipient_token: ctx.accounts.recipient_token.to_account_info(),
            payer_token: ctx.accounts.payer_token.to_account_info(),
            custody_token: ctx.accounts.token_bridge_custody_token.to_account_info(),
            mint: ctx.accounts.mint.to_account_info(),
            custody_authority: ctx
                .accounts
                .token_bridge_custody_authority
                .to_account_info(),
            recipient: Some(ctx.accounts.recipient.to_account_info()),
            system_program: ctx.accounts.system_program.to_account_info(),
            core_bridge_program: ctx.accounts.core_bridge_program.to_account_info(),
            token_program: ctx.accounts.token_program.to_account_info(),
        },
    ))
}
