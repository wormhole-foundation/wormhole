use anchor_lang::prelude::*;
use anchor_spl::token;
use token_bridge_program::sdk as token_bridge;

#[derive(Accounts)]
pub struct MockLegacyCompleteTransferWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        mut,
        token::mint = token_bridge_wrapped_mint,
        token::authority = recipient,
    )]
    recipient_token: Account<'info, token::TokenAccount>,

    /// CHECK: VAA recipient (i.e. recipient token owner).
    recipient: AccountInfo<'info>,

    #[account(
        mut,
        token::mint = token_bridge_wrapped_mint,
    )]
    payer_token: Account<'info, token::TokenAccount>,

    /// CHECK: This account is needed for the Token Bridge program.
    vaa: UncheckedAccount<'info>,

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
    token_bridge_program: Program<'info, token_bridge::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

pub fn mock_legacy_complete_transfer_wrapped(
    ctx: Context<MockLegacyCompleteTransferWrapped>,
) -> Result<()> {
    token_bridge::complete_transfer_wrapped(CpiContext::new(
        ctx.accounts.token_bridge_program.to_account_info(),
        token_bridge::CompleteTransferWrapped {
            payer: ctx.accounts.payer.to_account_info(),
            vaa: ctx.accounts.vaa.to_account_info(),
            claim: ctx.accounts.token_bridge_claim.to_account_info(),
            registered_emitter: ctx
                .accounts
                .token_bridge_registered_emitter
                .to_account_info(),
            recipient_token: ctx.accounts.recipient_token.to_account_info(),
            payer_token: ctx.accounts.payer_token.to_account_info(),
            wrapped_mint: ctx.accounts.token_bridge_wrapped_mint.to_account_info(),
            wrapped_asset: ctx.accounts.token_bridge_wrapped_asset.to_account_info(),
            mint_authority: ctx.accounts.token_bridge_mint_authority.to_account_info(),
            recipient: Some(ctx.accounts.recipient.to_account_info()),
            system_program: ctx.accounts.system_program.to_account_info(),
            token_program: ctx.accounts.token_program.to_account_info(),
        },
    ))
}
