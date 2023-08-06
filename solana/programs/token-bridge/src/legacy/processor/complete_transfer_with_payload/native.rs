use crate::{
    constants::CUSTODY_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::EmptyArgs,
    processor::withdraw_native_tokens,
    state::{Claim, RegisteredEmitter},
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
pub struct CompleteTransferWithPayloadNative<'info> {
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
        token::mint = mint,
    )]
    redeemer_token: Box<Account<'info, TokenAccount>>,

    redeemer_authority: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _relayer_fee_token: UncheckedAccount<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        token::mint = mint,
        token::authority = custody_authority,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: Box<Account<'info, TokenAccount>>,

    mint: Box<Account<'info, Mint>>,

    /// CHECK: This account is the authority that can move tokens from the custody account.
    #[account(
        seeds = [CUSTODY_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    custody_authority: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _core_bridge_program: UncheckedAccount<'info>,

    token_program: Program<'info, Token>,
}

impl<'info> CompleteTransferWithPayloadNative<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        crate::utils::require_native_mint(&ctx.accounts.mint)?;

        let (token_chain, token_address) = validate_token_transfer_with_payload(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.redeemer_authority,
            &ctx.accounts.redeemer_token,
        )?;

        // For native transfers, this mint must have been created on Solana.
        require_eq!(token_chain, SOLANA_CHAIN, TokenBridgeError::WrappedAsset);

        // Mint account must agree with the encoded token address.
        require_eq!(
            Pubkey::from(token_address),
            ctx.accounts.mint.key(),
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWithPayloadNative::accounts(&ctx))]
pub fn complete_transfer_with_payload_native(
    ctx: Context<CompleteTransferWithPayloadNative>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let norm_amount = TokenBridgeMessage::parse(&ctx.accounts.posted_vaa.payload)
        .unwrap()
        .transfer_with_message()
        .unwrap()
        .amount();

    // Denormalize transfer amount based on this mint's decimals. When these transfers were made
    // outbound, the amounts were normalized, so it is safe to unwrap these operations.
    let transfer_amount = EncodedAmount::from(norm_amount)
        .denorm(ctx.accounts.mint.decimals)
        .try_into()
        .expect("Solana token amounts are u64");

    // Finally transfer encoded amount.
    withdraw_native_tokens(
        &ctx.accounts.token_program,
        &ctx.accounts.custody_token,
        &ctx.accounts.redeemer_token,
        &ctx.accounts.custody_authority,
        ctx.bumps["custody_authority"],
        transfer_amount,
    )
}
