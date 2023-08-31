use crate::{
    constants::{EMITTER_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    legacy::TransferTokensWithPayloadArgs,
    processor::{burn_wrapped_tokens, post_token_bridge_message},
    state::WrappedAsset,
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::{self, state::Config as CoreBridgeConfig, CoreBridge};
use ruint::aliases::U256;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct TransferTokensWithPayloadWrapped<'info> {
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

    #[account(
        seeds = [WrappedAsset::SEED_PREFIX, wrapped_mint.key().as_ref()],
        bump
    )]
    wrapped_asset: Box<Account<'info, WrappedAsset>>,

    /// CHECK: This authority is whom the source token account owner delegates spending approval for
    /// transferring native assets or burning wrapped assets.
    #[account(
        seeds = [TRANSFER_AUTHORITY_SEED_PREFIX],
        bump
    )]
    transfer_authority: AccountInfo<'info>,

    /// We need to deserialize this account to determine the Wormhole message fee. We do not have to
    /// check the seeds here because the Core Bridge program will do this for us.
    #[account(mut)]
    core_bridge_config: Box<Account<'info, CoreBridgeConfig>>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_message: Signer<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump,
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    #[account(mut)]
    core_fee_collector: Option<UncheckedAccount<'info>>,

    /// CHECK: Previously needed sysvar.
    _clock: UncheckedAccount<'info>,

    /// This account is used to authorize sending the transfer with payload. This signer is either
    /// an ordinary signer (whose pubkey will be encoded as the sender address) or a program's PDA,
    /// whose seeds are [b"sender"] (and the program ID will be encoded as the sender address).
    sender_authority: Signer<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> core_bridge_program::sdk::cpi::InvokeCoreBridge<'info>
    for TransferTokensWithPayloadWrapped<'info>
{
    fn core_bridge_program(&self) -> AccountInfo<'info> {
        self.core_bridge_program.to_account_info()
    }
}

impl<'info> core_bridge_program::sdk::cpi::InvokePostMessageV1<'info>
    for TransferTokensWithPayloadWrapped<'info>
{
    fn payer(&self) -> AccountInfo<'info> {
        self.payer.to_account_info()
    }

    fn config(&self) -> AccountInfo<'info> {
        self.core_bridge_config.to_account_info()
    }

    fn message(&self) -> AccountInfo<'info> {
        self.core_message.to_account_info()
    }

    fn emitter(&self) -> AccountInfo<'info> {
        self.core_emitter.to_account_info()
    }

    fn emitter_sequence(&self) -> AccountInfo<'info> {
        self.core_emitter_sequence.to_account_info()
    }

    fn fee_collector(&self) -> Option<AccountInfo<'info>> {
        self.core_fee_collector
            .as_ref()
            .map(|acc| acc.to_account_info())
    }

    fn system_program(&self) -> AccountInfo<'info> {
        self.system_program.to_account_info()
    }
}

pub fn transfer_tokens_with_payload_wrapped(
    ctx: Context<TransferTokensWithPayloadWrapped>,
    args: TransferTokensWithPayloadArgs,
) -> Result<()> {
    let TransferTokensWithPayloadArgs {
        nonce,
        amount,
        redeemer,
        redeemer_chain,
        payload,
        cpi_program_id,
    } = args;

    // Generate sender address, which may either be the program ID or the sender authority's pubkey.
    //
    // NOTE: We perform the derivation check here instead of in the access control because we do not
    // want to spend compute units to re-derive the authority if cpi_program_id is Some(pubkey).
    let sender = crate::utils::new_sender_address(&ctx.accounts.sender_authority, cpi_program_id)?;

    // Burn wrapped assets from the source token account.
    burn_wrapped_tokens(
        &ctx.accounts.token_program,
        &ctx.accounts.wrapped_mint,
        &ctx.accounts.src_token,
        &ctx.accounts.transfer_authority,
        ctx.bumps["transfer_authority"],
        amount,
    )?;

    // Prepare Wormhole message. Amounts do not need to be normalized because we are working with
    // wrapped assets.
    let wrapped_asset = &ctx.accounts.wrapped_asset;
    let token_transfer = crate::messages::TransferWithMessage {
        norm_amount: U256::from(amount),
        token_address: wrapped_asset.token_address,
        token_chain: wrapped_asset.token_chain,
        redeemer,
        redeemer_chain,
        sender,
        payload,
    };

    // Finally publish Wormhole message using the Core Bridge.
    post_token_bridge_message(
        ctx.accounts,
        ctx.bumps["core_emitter"],
        nonce,
        token_transfer,
    )
}
