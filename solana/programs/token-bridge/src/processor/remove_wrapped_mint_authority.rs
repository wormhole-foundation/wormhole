use crate::{constants::MINT_AUTHORITY_SEED_PREFIX, error::TokenBridgeError, state::WrappedAsset};
use anchor_lang::prelude::*;
use anchor_spl::token;

#[derive(Accounts)]
pub struct RemoveWrappedMintAuthority<'info> {
    /// Native mint. We ensure this mint is not one that has originated from a foreign
    /// network in access control.
    mint: Account<'info, token::Mint>,

    /// Non-existent wrapped asset account.
    ///
    /// CHECK: This account should have zero data. Otherwise this mint was created by the Token
    /// Bridge program.
    #[account(
        seeds = [
            WrappedAsset::SEED_PREFIX,
            mint.key().as_ref()
        ],
        bump,
        constraint = non_existent_wrapped_asset.data_is_empty() @ TokenBridgeError::WrappedAsset
    )]
    non_existent_wrapped_asset: AccountInfo<'info>,

    /// Wrapped mint authority.
    ///
    /// CHECK: We need this signer to authorize setting the new authority to None.
    #[account(
        seeds = [MINT_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    mint_authority: AccountInfo<'info>,

    token_program: Program<'info, token::Token>,
}

pub fn remove_wrapped_mint_authority(ctx: Context<RemoveWrappedMintAuthority>) -> Result<()> {
    token::set_authority(
        CpiContext::new_with_signer(
            ctx.accounts.token_program.to_account_info(),
            token::SetAuthority {
                current_authority: ctx.accounts.mint_authority.to_account_info(),
                account_or_mint: ctx.accounts.mint.to_account_info(),
            },
            &[&[MINT_AUTHORITY_SEED_PREFIX, &[ctx.bumps["mint_authority"]]]],
        ),
        token::spl_token::instruction::AuthorityType::MintTokens,
        None,
    )
}
