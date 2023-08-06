use crate::{
    constants::{MINT_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::EmptyArgs,
    processor::mint_wrapped_tokens,
    state::{Claim, RegisteredEmitter, WrappedAsset},
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, Token, TokenAccount};
use core_bridge_program::{
    constants::SOLANA_CHAIN,
    state::{PostedVaaV1Bytes, VaaV1MessageHash},
};
use wormhole_raw_vaas::{support::EncodedAmount, token_bridge::TokenBridgeMessage};
use wormhole_solana_common::SeedPrefix;

use super::validate_token_transfer_with_payload;

#[derive(Accounts)]
pub struct CompleteTransferWithPayloadWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    #[account(
        seeds = [
            PostedVaaV1Bytes::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedVaaV1Bytes>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            posted_vaa.emitter_address.as_ref(),
            &posted_vaa.emitter_chain.to_be_bytes(),
            &posted_vaa.sequence.to_be_bytes()
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    /// This account is a foreign token Bridge and is created via the Register Chain governance
    /// decree.
    ///
    /// NOTE: The seeds of this account are insane because they include the emitter address, which
    /// allows registering multiple emitter addresses for the same chain ID. These seeds are not
    /// checked via Anchor macro, but will be checked in the access control function instead.
    ///
    /// See the `require_valid_token_bridge_posted_vaa` instruction handler for more details.
    registered_emitter: Account<'info, RegisteredEmitter>,

    #[account(
        mut,
        token::mint = wrapped_mint,
    )]
    redeemer_token: Box<Account<'info, TokenAccount>>,

    redeemer_authority: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _relayer_fee_token: UncheckedAccount<'info>,

    #[account(
        seeds = [
            WRAPPED_MINT_SEED_PREFIX,
            &posted_vaa.payload.token_chain.to_be_bytes(),
            posted_vaa.payload.token_address.as_ref()
        ],
        bump
    )]
    wrapped_mint: Box<Account<'info, Mint>>,

    #[account(
        seeds = [wrapped_mint.key().as_ref()],
        bump
    )]
    wrapped_asset: Account<'info, WrappedAsset>,

    /// CHECK: This account is the authority that can burn and mint wrapped assets.
    #[account(
        seeds = [MINT_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    mint_authority: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _core_bridge_program: UncheckedAccount<'info>,

    token_program: Program<'info, Token>,
}

impl<'info> CompleteTransferWithPayloadWrapped<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let (token_chain, token_address) = validate_token_transfer_with_payload(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.redeemer_authority,
            &ctx.accounts.redeemer_token,
        )?;

        // For wrapped transfers, this token must have originated from another network.
        require_neq!(token_chain, SOLANA_CHAIN, TokenBridgeError::NativeAsset);

        // Wrapped asset account must agree with the encoded token address.
        require_eq!(
            token_address,
            ctx.accounts.wrapped_asset.token_address,
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWithPayloadWrapped::accounts(&ctx))]
pub fn complete_transfer_with_payload_wrapped(
    ctx: Context<CompleteTransferWithPayloadWrapped>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let norm_amount = TokenBridgeMessage::parse(&ctx.accounts.posted_vaa.payload)
        .unwrap()
        .transfer_with_message()
        .unwrap()
        .norm_amount();

    // Take transfer amount as-is.
    let mint_amount = EncodedAmount::from(norm_amount)
        .0
        .try_into()
        .map_err(|_| TokenBridgeError::U64Overflow)?;

    // Finally transfer encoded amount by minting to the redeemer's token account.
    mint_wrapped_tokens(
        &ctx.accounts.token_program,
        &ctx.accounts.wrapped_mint,
        &ctx.accounts.redeemer_token,
        &ctx.accounts.mint_authority,
        ctx.bumps["mint_authority"],
        mint_amount,
    )
}
