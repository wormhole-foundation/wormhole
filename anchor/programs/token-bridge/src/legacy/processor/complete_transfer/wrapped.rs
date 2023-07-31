use crate::{
    constants::MINT_AUTHORITY_SEED_PREFIX,
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

use super::validate_token_transfer;

#[derive(Accounts)]
pub struct CompleteTransferWrapped<'info> {
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
    dst_token: Box<Account<'info, TokenAccount>>,

    #[account(
        mut,
        token::mint = wrapped_mint,
        token::authority = payer
    )]
    payer_token: Box<Account<'info, TokenAccount>>,

    /// NOTE: We do not need to check this PDA because access control checks the wrapped asset's
    /// token address.
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

impl<'info> CompleteTransferWrapped<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let (token_chain, token_address) = validate_token_transfer(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.dst_token,
        )?;

        // For wrapped transfers, this token must have originated from another network.
        //
        // NOTE: This check may be redundant because our wrapped mint PDA should only exist for wrapped assets (i.e.
        // chain ID != 1. But there may be accounts that exist where the chain ID == 1, so we do perform this check as a
        // precaution).
        require_neq!(token_chain, SOLANA_CHAIN, TokenBridgeError::NativeAsset);

        // Wrapped asset account must agree with the encoded token info.
        let asset = &ctx.accounts.wrapped_asset;
        require!(
            token_chain == asset.token_chain,
            TokenBridgeError::InvalidMint
        );
        require!(
            token_address == asset.token_address,
            TokenBridgeError::InvalidMint,
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWrapped::accounts(&ctx))]
pub fn complete_transfer_wrapped(
    ctx: Context<CompleteTransferWrapped>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let transfer = TokenBridgeMessage::parse(&ctx.accounts.posted_vaa.payload)
        .unwrap()
        .transfer()
        .unwrap();

    // We do not have to denormalize wrapped mint amounts because by definition wrapped mints can
    // only have a max of 8 decimals, which is the same as the cap for normalized amounts.
    let mut mint_amount = EncodedAmount::from(transfer.amount())
        .0
        .try_into()
        .map_err(|_| TokenBridgeError::U64Overflow)?;
    let relayer_payout = EncodedAmount::from(transfer.relayer_fee())
        .0
        .try_into()
        .unwrap();

    // Save references to the token accounts to be used later.
    let token_program = &ctx.accounts.token_program;
    let wrapped_mint = &ctx.accounts.wrapped_mint;
    let mint_authority = &ctx.accounts.mint_authority;
    let dst_token = &ctx.accounts.dst_token;
    let payer_token = &ctx.accounts.payer_token;

    // Mint authority is who has the authority to mint.
    let mint_authority_bump = ctx.bumps["mint_authority"];

    // If there is a payout to the relayer and the relayer's token account differs from the transfer
    // recipient's, we have to make an extra mint.
    if relayer_payout > 0 && dst_token.key() != payer_token.key() {
        // NOTE: This math operation is safe because the relayer payout is always <= to the
        // total outbound transfer amount.
        mint_amount -= relayer_payout;

        mint_wrapped_tokens(
            token_program,
            wrapped_mint,
            payer_token,
            mint_authority,
            mint_authority_bump,
            relayer_payout,
        )?;
    }

    // If there is any amount left after the relayer payout, finally mint remaining.
    mint_wrapped_tokens(
        token_program,
        wrapped_mint,
        dst_token,
        mint_authority,
        mint_authority_bump,
        mint_amount,
    )
}
