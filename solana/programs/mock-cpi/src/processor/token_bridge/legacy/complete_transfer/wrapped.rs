use anchor_lang::prelude::*;
use anchor_spl::token;
use token_bridge_program::sdk as token_bridge_sdk;

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
    token_bridge_program: Program<'info, token_bridge_sdk::cpi::TokenBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> token_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for MockLegacyCompleteTransferWrapped<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> token_bridge_sdk::cpi::CompleteTransfer<'info>
    for MockLegacyCompleteTransferWrapped<'info>
{
    fn token_bridge_program(&self) -> AccountInfo<'info> {
        self.token_bridge_program.to_account_info()
    }

    fn dst_token_account(&self) -> AccountInfo<'info> {
        self.recipient_token.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.token_bridge_wrapped_mint.to_account_info()
    }

    fn payer_token(&self) -> Option<AccountInfo<'info>> {
        Some(self.payer_token.to_account_info())
    }

    fn recipient(&self) -> Option<AccountInfo<'info>> {
        Some(self.recipient.to_account_info())
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

pub fn mock_legacy_complete_transfer_wrapped(
    ctx: Context<MockLegacyCompleteTransferWrapped>,
) -> Result<()> {
    token_bridge_sdk::cpi::complete_transfer_specified(
        ctx.accounts,
        true, // is_wrapped_asset
        None,
    )
}
