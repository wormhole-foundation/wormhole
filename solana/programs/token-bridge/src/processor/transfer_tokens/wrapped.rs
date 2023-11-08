use crate::{
    constants::{TRANSFER_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    state::LegacyWrappedAsset,
    utils,
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::{legacy::utils::LegacyAnchorized, sdk as core_bridge_sdk};
use ruint::aliases::U256;

use super::TransferTokensArgs;

#[derive(Accounts)]
pub struct TransferTokensWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// Source token account. Because we verify the wrapped mint, we can depend on the Token
    /// Program to burn the right tokens to this account because it requires that this mint
    /// equals the wrapped mint.
    #[account(
        mut,
        token::mint = wrapped_mint,
    )]
    src_token: Account<'info, token::TokenAccount>,

    /// CHECK: Wrapped mint (i.e. minted by Token Bridge program).
    #[account(
        mut,
        seeds = [
            WRAPPED_MINT_SEED_PREFIX,
            &wrapped_asset.token_chain.to_be_bytes(),
            wrapped_asset.token_address.as_ref()
        ],
        bump
    )]
    wrapped_mint: AccountInfo<'info>,

    /// Wrapped asset account, which is deserialized as its legacy representation. The latest
    /// version has an additional field (sequence number), which may not deserialize if wrapped
    /// metadata were not attested again to realloc this account. So we must deserialize this as the
    /// legacy representation.
    #[account(
        seeds = [LegacyWrappedAsset::SEED_PREFIX, wrapped_mint.key().as_ref()],
        bump,
    )]
    wrapped_asset: Account<'info, LegacyAnchorized<LegacyWrappedAsset>>,

    /// CHECK: This authority is whom the source token account owner delegates spending approval for
    /// transferring native assets or burning wrapped assets.
    #[account(
        seeds = [TRANSFER_AUTHORITY_SEED_PREFIX],
        bump
    )]
    transfer_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_bridge_config: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_message: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    /// This PDA address is checked in `post_token_bridge_message`.
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<UncheckedAccount<'info>>,

    system_program: Program<'info, System>,
    token_program: Program<'info, token::Token>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for TransferTokensWrapped<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for TransferTokensWrapped<'info> {
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }

    fn core_bridge_config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn core_emitter_authority(&self) -> AccountInfo<'info> {
        self.core_emitter.to_account_info()
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

impl<'info> TransferTokensWrapped<'info> {
    fn constraints(args: &TransferTokensArgs) -> Result<()> {
        super::require_valid_relayer_fee(args)
    }
}

#[access_control(TransferTokensWrapped::constraints(&args))]
pub fn transfer_tokens_wrapped(
    ctx: Context<TransferTokensWrapped>,
    args: TransferTokensArgs,
) -> Result<()> {
    let TransferTokensArgs {
        nonce,
        amount,
        relayer_fee,
        recipient,
        recipient_chain,
    } = args;

    // Burn wrapped assets from the sender's account.
    token::burn(
        CpiContext::new_with_signer(
            ctx.accounts.token_program.to_account_info(),
            token::Burn {
                mint: ctx.accounts.wrapped_mint.to_account_info(),
                from: ctx.accounts.src_token.to_account_info(),
                authority: ctx.accounts.transfer_authority.to_account_info(),
            },
            &[&[
                TRANSFER_AUTHORITY_SEED_PREFIX,
                &[ctx.bumps["transfer_authority"]],
            ]],
        ),
        amount,
    )?;

    // Prepare Wormhole message. Amounts do not need to be normalized because we are working with
    // wrapped assets.
    let wrapped_asset = &ctx.accounts.wrapped_asset;
    let token_transfer = crate::messages::Transfer {
        norm_amount: U256::from(amount),
        token_address: wrapped_asset.token_address,
        token_chain: wrapped_asset.token_chain,
        recipient,
        recipient_chain,
        norm_relayer_fee: U256::from(relayer_fee),
    };

    // Finally publish Wormhole message using the Core Bridge.
    utils::cpi::post_token_bridge_message(
        ctx.accounts,
        &ctx.accounts.core_message,
        nonce,
        token_transfer,
    )
}
