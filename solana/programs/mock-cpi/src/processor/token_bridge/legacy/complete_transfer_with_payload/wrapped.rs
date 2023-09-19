use anchor_lang::prelude::*;
use anchor_spl::token;
use token_bridge_program::sdk as token_bridge_sdk;

use crate::constants::CUSTOM_REDEEMER_SEED_PREFIX;

#[derive(Accounts)]
pub struct MockLegacyCompleteTransferWithPayloadWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: This account is needed for the Token Bridge program.
    #[account(
        seeds = [token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX],
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
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> token_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for MockLegacyCompleteTransferWithPayloadWrapped<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> token_bridge_sdk::cpi::CompleteTransfer<'info>
    for MockLegacyCompleteTransferWithPayloadWrapped<'info>
{
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn dst_token_account(&self) -> AccountInfo<'info> {
        self.dst_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.token_bridge_wrapped_mint.to_account_info()
    }

    fn redeemer_authority(&self) -> Option<AccountInfo<'info>> {
        match (
            &self.token_bridge_program_redeemer_authority,
            &self.token_bridge_custom_redeemer_authority,
        ) {
            (Some(authority), _) => Some(authority.to_account_info()),
            (None, Some(authority)) => Some(authority.to_account_info()),
            (None, None) => None,
        }
    }

    fn token_bridge_claim(&self) -> AccountInfo<'info> {
        self.token_bridge_claim.to_account_info()
    }

    fn token_bridge_mint_authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.token_bridge_mint_authority.to_account_info())
    }

    fn token_bridge_registered_emitter(&self) -> AccountInfo<'info> {
        self.token_bridge_registered_emitter.to_account_info()
    }

    fn token_bridge_wrapped_asset(&self) -> Option<AccountInfo<'info>> {
        Some(self.token_bridge_wrapped_asset.to_account_info())
    }

    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn vaa(&self) -> AccountInfo<'info> {
        self.vaa.to_account_info()
    }
}

pub fn mock_legacy_complete_transfer_with_payload_wrapped(
    ctx: Context<MockLegacyCompleteTransferWithPayloadWrapped>,
) -> Result<()> {
    let (redeemer_seed_prefix, redeemer_bump) = match (
        &ctx.accounts.token_bridge_program_redeemer_authority,
        &ctx.accounts.token_bridge_custom_redeemer_authority,
    ) {
        (Some(_), _) => (
            token_bridge_sdk::PROGRAM_REDEEMER_SEED_PREFIX,
            ctx.bumps["token_bridge_program_redeemer_authority"],
        ),
        (None, Some(_)) => (
            CUSTOM_REDEEMER_SEED_PREFIX,
            ctx.bumps["token_bridge_custom_redeemer_authority"],
        ),
        (None, None) => return err!(ErrorCode::AccountNotEnoughKeys),
    };

    token_bridge_sdk::cpi::complete_transfer_specified(
        ctx.accounts,
        true, // is_wrapped_asset
        Some(&[&[redeemer_seed_prefix, &[redeemer_bump]]]),
    )
}
