use crate::{
    constants::CUSTODY_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::EmptyArgs,
    processor::withdraw_native_tokens,
    state::{Claim, RegisteredEmitter},
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, Token, TokenAccount};
use core_bridge_program::{constants::SOLANA_CHAIN, zero_copy::PostedVaaV1, CoreBridge};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;
use wormhole_solana_common::SeedPrefix;

#[derive(Accounts)]
pub struct CompleteTransferWithPayloadNative<'info> {
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
    registered_emitter: Account<'info, RegisteredEmitter>,

    #[account(
        mut,
        token::mint = mint,
    )]
    recipient_token: Box<Account<'info, TokenAccount>>,

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
    core_bridge_program: Program<'info, CoreBridge>,
    token_program: Program<'info, Token>,
}

impl<'info> CompleteTransferWithPayloadNative<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        // Make sure the mint authority is not the Token Bridge's. If it is, then this mint
        // originated from a foreign network.
        crate::utils::require_native_mint(&ctx.accounts.mint)?;

        let vaa = &ctx.accounts.posted_vaa;
        let vaa_key = vaa.key();
        let acc_data = vaa.try_borrow_data()?;
        let transfer = super::validate_posted_token_transfer_with_payload(
            &vaa_key,
            &acc_data,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.redeemer_authority,
            &ctx.accounts.recipient_token,
        )?;

        // For native transfers, this mint must have been created on Solana.
        require_eq!(
            transfer.token_chain(),
            SOLANA_CHAIN,
            TokenBridgeError::WrappedAsset
        );

        // Mint account must agree with the encoded token address.
        require_eq!(
            ctx.accounts.mint.key(),
            Pubkey::from(transfer.token_address()),
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWithPayloadNative::constraints(&ctx))]
pub fn complete_transfer_with_payload_native(
    ctx: Context<CompleteTransferWithPayloadNative>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    let acc_data = ctx.accounts.posted_vaa.data.borrow();
    let vaa = PostedVaaV1::parse(&acc_data).unwrap();

    // Denormalize transfer amount based on this mint's decimals. When these transfers were made
    // outbound, the amounts were normalized, so it is safe to unwrap these operations.
    let transfer_amount = TokenBridgeMessage::parse(vaa.payload())
        .unwrap()
        .transfer_with_message()
        .unwrap()
        .encoded_amount()
        .denorm(ctx.accounts.mint.decimals)
        .try_into()
        .expect("Solana token amounts are u64");

    // Finally transfer encoded amount.
    withdraw_native_tokens(
        &ctx.accounts.token_program,
        &ctx.accounts.custody_token,
        &ctx.accounts.recipient_token,
        &ctx.accounts.custody_authority,
        ctx.bumps["custody_authority"],
        transfer_amount,
    )
}
