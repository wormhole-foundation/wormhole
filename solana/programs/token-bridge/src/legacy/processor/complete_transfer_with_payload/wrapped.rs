use crate::{
    constants::MINT_AUTHORITY_SEED_PREFIX,
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{Claim, LegacyWrappedAsset, RegisteredEmitter},
    utils,
};
use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN, legacy::utils::LegacyAnchorized, zero_copy::PostedVaaV1,
};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

#[derive(Accounts)]
pub struct CompleteTransferWithPayloadWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

    /// CHECK: Posted VAA account, which will be read via zero-copy deserialization in the
    /// instruction handler, which also checks this account discriminator (so there is no need to
    /// check PDA seeds here).
    #[account(owner = core_bridge_program::ID)]
    posted_vaa: AccountInfo<'info>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            PostedVaaV1::parse(&posted_vaa)
                .map(|vaa| vaa.emitter_address())?
                .as_ref(),
            PostedVaaV1::parse(&posted_vaa)
                .map(|vaa| vaa.emitter_chain().to_be_bytes())?
                .as_ref(),
            PostedVaaV1::parse(&posted_vaa)
                .map(|vaa| vaa.sequence().to_be_bytes())?
                .as_ref(),
        ],
        bump,
    )]
    claim: Account<'info, LegacyAnchorized<0, Claim>>,

    /// This account is a foreign token Bridge and is created via the Register Chain governance
    /// decree.
    ///
    /// NOTE: The seeds of this account are insane because they include the emitter address, which
    /// allows registering multiple emitter addresses for the same chain ID. These seeds are not
    /// checked via Anchor macro, but will be checked in the access control function instead.
    ///
    /// See the `require_valid_token_bridge_posted_vaa` instruction handler for more details.
    registered_emitter: Box<Account<'info, LegacyAnchorized<0, RegisteredEmitter>>>,

    /// CHECK: Destination token account. Because we verify the wrapped mint, we can depend on the
    /// Token Program to mint the right tokens to this account because it requires that this mint
    /// equals the wrapped mint.
    #[account(mut)]
    dst_token: AccountInfo<'info>,

    redeemer_authority: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _relayer_fee_token: UncheckedAccount<'info>,

    /// CHECK: Wrapped mint (i.e. minted by Token Bridge program).
    ///
    /// NOTE: Instead of checking the seeds, we check that the mint authority is the Token Bridge's
    /// in access control.
    #[account(mut)]
    wrapped_mint: AccountInfo<'info>,

    /// Wrapped asset account, which is deserialized as its legacy representation. The latest
    /// version has an additional field (sequence number), which may not deserialize if wrapped
    /// metadata were not attested again to realloc this account. So we must deserialize this as the
    /// legacy representation.
    #[account(
        seeds = [LegacyWrappedAsset::SEED_PREFIX, wrapped_mint.key().as_ref()],
        bump,
    )]
    wrapped_asset: Box<Account<'info, LegacyAnchorized<0, LegacyWrappedAsset>>>,

    /// CHECK: This account is the authority that can burn and mint wrapped assets.
    #[account(
        seeds = [MINT_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    mint_authority: AccountInfo<'info>,

    /// CHECK: Previously needed sysvar.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_program: Program<'info, anchor_spl::token::Token>,
}

impl<'info> utils::cpi::MintTo<'info> for CompleteTransferWithPayloadWrapped<'info> {
    fn token_program(&self) -> AccountInfo<'info> {
        self.token_program.to_account_info()
    }

    fn mint(&self) -> AccountInfo<'info> {
        self.wrapped_mint.to_account_info()
    }

    fn mint_authority(&self) -> AccountInfo<'info> {
        self.mint_authority.to_account_info()
    }
}

impl<'info> core_bridge_program::legacy::utils::ProcessLegacyInstruction<'info, EmptyArgs>
    for CompleteTransferWithPayloadWrapped<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyCompleteTransferWithPayloadWrapped";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> =
        complete_transfer_with_payload_wrapped;

    fn order_account_infos<'a>(
        account_infos: &'a [AccountInfo<'info>],
    ) -> Result<Vec<AccountInfo<'info>>> {
        super::order_complete_transfer_with_payload_account_infos(account_infos)
    }
}

impl<'info> CompleteTransferWithPayloadWrapped<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        crate::zero_copy::Mint::require_mint_authority(
            &ctx.accounts.wrapped_mint,
            Some(&ctx.accounts.mint_authority.key()),
        )?;

        let (token_chain, token_address) = super::validate_posted_token_transfer_with_payload(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
            &ctx.accounts.redeemer_authority,
            &ctx.accounts.dst_token,
        )?;

        // For wrapped transfers, this token must have originated from another network.
        require_neq!(token_chain, SOLANA_CHAIN, TokenBridgeError::NativeAsset);

        // Wrapped asset account must agree with the encoded token info.
        let asset = &ctx.accounts.wrapped_asset;
        require!(
            token_chain == asset.token_chain && token_address == asset.token_address,
            TokenBridgeError::InvalidMint
        );

        // Done.
        Ok(())
    }
}

#[access_control(CompleteTransferWithPayloadWrapped::constraints(&ctx))]
fn complete_transfer_with_payload_wrapped(
    ctx: Context<CompleteTransferWithPayloadWrapped>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete. The account only exists to ensure that the VAA is not processed,
    // so this value does not matter. But the legacy program set this data to true.
    ctx.accounts.claim.is_complete = true;

    let vaa = PostedVaaV1::parse(&ctx.accounts.posted_vaa).unwrap();

    // Take transfer amount as-is.
    let mint_amount = TokenBridgeMessage::parse(vaa.payload())
        .unwrap()
        .transfer_with_message()
        .unwrap()
        .encoded_amount()
        .0
        .try_into()
        .map_err(|_| TokenBridgeError::U64Overflow)?;

    // Finally transfer encoded amount by minting to the redeemer's token account.
    utils::cpi::mint_to(
        ctx.accounts,
        ctx.accounts.dst_token.to_account_info(),
        mint_amount,
        Some(&[&[MINT_AUTHORITY_SEED_PREFIX, &[ctx.bumps["mint_authority"]]]]),
    )
}
