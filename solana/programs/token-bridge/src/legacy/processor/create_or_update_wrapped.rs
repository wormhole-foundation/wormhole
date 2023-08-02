use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::EmptyArgs,
    state::{Claim, RegisteredEmitter, WrappedAsset},
};
use anchor_lang::prelude::*;
use anchor_spl::{
    metadata::{
        self as token_metadata, CreateMetadataAccountsV3, Metadata as MplTokenMetadata,
        MetadataAccount, UpdateMetadataAccountsV2,
    },
    token::{Mint, Token},
};
use core_bridge_program::{
    self,
    constants::SOLANA_CHAIN,
    state::{PostedVaaV1, VaaV1MessageHash},
};
use mpl_token_metadata::state::DataV2;
use wormhole_solana_common::SeedPrefix;
use wormhole_vaas::payloads::token_bridge::Attestation;

#[derive(Accounts)]
pub struct CreateOrUpdateWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _config: UncheckedAccount<'info>,

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
        seeds = [
            PostedVaaV1::<Attestation>::seed_prefix(),
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump
    )]
    posted_vaa: Account<'info, PostedVaaV1<Attestation>>,

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

    #[account(
        init_if_needed,
        payer = payer,
        mint::decimals = cap_decimals(posted_vaa.payload.decimals),
        mint::authority = mint_authority,
        seeds = [
            WRAPPED_MINT_SEED_PREFIX,
            &posted_vaa.payload.token_chain.to_be_bytes(),
            posted_vaa.payload.token_address.as_ref()
        ],
        bump
    )]
    wrapped_mint: Box<Account<'info, Mint>>,

    #[account(
        init_if_needed,
        payer = payer,
        space = WrappedAsset::INIT_SPACE,
        seeds = [wrapped_mint.key().as_ref()],
        bump
    )]
    wrapped_asset: Account<'info, WrappedAsset>,

    /// CHECK: This account is managed by the MPL Token Metadata program. But we still want to
    /// verify the PDA address because we will deserialize this account once it exists to determine
    /// whether we need to update metadata based on the new VAA (before passing this account into
    /// the update metadata instruction).
    #[account(
        mut,
        seeds = [
            b"metadata",
            mpl_token_metadata_program.key().as_ref(),
            wrapped_mint.key().as_ref()
        ],
        bump,
        seeds::program = mpl_token_metadata_program
    )]
    token_metadata: AccountInfo<'info>,

    /// CHECK: This account is the authority that can burn and mint wrapped assets.
    #[account(
        seeds = [MINT_AUTHORITY_SEED_PREFIX],
        bump,
    )]
    mint_authority: AccountInfo<'info>,

    /// CHECK: Rent is needed for the MPL Token Metadata program.
    rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,

    /// CHECK: Token Bridge never needed this account for this instruction.
    _core_bridge_program: UncheckedAccount<'info>,

    token_program: Program<'info, Token>,
    mpl_token_metadata_program: Program<'info, MplTokenMetadata>,
}

impl<'info> CreateOrUpdateWrapped<'info> {
    fn accounts(ctx: &Context<Self>) -> Result<()> {
        let attestation = crate::utils::require_valid_token_bridge_posted_vaa(
            &ctx.accounts.posted_vaa,
            &ctx.accounts.registered_emitter,
        )?;

        // Mint account must agree with the encoded token address.
        require_eq!(
            Pubkey::from(attestation.token_address.0),
            ctx.accounts.wrapped_mint.key()
        );

        // For wrapped transfers, this token must have originated from another network.
        require_neq!(
            attestation.token_chain,
            SOLANA_CHAIN,
            TokenBridgeError::NativeAsset
        );

        // Done.
        Ok(())
    }
}

#[access_control(CreateOrUpdateWrapped::accounts(&ctx))]
pub fn create_or_update_wrapped(
    ctx: Context<CreateOrUpdateWrapped>,
    _args: EmptyArgs,
) -> Result<()> {
    // Mark the claim as complete.
    ctx.accounts.claim.is_complete = true;

    // Check if token metadata has been created yet. If it isn't, we must create this account and
    // the wrapped asset account.
    if ctx.accounts.token_metadata.data_is_empty() {
        handle_create_wrapped(ctx)
    } else {
        handle_update_wrapped(ctx)
    }
}

fn handle_create_wrapped(ctx: Context<CreateOrUpdateWrapped>) -> Result<()> {
    let attestation = &ctx.accounts.posted_vaa.payload;
    let (symbol, name) = fix_symbol_and_name(attestation);

    let wrapped_asset = &mut ctx.accounts.wrapped_asset;
    wrapped_asset.set_inner(WrappedAsset {
        token_chain: attestation.token_chain,
        token_address: attestation.token_address.0,
        native_decimals: attestation.decimals,
    });

    // The wrapped asset account data will be encoded as JSON in the token metadata's URI.
    let uri = wrapped_asset
        .to_uri()
        .map_err(|_| TokenBridgeError::CannotSerializeJson)?;

    token_metadata::create_metadata_accounts_v3(
        CpiContext::new_with_signer(
            ctx.accounts.mpl_token_metadata_program.to_account_info(),
            CreateMetadataAccountsV3 {
                metadata: ctx.accounts.token_metadata.to_account_info(),
                mint: ctx.accounts.wrapped_mint.to_account_info(),
                mint_authority: ctx.accounts.mint_authority.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
                update_authority: ctx.accounts.mint_authority.to_account_info(),
                system_program: ctx.accounts.system_program.to_account_info(),
                rent: ctx.accounts.rent.to_account_info(),
            },
            &[&[MINT_AUTHORITY_SEED_PREFIX, &[ctx.bumps["mint_authority"]]]],
        ),
        DataV2 {
            symbol,
            name,
            uri,
            seller_fee_basis_points: 0,
            creators: None,
            collection: None,
            uses: None,
        },
        true,
        true,
        None,
    )
}

fn handle_update_wrapped(ctx: Context<CreateOrUpdateWrapped>) -> Result<()> {
    let (symbol, name) = fix_symbol_and_name(&ctx.accounts.posted_vaa.payload);

    // Deserialize token metadata so we can check whether the name or symbol have changed in
    // this asset metadata VAA.
    let data = {
        let mut acct_data: &[u8] = &ctx.accounts.token_metadata.try_borrow_data()?;
        MetadataAccount::try_deserialize(&mut acct_data).map(|acct| acct.data.clone())?
    };

    if name != data.name || symbol != data.symbol {
        // Finally update token metadata.
        token_metadata::update_metadata_accounts_v2(
            CpiContext::new_with_signer(
                ctx.accounts.mpl_token_metadata_program.to_account_info(),
                UpdateMetadataAccountsV2 {
                    metadata: ctx.accounts.token_metadata.to_account_info(),
                    update_authority: ctx.accounts.mint_authority.to_account_info(),
                },
                &[&[MINT_AUTHORITY_SEED_PREFIX, &[ctx.bumps["mint_authority"]]]],
            ),
            None,
            Some(DataV2 {
                symbol,
                name,
                uri: data.uri,
                seller_fee_basis_points: 0,
                creators: None,
                collection: None,
                uses: None,
            }),
            None,
            None,
        )
    } else {
        Ok(())
    }
}

fn cap_decimals(decimals: u8) -> u8 {
    if decimals > MAX_DECIMALS {
        MAX_DECIMALS
    } else {
        decimals
    }
}

fn fix_symbol_and_name(attestation: &Attestation) -> (String, String) {
    // Truncate symbol to 10 characters (the maximum length for Token Metadata's symbol).
    let mut symbol = attestation.symbol_string();
    symbol.truncate(mpl_token_metadata::state::MAX_SYMBOL_LENGTH);

    (symbol, attestation.name_string())
}
