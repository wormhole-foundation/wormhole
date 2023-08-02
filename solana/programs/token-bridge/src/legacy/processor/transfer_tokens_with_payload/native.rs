use crate::{
    constants::{
        CUSTODY_AUTHORITY_SEED_PREFIX, EMITTER_SEED_PREFIX, TRANSFER_AUTHORITY_SEED_PREFIX,
    },
    legacy::LegacyTransferTokensWithPayloadArgs,
    processor::{deposit_native_tokens, post_token_bridge_message, PostTokenBridgeMessage},
    utils,
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, Token, TokenAccount};
use core_bridge_program::{self, constants::SOLANA_CHAIN, state::BridgeProgramData, CoreBridge};
use wormhole_raw_vaas::support::EncodedAmount;
use wormhole_solana_common::SeedPrefix;

use super::new_sender_address;

#[derive(Accounts)]
pub struct TransferTokensWithPayloadNative<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    #[account(
        mut,
        token::mint = mint
    )]
    src_token: Box<Account<'info, TokenAccount>>,

    mint: Box<Account<'info, Mint>>,

    #[account(
        init_if_needed,
        payer = payer,
        token::mint = mint,
        token::authority = custody_authority,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: Box<Account<'info, TokenAccount>>,

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

impl<'info> TransferTokensWithPayloadNative<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        utils::require_native_mint(&ctx.accounts.mint)
    }
}

#[access_control(TransferTokensWithPayloadNative::accounts(&ctx))]
pub fn transfer_tokens_with_payload_native(
    ctx: Context<TransferTokensWithPayloadNative>,
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

    // Deposit native assets from the source token account into the custody account.
    deposit_native_tokens(
        &ctx.accounts.token_program,
        &ctx.accounts.src_token,
        &ctx.accounts.custody_token,
        &ctx.accounts.transfer_authority,
        ctx.bumps["transfer_authority"],
        amount,
    )?;

    // Prepare Wormhole message. We need to normalize these amounts because we are working with
    // native assets.
    let mint = &ctx.accounts.mint;
    let token_transfer = super::TransferWithMessage {
        norm_amount: EncodedAmount::norm(amount.into(), mint.decimals),
        token_address: mint.key().to_bytes().into(),
        token_chain: SOLANA_CHAIN,
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
