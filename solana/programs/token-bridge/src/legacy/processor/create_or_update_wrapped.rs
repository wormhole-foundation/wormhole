use crate::{
    constants::{MAX_DECIMALS, MINT_AUTHORITY_SEED_PREFIX, WRAPPED_MINT_SEED_PREFIX},
    error::TokenBridgeError,
    legacy::instruction::EmptyArgs,
    state::{LegacyWrappedAsset, RegisteredEmitter, WrappedAsset},
};
use anchor_lang::{prelude::*, system_program};
use anchor_spl::{metadata, token};
use core_bridge_program::sdk as core_bridge;
use mpl_token_metadata::state::DataV2;
use wormhole_raw_vaas::token_bridge::{Attestation, TokenBridgeMessage};

#[derive(Accounts)]
pub struct CreateOrUpdateWrapped<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    /// Previously needed config account.
    ///
    /// CHECK: This account is unchecked.
    _config: UncheckedAccount<'info>,

    /// This account is a foreign token Bridge and is created via the Register Chain governance
    /// decree.
    ///
    /// NOTE: The seeds of this account are insane because they include the emitter address, which
    /// allows registering multiple emitter addresses for the same chain ID. These seeds are not
    /// checked via Anchor macro, but will be checked in the access control function instead.
    ///
    /// See the `require_valid_token_bridge_vaa` instruction handler for more details.
    registered_emitter: Account<'info, core_bridge::legacy::LegacyAnchorized<RegisteredEmitter>>,

    /// VAA account, which may either be the new EncodedVaa account or legacy PostedVaaV1
    /// account.
    ///
    /// CHECK: This account will be read via zero-copy deserialization in the instruction
    /// handler, which will determine which type of VAA account is being used. If this account
    /// is the legacy PostedVaaV1 account, its PDA address will be verified by this zero-copy
    /// reader.
    #[account(
        owner = core_bridge::id(),
        constraint = try_attestation(&vaa, |attestation| attestation.token_chain())?
                        != core_bridge_program::SOLANA_CHAIN @ TokenBridgeError::NativeAsset,
    )]
    vaa: AccountInfo<'info>,

    /// Claim account (mut), which acts as replay protection after consuming data from the VAA
    /// account.
    ///
    /// Seeds: [emitter_address, emitter_chain, sequence],
    /// seeds::program = token_bridge_program.
    ///
    /// CHECK: This account is created via [claim_vaa](core_bridge_program::sdk::claim_vaa).
    /// This account can only be created once for this VAA.
    #[account(mut)]
    claim: AccountInfo<'info>,

    /// CHECK: To avoid multiple borrows to the posted vaa account to generate seeds and other mint
    /// parameters, we perform these checks outside of this accounts context. The pubkey for this
    /// wrapped mint is checked in access control and the account is created in the instruction
    /// handler.
    #[account(
        init_if_needed,
        payer = payer,
        mint::decimals = try_attestation(&vaa, |att| cap_decimals(att.decimals()))?,
        mint::authority = mint_authority,
        seeds = [
            WRAPPED_MINT_SEED_PREFIX,
            try_attestation(&vaa, |att| att.token_chain())?.to_be_bytes().as_ref(),
            try_attestation(&vaa, |att| att.token_address())?.as_ref(),
        ],
        bump,
    )]
    wrapped_mint: Box<Account<'info, token::Mint>>,

    /// CHECK: Wrapped asset. This account will either be created if it does not exist or its size
    /// be reallocated in case this account if this account uses the old schema. In the old schema,
    /// there was no data reflecting the last VAA sequence number used, which can lead to metadata
    /// being overwritten by a stale VAA.
    ///
    /// NOTE: Because this account needs special handling via realloc, we cannot use the
    /// `init_if_needed` macro here.
    #[account(
        mut,
        seeds = [
            WrappedAsset::SEED_PREFIX,
            wrapped_mint.key().as_ref(),
        ],
        bump,
    )]
    wrapped_asset: AccountInfo<'info>,

    /// CHECK: This account is managed by the MPL Token Metadata program. We verify this PDA to
    /// ensure that we deserialize the correct metadata before creating or updating.
    ///
    /// NOTE: We do not actually have to re-derive this PDA address because the MPL program should
    /// perform this check anyway. But we are being extra safe here.
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

    /// Previously needed sysvar.
    ///
    /// CHECK: This account is unchecked.
    _rent: UncheckedAccount<'info>,

    system_program: Program<'info, System>,
    token_program: Program<'info, token::Token>,
    mpl_token_metadata_program: Program<'info, metadata::Metadata>,
}

impl<'info> core_bridge::legacy::ProcessLegacyInstruction<'info, EmptyArgs>
    for CreateOrUpdateWrapped<'info>
{
    const LOG_IX_NAME: &'static str = "LegacyCreateOrUpdateWrapped";

    const ANCHOR_IX_FN: fn(Context<Self>, EmptyArgs) -> Result<()> = create_or_update_wrapped;
}

impl<'info> CreateOrUpdateWrapped<'info> {
    fn constraints(ctx: &Context<Self>) -> Result<()> {
        let vaa = &ctx.accounts.vaa;

        // NOTE: Other attestation validation is performed using the try_attestation_* methods,
        // which were used in the accounts context.
        crate::utils::vaa::require_valid_token_bridge_vaa(
            &vaa.key(),
            &core_bridge::VaaAccount::load(vaa).unwrap(),
            &ctx.accounts.registered_emitter,
        )?;

        // Done.
        Ok(())
    }
}

#[access_control(CreateOrUpdateWrapped::constraints(&ctx))]
fn create_or_update_wrapped(ctx: Context<CreateOrUpdateWrapped>, _args: EmptyArgs) -> Result<()> {
    // Check if token metadata has been created yet. If it isn't, we must create this account and
    // the wrapped asset account.
    if ctx.accounts.token_metadata.data_is_empty() {
        handle_create_wrapped(ctx)
    } else {
        handle_update_wrapped(ctx)
    }
}

fn handle_create_wrapped(ctx: Context<CreateOrUpdateWrapped>) -> Result<()> {
    let vaa = core_bridge::VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    core_bridge::claim_vaa(
        CpiContext::new(
            ctx.accounts.system_program.to_account_info(),
            core_bridge::ClaimVaa {
                claim: ctx.accounts.claim.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
            },
        ),
        &crate::ID,
        &vaa,
        None,
    )?;

    let msg = TokenBridgeMessage::try_from(vaa.try_payload().unwrap()).unwrap();
    let attestation = msg.attestation().unwrap();

    let core_bridge::EmitterInfo {
        chain: _,
        address: _,
        sequence,
    } = vaa.try_emitter_info().unwrap();
    let wrapped_asset = WrappedAsset {
        legacy: LegacyWrappedAsset {
            token_chain: attestation.token_chain(),
            token_address: attestation.token_address(),
            native_decimals: attestation.decimals(),
        },
        last_updated_sequence: sequence,
    };

    // The wrapped asset account data will be encoded as JSON in the token metadata's URI.
    let uri = wrapped_asset.to_uri();

    // Create and set wrapped asset data.
    {
        core_bridge::system_program::create_account_safe(
            CpiContext::new_with_signer(
                ctx.accounts.system_program.to_account_info(),
                core_bridge::system_program::CreateAccountSafe {
                    new_account: ctx.accounts.wrapped_asset.to_account_info(),
                    payer: ctx.accounts.payer.to_account_info(),
                },
                &[&[
                    WrappedAsset::SEED_PREFIX,
                    ctx.accounts.wrapped_mint.key().as_ref(),
                    &[ctx.bumps["wrapped_asset"]],
                ]],
            ),
            WrappedAsset::INIT_SPACE,
            &crate::ID,
        )?;

        let acc_data: &mut [_] = &mut ctx.accounts.wrapped_asset.data.borrow_mut();
        let mut writer = std::io::Cursor::new(acc_data);
        core_bridge::legacy::LegacyAnchorized::from(wrapped_asset).try_serialize(&mut writer)?;
    }

    let FixedMeta { symbol, name } = fix_symbol_and_name(attestation);

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
                rent: ctx.accounts.mpl_token_metadata_program.to_account_info(), // optional rent
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
    let vaa = core_bridge::VaaAccount::load(&ctx.accounts.vaa).unwrap();

    // Create the claim account to provide replay protection. Because this instruction creates this
    // account every time it is executed, this account cannot be created again with this emitter
    // address, chain and sequence combination.
    core_bridge::claim_vaa(
        CpiContext::new(
            ctx.accounts.system_program.to_account_info(),
            core_bridge::ClaimVaa {
                claim: ctx.accounts.claim.to_account_info(),
                payer: ctx.accounts.payer.to_account_info(),
            },
        ),
        &crate::ID,
        &vaa,
        None,
    )?;

    let msg = TokenBridgeMessage::try_from(vaa.try_payload().unwrap()).unwrap();
    let attestation = msg.attestation().unwrap();

    // For wrapped assets created before this implementation, the wrapped asset schema did not
    // include a VAA sequence number, which prevents metadata attestations to be redeemed out-of-
    // order. For example, if a VAA with metadata name = "A" were never redeemed and then the name
    // changed to "B", someone would have been able to redeem name = "B" and then overwrite the name
    // with "A" by redeeming the old VAA.
    //
    // Here we need to check whether the wrapped asset is the old schema. If it is, we need to
    // increase the size of the account by 8 bytes to account for the sequence.
    if ctx.accounts.wrapped_asset.data_len() == LegacyWrappedAsset::INIT_SPACE {
        let acc_info = &ctx.accounts.wrapped_asset;

        let lamports_diff = Rent::get().map(|rent| {
            rent.minimum_balance(WrappedAsset::INIT_SPACE)
                .saturating_sub(acc_info.lamports())
        })?;

        system_program::transfer(
            CpiContext::new(
                ctx.accounts.system_program.to_account_info(),
                system_program::Transfer {
                    from: ctx.accounts.payer.to_account_info(),
                    to: acc_info.to_account_info(),
                },
            ),
            lamports_diff,
        )?;

        acc_info.realloc(WrappedAsset::INIT_SPACE, false)?;
    }

    // Now check the sequence to see whether this VAA is stale.
    let core_bridge::EmitterInfo {
        chain: _,
        address: _,
        sequence: updated_sequence,
    } = vaa.try_emitter_info().unwrap();
    let wrapped_asset = {
        let acc_data = ctx.accounts.wrapped_asset.data.borrow();
        let mut wrapped_asset =
            core_bridge::legacy::LegacyAnchorized::<WrappedAsset>::try_deserialize(
                &mut acc_data.as_ref(),
            )?;
        require!(
            updated_sequence > wrapped_asset.last_updated_sequence,
            TokenBridgeError::AttestationOutOfSequence
        );

        // Modify this wrapped asset to prepare it for writing.
        wrapped_asset.last_updated_sequence = updated_sequence;

        wrapped_asset
    };

    // Update wrapped asset.
    {
        let acc_data: &mut [_] = &mut ctx.accounts.wrapped_asset.data.borrow_mut();
        let mut writer = std::io::Cursor::new(acc_data);
        wrapped_asset.try_serialize(&mut writer)?;
    }

    // Deserialize token metadata so we can check whether the name or symbol have changed in
    // this asset metadata VAA.
    let data = {
        metadata::MetadataAccount::try_deserialize(
            &mut ctx.accounts.token_metadata.data.borrow().as_ref(),
        )
        .map(|meta| meta.data.clone())?
    };

    let FixedMeta { symbol, name } = fix_symbol_and_name(attestation);

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

fn try_attestation<F, T>(vaa_acc_info: &AccountInfo, func: F) -> Result<T>
where
    F: FnOnce(&Attestation) -> T,
{
    let vaa = core_bridge::VaaAccount::load(vaa_acc_info)?;
    let msg = TokenBridgeMessage::try_from(vaa.try_payload()?)
        .map_err(|_| TokenBridgeError::InvalidTokenBridgePayload)?;
    msg.attestation()
        .map(func)
        .ok_or(error!(TokenBridgeError::InvalidTokenBridgeVaa))
}

fn cap_decimals(decimals: u8) -> u8 {
    std::cmp::min(decimals, MAX_DECIMALS)
}

struct FixedMeta {
    symbol: String,
    name: String,
}

fn fix_symbol_and_name(attestation: &Attestation) -> FixedMeta {
    // Truncate symbol to 10 characters (the maximum length for Token Metadata's symbol).
    let mut symbol = attestation.symbol().to_string();
    symbol.truncate(mpl_token_metadata::state::MAX_SYMBOL_LENGTH);

    FixedMeta {
        symbol,
        name: attestation.name().to_string(),
    }
}
