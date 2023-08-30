use crate::{
    constants::MINT_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::EmptyArgs,
    processor::mint_wrapped_tokens,
    state::{Claim, RegisteredEmitter, WrappedAsset},
};
use anchor_lang::prelude::*;
use anchor_spl::token;
use core_bridge_program::{constants::SOLANA_CHAIN, zero_copy::PostedVaaV1, CoreBridge};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct CompleteTransferWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: We will be performing zero-copy deserialization in the instruction handler.
    #[account(
        seeds = [
            PostedVaaV1::SEED_PREFIX,
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.message_hash().as_ref()
        ],
        bump,
        seeds::program = core_bridge_program
    )]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_address().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.emitter_chain().to_be_bytes().as_ref(),
            PostedVaaV1::parse(&posted_vaa.try_borrow_data()?)?.sequence().to_be_bytes().as_ref(),
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
    registered_emitter: Box<Account<'info, RegisteredEmitter>>,

    /// CHECK: Recipient token account. Because we verify the wrapped mint, we can depend on the
    /// Token Program to mint the right tokens to this account because it requires that this mint
    /// equals the wrapped mint.
    #[account(mut)]
    recipient_token: AccountInfo<'info>,

    /// CHECK: Payer (relayer) token account. Because we verify the wrapped mint, we can depend on
    /// the Token Program to mint the right tokens to this account because it requires that this
    /// mint equals the wrapped mint.
    #[account(mut)]
    payer_token: AccountInfo<'info>,

    /// CHECK: Wrapped mint (i.e. minted by Token Bridge program).
    ///
    /// NOTE: Instead of checking the seeds, we check that the mint authority is the Token Bridge's
    /// in access control.
    #[account(mut)]
    wrapped_mint: AccountInfo<'info>,

    #[account(
        seeds = [WrappedAsset::SEED_PREFIX, wrapped_mint.key().as_ref()],
        bump,
    )]
    wrapped_asset: Box<Account<'info, WrappedAsset>>,

    /// CHECK: This account is the authority that can burn and mint wrapped assets.
    #[account(
        seeds = [MINT_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    mint_authority: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    core_bridge_program: Program<'info, CoreBridge>,
    token_program: Program<'info, token::Token>,
}

impl<'info> CompleteTransferWrapped<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        crate::zero_copy::Mint::require_mint_authority(
            &ctx.accounts.wrapped_mint.try_borrow_data()?,
            Some(&ctx.accounts.mint_authority.key()),
        )?;

        let vaa = &ctx.accounts.posted_vaa;
        let vaa_key = vaa.key();
        let acc_data = vaa.try_borrow_data()?;
        let transfer = super::validate_posted_token_transfer(
            &vaa_key,
            &acc_data,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.recipient_token,
        )?;

        // For wrapped transfers, this token must have originated from another network.
        //
        // NOTE: This check may be redundant because our wrapped mint PDA should only exist for wrapped assets (i.e.
        // chain ID != 1. But there may be accounts that exist where the chain ID == 1, so we do perform this check as a
        // precaution).
        require_neq!(
            transfer.token_chain(),
            SOLANA_CHAIN,
            TokenBridgeError::NativeAsset
        );

        // Wrapped asset account must agree with the encoded token info.
        let asset = &ctx.accounts.wrapped_asset;
        require!(
            transfer.token_chain() == asset.token_chain
                && transfer.token_address() == asset.token_address,
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWrapped::constraints(&ctx))]
pub fn complete_transfer_wrapped(
    ctx: Context<CompleteTransferWrapped>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.posted_vaa.data.borrow();
    let vaa = PostedVaaV1::parse(&acc_data).unwrap();
    let msg = TokenBridgeMessage::parse(vaa.payload()).unwrap();
    let transfer = msg.transfer().unwrap();

    // We do not have to denormalize wrapped mint amounts because by definition wrapped mints can
    // only have a max of 8 decimals, which is the same as the cap for normalized amounts.
    let mut mint_amount = transfer
        .encoded_amount()
        .0
        .try_into()
        .map_err(|_| TokenBridgeError::U64Overflow)?;
    let relayer_payout = transfer.encoded_relayer_fee().0.try_into().unwrap();

    // Save references to the token accounts to be used later.
    let token_program = &ctx.accounts.token_program;
    let wrapped_mint = &ctx.accounts.wrapped_mint;
    let mint_authority = &ctx.accounts.mint_authority;
    let recipient_token = &ctx.accounts.recipient_token;
    let payer_token = &ctx.accounts.payer_token;

    // Mint authority is who has the authority to mint.
    let mint_authority_bump = ctx.bumps["mint_authority"];

    // If there is a payout to the relayer and the relayer's token account differs from the transfer
    // recipient's, we have to make an extra mint.
    if relayer_payout > 0 && recipient_token.key() != payer_token.key() {
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
        recipient_token,
        mint_authority,
        mint_authority_bump,
        mint_amount,
    )
}
