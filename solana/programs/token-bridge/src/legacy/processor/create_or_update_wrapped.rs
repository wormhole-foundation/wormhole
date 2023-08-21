use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::EmptyArgs,
    state::{Claim, RegisteredEmitter, WrappedAsset},
};
use anchor_lang::prelude::*;
use anchor_spl::{metadata, token};
use core_bridge_program::{
    self,
    constants::SOLANA_CHAIN,
    state::{PartialPostedVaaV1, VaaV1Account},
    CoreBridge,
};
use mpl_token_metadata::state::DataV2;
use wormhole_raw_vaas::token_bridge::{Attestation, TokenBridgeMessage};
use wormhole_solana_common::SeedPrefix;

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
            PartialPostedVaaV1::SEED_PREFIX,
            posted_vaa.try_message_hash()?.as_ref()
        ],
        bump,
        seeds::program = core_bridge_program,
    )]
    posted_vaa: Account<'info, PartialPostedVaaV1>,

    #[account(
        init,
        payer = payer,
        space = Claim::INIT_SPACE,
        seeds = [
            posted_vaa.emitter_address.as_ref(),
            &posted_vaa.emitter_chain.to_be_bytes(),
            &posted_vaa.sequence.to_be_bytes(),
        ],
        bump,
    )]
    claim: Account<'info, Claim>,

    /// CHECK: To avoid multiple borrows to the posted vaa account to generate seeds and other mint
    /// parameters, we perform these checks outside of this accounts context. The pubkey for this
    /// wrapped mint is checked in access control and the account is created in the instruction
    /// handler.
    #[account(
        init_if_needed,
        payer = payer,
        mint::decimals = try_attestation_decimals(&posted_vaa)?,
        mint::authority = mint_authority,
        seeds = [
            WRAPPED_MINT_SEED_PREFIX,
            &try_attestation_token_chain(&posted_vaa)?.to_be_bytes(),
            try_attestation_token_address(&posted_vaa)?.as_ref(),
        ],
        bump,
    )]
    wrapped_mint: Box<Account<'info, token::Mint>>,

    #[account(
        init_if_needed,
        payer = payer,
        space = WrappedAsset::INIT_SPACE,
        seeds = [WrappedAsset::SEED_PREFIX, wrapped_mint.key().as_ref()],
        bump,
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
            wrapped_mint.key().as_ref(),
        ],
        bump,
        seeds::program = mpl_token_metadata_program,
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

    core_bridge_program: Program<'info, CoreBridge>,
    system_program: Program<'info, System>,
    token_program: Program<'info, token::Token>,
    mpl_token_metadata_program: Program<'info, metadata::Metadata>,
}

fn try_attestation_decimals(posted_vaa: &Account<'_, PartialPostedVaaV1>) -> Result<u8> {
    let acc_info: &AccountInfo = posted_vaa.as_ref();
    let data = &acc_info.try_borrow_data()?[PartialPostedVaaV1::PAYLOAD_START..];
    let msg =
        TokenBridgeMessage::parse(data).map_err(|_| TokenBridgeError::InvalidTokenBridgePayload)?;
    match msg.attestation() {
        Some(attestation) => Ok(cap_decimals(attestation.decimals())),
        None => err!(TokenBridgeError::InvalidTokenBridgeVaa),
    }
}

fn try_attestation_token_chain(posted_vaa: &Account<'_, PartialPostedVaaV1>) -> Result<u16> {
    let acc_info: &AccountInfo = posted_vaa.as_ref();
    let data = &acc_info.try_borrow_data()?[PartialPostedVaaV1::PAYLOAD_START..];
    let msg =
        TokenBridgeMessage::parse(data).map_err(|_| TokenBridgeError::InvalidTokenBridgePayload)?;
    match msg.attestation() {
        Some(attestation) => {
            let token_chain = attestation.token_chain();

            // This token must have originated from another network.
            require_neq!(token_chain, SOLANA_CHAIN, TokenBridgeError::NativeAsset);

            // Done.
            Ok(attestation.token_chain())
        }
        None => err!(TokenBridgeError::InvalidTokenBridgeVaa),
    }
}

fn try_attestation_token_address(posted_vaa: &Account<'_, PartialPostedVaaV1>) -> Result<[u8; 32]> {
    let acc_info: &AccountInfo = posted_vaa.as_ref();
    let data = &acc_info.try_borrow_data()?[PartialPostedVaaV1::PAYLOAD_START..];
    let msg =
        TokenBridgeMessage::parse(data).map_err(|_| TokenBridgeError::InvalidTokenBridgePayload)?;
    match msg.attestation() {
        Some(attestation) => Ok(attestation.token_address()),
        None => err!(TokenBridgeError::InvalidTokenBridgeVaa),
    }
}

impl<'info> CreateOrUpdateWrapped<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = &ctx.accounts.posted_vaa;

        let acc_info: &AccountInfo = vaa.as_ref();
        let acc_data = &acc_info.try_borrow_data()?;
        crate::utils::require_valid_token_bridge_posted_vaa(
            vaa.details(),
            acc_data,
            &ctx.accounts.registered_emitter,
        )?;

        // NOTE: Other attestation validation is performed using the try_attestation_* methods,
        // which were used in the accounts context.

        // Done.
        Ok(())
    }
}

#[access_control(CreateOrUpdateWrapped::constraints(&ctx))]
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
    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let acc_data = &acc_info.data.borrow();

    let msg = crate::utils::parse_token_bridge_message(acc_data).unwrap();
    let attestation = msg.attestation().unwrap();

    let (symbol, name) = fix_symbol_and_name(attestation);
    let token_chain = attestation.token_chain();
    let token_address = attestation.token_address();
    let native_decimals = attestation.decimals();

    // Set wrapped asset data.
    let wrapped_asset = &mut ctx.accounts.wrapped_asset;
    wrapped_asset.set_inner(WrappedAsset {
        token_chain,
        token_address,
        native_decimals,
    });

    // The wrapped asset account data will be encoded as JSON in the token metadata's URI.
    let uri = wrapped_asset
        .to_uri()
        .map_err(|_| TokenBridgeError::CannotSerializeJson)?;

    metadata::create_metadata_accounts_v3(
        CpiContext::new_with_signer(
            ctx.accounts.mpl_token_metadata_program.to_account_info(),
            metadata::CreateMetadataAccountsV3 {
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
    let acc_info: &AccountInfo = ctx.accounts.posted_vaa.as_ref();
    let data = &acc_info.data.borrow()[PartialPostedVaaV1::PAYLOAD_START..];
    let msg = TokenBridgeMessage::parse(data).unwrap();
    let attestation = msg.attestation().unwrap();
    let (symbol, name) = fix_symbol_and_name(attestation);

    // Deserialize token metadata so we can check whether the name or symbol have changed in
    // this asset metadata VAA.
    let data = {
        let mut acc_data: &[u8] = &ctx.accounts.token_metadata.try_borrow_data()?;
        metadata::MetadataAccount::try_deserialize(&mut acc_data).map(|acct| acct.data.clone())?
    };

    if name != data.name || symbol != data.symbol {
        // Finally update token metadata.
        metadata::update_metadata_accounts_v2(
            CpiContext::new_with_signer(
                ctx.accounts.mpl_token_metadata_program.to_account_info(),
                metadata::UpdateMetadataAccountsV2 {
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
    let mut symbol = attestation.symbol().to_string();
    symbol.truncate(mpl_token_metadata::state::MAX_SYMBOL_LENGTH);

    (symbol, attestation.name().to_string())
}
