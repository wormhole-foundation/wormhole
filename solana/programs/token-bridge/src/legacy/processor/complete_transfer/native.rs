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

use super::validate_token_transfer;

#[derive(Accounts)]
pub struct CompleteTransferNative<'info> {
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
    dst_token: Box<Account<'info, TokenAccount>>,

    #[account(
        mut,
        token::mint = mint,
        token::authority = payer
    )]
    payer_token: Box<Account<'info, TokenAccount>>,

    #[account(
        init_if_needed,
        payer = payer,
        token::mint = mint,
        token::authority = custody_authority,
        seeds = [mint.key().as_ref()],
        bump,
    )]
    custody_token: Box<Account<'info, TokenAccount>>,

    mint: Account<'info, Mint>,

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

impl<'info> CompleteTransferNative<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        crate::utils::require_native_mint(&ctx.accounts.mint)?;

        let (token_chain, token_address) = validate_token_transfer(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.dst_token,
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

#[access_control(CompleteTransferNative::accounts(&ctx))]
pub fn complete_transfer_native(
    ctx: Context<CompleteTransferNative>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let transfer = TokenBridgeMessage::parse(&ctx.accounts.posted_vaa.payload)
        .unwrap()
        .transfer()
        .unwrap();
    let decimals = ctx.accounts.mint.decimals;

    // Denormalize transfer transfer_amount and relayer payouts based on this mint's decimals. When these
    // transfers were made outbound, the amounts were normalized, so it is safe to unwrap these
    // operations.
    let mut transfer_amount = EncodedAmount::from(transfer.amount())
        .denorm(decimals)
        .try_into()
        .expect("Solana token amounts are u64");
    let relayer_payout = EncodedAmount::from(transfer.relayer_fee())
        .denorm(decimals)
        .try_into()
        .unwrap();

    // Save references to these accounts to be used later.
    let token_program = &ctx.accounts.token_program;
    let custody_token = &ctx.accounts.custody_token;
    let custody_authority = &ctx.accounts.custody_authority;
    let dst_token = &ctx.accounts.dst_token;
    let payer_token = &ctx.accounts.payer_token;

    // Custody authority is who has the authority to transfer tokens from the custody account.
    let custody_authority_bump = ctx.bumps["custody_authority"];

    // If there is a payout to the relayer and the relayer's token account differs from the transfer
    // recipient's, we have to make an extra transfer.
    if relayer_payout > 0 && dst_token.key() != payer_token.key() {
        // NOTE: This math operation is safe because the relayer payout is always <= to the
        // total outbound transfer transfer_amount.
        transfer_amount -= relayer_payout;

        withdraw_native_tokens(
            token_program,
            custody_token,
            payer_token,
            custody_authority,
            custody_authority_bump,
            relayer_payout,
        )?;
    }

    // Finally transfer remaining transfer_amount to recipient.
    withdraw_native_tokens(
        token_program,
        custody_token,
        dst_token,
        custody_authority,
        custody_authority_bump,
        transfer_amount,
    )
}
