use crate::{
    constants::{EMITTER_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    legacy::LegacyTransferTokensWithPayloadArgs,
    processor::{burn_wrapped_tokens, post_token_bridge_message, PostTokenBridgeMessage},
    state::WrappedAsset,
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, Token, TokenAccount};
use core_bridge_program::{self, state::BridgeProgramData, CoreBridge};
use wormhole_solana_common::SeedPrefix;

use super::new_sender_address;

#[derive(Accounts)]
pub struct TransferTokensWithPayloadWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    #[account(
        mut,
        token::mint = wrapped_mint
    )]
    src_token: Box<Account<'info, TokenAccount>>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _src_owner: UncheckedAccount<'info>,

    #[account(
        seeds = [
            WRAPPED_MINT_SEED_PREFIX,
            &wrapped_asset.token_chain.to_be_bytes(),
            wrapped_asset.token_address.as_ref()
        ],
        bump
    )]
    wrapped_mint: Box<Account<'info, Mint>>,

    #[account(
        seeds = [wrapped_mint.key().as_ref()],
        bump
    )]
    wrapped_asset: Account<'info, WrappedAsset>,

    /// CHECK: This authority is whom the source token account owner delegates spending approval for
    /// transferring native assets or burning wrapped assets.
    #[account(
        seeds = [TRANSFER_AUTHORITY_SEED_PREFIX],
        bump
    )]
    transfer_authority: AccountInfo<'info>,

    /// We need to deserialize this account to determine the Wormhole message fee.
    #[account(
        seeds = [BridgeProgramData::seed_prefix()],
        bump,
        seeds::program = core_bridge_program
    )]
    core_bridge: Account<'info, BridgeProgramData>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_message: UncheckedAccount<'info>,

    /// CHECK: We need this emitter to invoke the Core Bridge program to send Wormhole messages.
    #[account(
        seeds = [EMITTER_SEED_PREFIX],
        bump,
    )]
    core_emitter: AccountInfo<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_emitter_sequence: UncheckedAccount<'info>,

    /// CHECK: This account is needed for the Core Bridge program.
    core_fee_collector: UncheckedAccount<'info>,

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
    token_program: Program<'info, Token>,
}

pub fn transfer_tokens_with_payload_wrapped(
    ctx: Context<TransferTokensWithPayloadWrapped>,
    args: LegacyTransferTokensWithPayloadArgs,
) -> Result<()> {
    let LegacyTransferTokensWithPayloadArgs {
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
    let sender = new_sender_address(&ctx.accounts.sender_authority, cpi_program_id)?;

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
    let token_transfer = super::TransferWithMessage {
        norm_amount: amount.try_into().unwrap(),
        token_address: wrapped_asset.token_address.into(),
        token_chain: wrapped_asset.token_chain,
        redeemer: redeemer.into(),
        redeemer_chain,
        sender,
        payload,
    };

    // Finally publish Wormhole message using the Core Bridge.
    post_token_bridge_message(
        PostTokenBridgeMessage {
            core_bridge: &ctx.accounts.core_bridge,
            core_message: &ctx.accounts.core_message,
            core_emitter: &ctx.accounts.core_emitter,
            core_emitter_sequence: &ctx.accounts.core_emitter_sequence,
            payer: &ctx.accounts.payer,
            core_fee_collector: &ctx.accounts.core_fee_collector,
            system_program: &ctx.accounts.system_program,
            core_bridge_program: &ctx.accounts.core_bridge_program,
        },
        ctx.bumps["core_emitter"],
        nonce,
        token_transfer,
    )
}
