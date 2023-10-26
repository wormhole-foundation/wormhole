use crate::{
    constants::{TRANSFER_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::instruction::TransferTokensArgs,
    state::LegacyWrappedAsset,
    utils,
};
use anchor_lang::prelude::*;
use core_bridge_program::{legacy::utils::LegacyAnchorized, sdk as core_bridge_sdk};
use ruint::aliases::U256;

#[derive(Accounts)]
pub struct TransferTokensWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Source token account. Because we verify the wrapped mint, we can depend on the Token
    /// Program to burn the right tokens to this account because it requires that this mint
    /// equals the wrapped mint.
    #[account(mut)]
    src_token: AccountInfo<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _src_owner: UncheckedAccount<'info>,

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
    wrapped_asset: Account<'info, LegacyAnchorized<0, LegacyWrappedAsset>>,

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

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_program: Program<'info, anchor_spl::token::Token>,
    core_bridge_program: Program<'info, core_bridge_sdk::cpi::CoreBridge>,
}

impl<'info> utils::cpi::Burn<'info> for TransferTokensWrapped<'info> {
    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.wrapped_mint.to_account_info()
    }

    fn from(&self) -> Option<AccountInfo<'info>> {
        Some(self.src_token.to_account_info())
    }

    fn authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.transfer_authority.to_account_info())
    }
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

impl<'info> core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, TransferTokensArgs>
    for TransferTokensWrapped<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyTransferTokensWrapped";

    const ANCHOR_IX_FN: fn(Context<Self>, TransferTokensArgs) -> Result<()> =
        transfer_tokens_wrapped;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_transfer_tokens_account_infos(account_infos)
    }
}

impl<'info> TransferTokensWrapped<'info> {
    fn constraints(args: &TransferTokensArgs) -> Result<()> {
        // Cannot configure a fee greater than the total transfer amount.
        require!(
            args.relayer_fee <= args.amount,
            TokenBridgeError::InvalidRelayerFee
        );

        // Done.
        Ok(())
    }
}

#[access_control(TransferTokensWrapped::constraints(&args))]
fn transfer_tokens_wrapped(
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
    utils::cpi::burn(
        ctx.accounts,
        amount,
        Some(&[&[
            TRANSFER_AUTHORITY_SEED_PREFIX,
            &[ctx.bumps["transfer_authority"]],
        ]]),
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
