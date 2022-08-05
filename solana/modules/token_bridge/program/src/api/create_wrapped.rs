use crate::{
    accounts::{
        ConfigAccount,
        Endpoint,
        EndpointDerivationData,
        MintSigner,
        SplTokenMeta,
        SplTokenMetaDerivationData,
        WrappedDerivationData,
        WrappedMetaDerivationData,
        WrappedMint,
        WrappedTokenMeta,
    },
    messages::PayloadAssetMeta,
    TokenBridgeError::{
        InvalidChain,
        InvalidMetadata,
        InvalidVAA,
    },
    INVALID_VAAS,
};
use bridge::{
    accounts::claim::{
        self,
        Claim,
    },
    PayloadMessage,
    CHAIN_ID_SOLANA,
};
use solana_program::{
    account_info::AccountInfo,
    program::invoke_signed,
};
use solitaire::{
    processors::seeded::{
        invoke_seeded,
        Seeded,
    },
    CreationLamports::Exempt,
    *,
};

use spl_token_metadata::state::{
    Data as SplData,
    Metadata,
};
use std::cmp::min;

#[derive(FromAccounts)]
pub struct CreateWrapped<'b> {
    pub payer: Mut<Signer<AccountInfo<'b>>>,
    pub config: ConfigAccount<'b, { AccountState::Initialized }>,

    pub chain_registration: Endpoint<'b, { AccountState::Initialized }>,
    pub vaa: PayloadMessage<'b, PayloadAssetMeta>,
    pub claim: Mut<Claim<'b>>,

    // New Wrapped
    pub mint: Mut<WrappedMint<'b, { AccountState::MaybeInitialized }>>,
    pub meta: Mut<WrappedTokenMeta<'b, { AccountState::MaybeInitialized }>>,

    /// SPL Metadata for the associated Mint
    pub spl_metadata: Mut<SplTokenMeta<'b>>,

    pub mint_authority: MintSigner<'b>,
}

impl<'a> From<&CreateWrapped<'a>> for EndpointDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        EndpointDerivationData {
            emitter_chain: accs.vaa.meta().emitter_chain,
            emitter_address: accs.vaa.meta().emitter_address,
        }
    }
}

impl<'a> From<&CreateWrapped<'a>> for WrappedDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        WrappedDerivationData {
            token_chain: accs.vaa.token_chain,
            token_address: accs.vaa.token_address,
        }
    }
}

impl<'a> From<&CreateWrapped<'a>> for WrappedMetaDerivationData {
    fn from(accs: &CreateWrapped<'a>) -> Self {
        WrappedMetaDerivationData {
            mint_key: *accs.mint.info().key,
        }
    }
}

#[derive(BorshDeserialize, BorshSerialize, Default)]
pub struct CreateWrappedData {}

pub fn create_wrapped(
    ctx: &ExecutionContext,
    accs: &mut CreateWrapped,
    data: CreateWrappedData,
) -> Result<()> {
    // Do not process attestations sourced from the current chain.
    if accs.vaa.token_chain == CHAIN_ID_SOLANA {
        return Err(InvalidChain.into());
    }

    let derivation_data: WrappedDerivationData = (&*accs).into();
    accs.mint
        .verify_derivation(ctx.program_id, &derivation_data)?;

    let meta_derivation_data: WrappedMetaDerivationData = (&*accs).into();
    accs.meta
        .verify_derivation(ctx.program_id, &meta_derivation_data)?;

    let derivation_data: EndpointDerivationData = (&*accs).into();
    accs.chain_registration
        .verify_derivation(ctx.program_id, &derivation_data)?;

    if INVALID_VAAS.contains(&&*accs.vaa.info().key.to_string()) {
        return Err(InvalidVAA.into());
    }

    claim::consume(ctx, accs.payer.key, &mut accs.claim, &accs.vaa)?;

    if accs.mint.is_initialized() {
        update_accounts(ctx, accs, data)
    } else {
        create_accounts(ctx, accs, data)
    }
}

pub fn create_accounts(
    ctx: &ExecutionContext,
    accs: &mut CreateWrapped,
    _data: CreateWrappedData,
) -> Result<()> {
    // Create mint account
    accs.mint
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

    // Initialize mint
    let init_ix = spl_token::instruction::initialize_mint(
        &spl_token::id(),
        accs.mint.info().key,
        accs.mint_authority.key,
        None,
        min(8, accs.vaa.decimals), // Limit to 8 decimals, truncation is handled on the other side
    )?;
    invoke_signed(&init_ix, ctx.accounts, &[])?;

    // Create meta account
    accs.meta
        .create(&((&*accs).into()), ctx, accs.payer.key, Exempt)?;

    // Initialize spl meta
    accs.spl_metadata.verify_derivation(
        &spl_token_metadata::id(),
        &SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        },
    )?;

    // Normalize Token Metadata.
    let name = truncate_utf8(&accs.vaa.name, 32 - 11) + " (Wormhole)";
    let symbol = truncate_utf8(&accs.vaa.symbol, 10);

    let spl_token_metadata_ix = spl_token_metadata::instruction::create_metadata_accounts(
        spl_token_metadata::id(),
        *accs.spl_metadata.key,
        *accs.mint.info().key,
        *accs.mint_authority.info().key,
        *accs.payer.info().key,
        *accs.mint_authority.info().key,
        name,
        symbol,
        String::from(""),
        None,
        0,
        false,
        true,
    );
    invoke_seeded(&spl_token_metadata_ix, ctx, &accs.mint_authority, None)?;

    // Populate meta account
    accs.meta.chain = accs.vaa.token_chain;
    accs.meta.token_address = accs.vaa.token_address;
    accs.meta.original_decimals = accs.vaa.decimals;

    Ok(())
}

pub fn update_accounts(
    ctx: &ExecutionContext,
    accs: &mut CreateWrapped,
    _data: CreateWrappedData,
) -> Result<()> {
    accs.spl_metadata.verify_derivation(
        &spl_token_metadata::id(),
        &SplTokenMetaDerivationData {
            mint: *accs.mint.info().key,
        },
    )?;

    let mut metadata: SplData = Metadata::from_account_info(accs.spl_metadata.info())
        .ok_or(InvalidMetadata)?
        .data;

    // Normalize token metadata.
    metadata.name = truncate_utf8(&accs.vaa.name, 32 - 11) + " (Wormhole)";
    metadata.symbol = truncate_utf8(&accs.vaa.symbol, 10);

    // Update SPL Metadata
    let spl_token_metadata_ix = spl_token_metadata::instruction::update_metadata_accounts(
        spl_token_metadata::id(),
        *accs.spl_metadata.key,
        *accs.mint_authority.info().key,
        None,
        Some(metadata),
        None,
    );
    invoke_seeded(&spl_token_metadata_ix, ctx, &accs.mint_authority, None)?;

    Ok(())
}

// Byte-truncates potentially invalid UTF-8 encoded strings by converting to Unicode codepoints and
// stripping unrecognised characters.
pub fn truncate_utf8(data: impl AsRef<[u8]>, len: usize) -> String {
    use bstr::ByteSlice;
    let mut data = data.as_ref().to_vec();
    data.truncate(len);
    let mut data: Vec<char> = data.chars().collect();
    data.retain(|&c| c != '\u{FFFD}');
    data.iter().collect()
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_unicode_truncation() {
        #[rustfmt::skip]
        let pairs = [
            // Empty string should not error or mutate.
            (
                "",
                ""
            ),
            // Unicode < 32 should not be corrupted.
            (
                "🔥",
                "🔥"
            ),
            // Unicode @ 32 should not be corrupted.
            (
                "🔥🔥🔥🔥🔥🔥🔥🔥",
                "🔥🔥🔥🔥🔥🔥🔥🔥"
            ),
            // Unicode > 32 should be truncated correctly.
            (
                "🔥🔥🔥🔥🔥🔥🔥🔥🔥",
                "🔥🔥🔥🔥🔥🔥🔥🔥"
            ),
            // Partially overflowing Unicode > 32 should be removed.
            // Note: Expecting 31 bytes.
            (
                "0000000000000000000000000000000🔥",
                "0000000000000000000000000000000"
            ),
        ];

        for (input, expected) in pairs {
            assert_eq!(expected, super::truncate_utf8(input, 32));
        }
    }
}
