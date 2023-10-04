use crate::{
    constants::{CUSTODY_AUTHORITY_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::instruction::TransferTokensArgs,
    utils::{self, TruncateAmount},
    zero_copy::Mint,
};
use anchor_lang::prelude::*;
use core_bridge_program::sdk::{self as core_bridge_sdk, LoadZeroCopy};
use ruint::aliases::U256;
use wormhole_raw_vaas::support::EncodedAmount;

#[derive(Accounts)]
pub struct TransferTokensNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Source token account. Because we check the mint of the custody token account, we can
    /// be sure that this token account is the same mint since the Token Program transfer
    /// instruction handler checks that the mints of these two accounts must be the same.
    #[account(mut)]
    src_token: AccountInfo<'info>,

    /// CHECK: Native mint. We ensure this mint is not one that has originated from a foreign
    /// network in access control.
    mint: AccountInfo<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        token::mint = mint,
        token::authority = custody_authority,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: Account<'info, anchor_spl::token::TokenAccount>,

    /// CHECK: This authority is whom the source token account owner delegates spending approval for
    /// transferring native assets or burning wrapped assets.
    #[account(
        seeds = [TRANSFER_AUTHORITY_SEED_PREFIX],
        bump
    )]
    transfer_authority: AccountInfo<'info>,

    /// CHECK: This account is the authority that can move tokens from the custody account.
    #[account(
        seeds = [CUSTODY_AUTHORITY_SEED_PREFIX],
        bump
    )]
    custody_authority: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
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

impl<'info> utils::cpi::Transfer<'info> for TransferTokensNative<'info> {
    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn from(&self) -> Option<AccountInfo<'info>> {
        Some(self.src_token.to_account_info())
    }

    fn authority(&self) -> Option<AccountInfo<'info>> {
        Some(self.transfer_authority.to_account_info())
    }
}

impl<'info> core_bridge_sdk::cpi::system_program::CreateAccount<'info>
    for TransferTokensNative<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

impl<'info> core_bridge_sdk::cpi::PublishMessage<'info> for TransferTokensNative<'info> {
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
    for TransferTokensNative<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyTransferTokensNative";

    const ANCHOR_IX_FN: fn(Context<Self>, TransferTokensArgs) -> Result<()> =
        transfer_tokens_native;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_transfer_tokens_account_infos(account_infos)
    }
}

impl<'info> TransferTokensNative<'info> {
    fn constraints(ctx: &Context<Self>, args: &TransferTokensArgs) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        let mint = Mint::load(&ctx.accounts.mint)?;
        require!(
            !crate::utils::is_wrapped_mint(&mint),
            TokenBridgeError::WrappedAsset
        );

        // Cannot configure a fee greater than the total transfer amount.
        require!(
            args.relayer_fee <= args.amount,
            TokenBridgeError::InvalidRelayerFee
        );

        // Done.
        Ok(())
    }
}

#[access_control(TransferTokensNative::constraints(&ctx, &args))]
fn transfer_tokens_native(
    ctx: Context<TransferTokensNative>,
    args: TransferTokensArgs,
) -> Result<()> {
    let TransferTokensArgs {
        nonce,
        amount,
        relayer_fee,
        recipient,
        recipient_chain,
    } = args;

    let mint = Mint::load(&ctx.accounts.mint).unwrap();

    // Deposit native assets from the sender's account into the custody account.
    utils::cpi::transfer(
        ctx.accounts,
        ctx.accounts.custody_token.as_ref(),
        mint.truncate_amount(amount),
        Some(&[&[
            TRANSFER_AUTHORITY_SEED_PREFIX,
            &[ctx.bumps["transfer_authority"]],
        ]]),
    )?;

    // Prepare Wormhole message. We need to normalize these amounts because we are working with
    // native assets.
    let decimals = mint.decimals();
    let token_transfer = crate::messages::Transfer {
        norm_amount: EncodedAmount::norm(U256::from(amount), decimals).0,
        token_address: ctx.accounts.mint.key().to_bytes(),
        token_chain: core_bridge_sdk::SOLANA_CHAIN,
        recipient,
        recipient_chain,
        norm_relayer_fee: EncodedAmount::norm(U256::from(relayer_fee), decimals).0,
    };

    // Finally publish Wormhole message using the Core Bridge.
    utils::cpi::post_token_bridge_message(
        ctx.accounts,
        &ctx.accounts.core_message,
        nonce,
        token_transfer,
    )
}
