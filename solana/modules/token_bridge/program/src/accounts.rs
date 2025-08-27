use crate::{
    types::*,
    TokenBridgeError,
};
use bridge::{
    accounts::BridgeData,
    api::ForeignAddress,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    *,
};
use spl_token_metadata::state::Key::MetadataV1;

pub type AuthoritySigner<'b> = Derive<Info<'b>, "authority_signer">;
pub type CustodySigner<'b> = Derive<Info<'b>, "custody_signer">;
pub type MintSigner<'b> = Derive<Info<'b>, "mint_signer">;

pub type CoreBridge<'a, const STATE: AccountState> = Data<'a, BridgeData, { STATE }>;

pub type EmitterAccount<'b> = Derive<Info<'b>, "emitter">;

pub type ConfigAccount<'b, const STATE: AccountState> =
    Derive<Data<'b, Config, { STATE }>, "config">;

pub type CustodyAccount<'b, const STATE: AccountState> = Data<'b, SplAccount, { STATE }>;

pub struct CustodyAccountDerivationData {
    pub mint: Pubkey,
}

impl<'b, const STATE: AccountState> Seeded<&CustodyAccountDerivationData>
    for CustodyAccount<'b, { STATE }>
{
    fn seeds(accs: &CustodyAccountDerivationData) -> Vec<Vec<u8>> {
        vec![accs.mint.to_bytes().to_vec()]
    }
}

pub type WrappedMint<'b, const STATE: AccountState> = Data<'b, SplMint, { STATE }>;

pub struct WrappedDerivationData {
    pub token_chain: ChainID,
    pub token_address: ForeignAddress,
}

impl<'b, const STATE: AccountState> Seeded<&WrappedDerivationData> for WrappedMint<'b, { STATE }> {
    fn seeds(data: &WrappedDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("wrapped").as_bytes().to_vec(),
            data.token_chain.to_be_bytes().to_vec(),
            data.token_address.to_vec(),
        ]
    }
}

pub type WrappedTokenMeta<'b, const STATE: AccountState> = Data<'b, WrappedMeta, { STATE }>;

pub struct WrappedMetaDerivationData {
    pub mint_key: Pubkey,
}

impl<'b, const STATE: AccountState> Seeded<&WrappedMetaDerivationData>
    for WrappedTokenMeta<'b, { STATE }>
{
    fn seeds(data: &WrappedMetaDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("meta").as_bytes().to_vec(),
            data.mint_key.to_bytes().to_vec(),
        ]
    }
}

/// Registered chain endpoint
pub type Endpoint<'b, const STATE: AccountState> = Data<'b, EndpointRegistration, { STATE }>;

pub struct EndpointDerivationData {
    pub emitter_chain: u16,
    pub emitter_address: ForeignAddress,
}

/// Seeded implementation based on an incoming VAA
impl<'b, const STATE: AccountState> Seeded<&EndpointDerivationData> for Endpoint<'b, { STATE }> {
    fn seeds(data: &EndpointDerivationData) -> Vec<Vec<u8>> {
        vec![
            data.emitter_chain.to_be_bytes().to_vec(),
            data.emitter_address.to_vec(),
        ]
    }
}

/// Metadata account for the token. This used to be exclusively the Metaplex
/// metadata account, but with token2022's metadata pointer extension, this may be any account.
/// `deserialize_and_verify_metadata` verifies that this account is what the token specifies (or falls back to Metaplex).
pub type SplTokenMeta<'b> = Info<'b>;

pub struct SplTokenMetaDerivationData {
    pub mint: Pubkey,
}

impl<'b> Seeded<&SplTokenMetaDerivationData> for SplTokenMeta<'b> {
    fn seeds(data: &SplTokenMetaDerivationData) -> Vec<Vec<u8>> {
        vec![
            "metadata".as_bytes().to_vec(),
            spl_token_metadata::id().as_ref().to_vec(),
            data.mint.as_ref().to_vec(),
        ]
    }
}

/// New data length of spl token metadata account (post https://developers.metaplex.com/token-metadata/guides/account-size-reduction)
/// The naming convention we adopt for future resizes is iterated NEW_* prefixes, e.g. NEW_NEW_NEW_NEW_NEW_MAX_METADATA_LEN.
/// This will continue until morale improves.
pub const NEW_MAX_METADATA_LEN: usize = 607;

/// Converts Token-2022 metadata to Metaplex metadata format for compatibility
fn convert_token2022_to_metaplex_metadata(
    token_metadata: &token_metadata_parser::TokenMetadata,
) -> spl_token_metadata::state::Metadata {
    use spl_token_metadata::state::{
        Data,
        Key,
        Metadata,
    };

    let data = Data {
        name: token_metadata.name.clone(),
        symbol: token_metadata.symbol.clone(),
        uri: token_metadata.uri.clone(),
        seller_fee_basis_points: 0, // Token-2022 doesn't have this concept
        creators: None,             // Token-2022 doesn't have creators
    };

    Metadata {
        key: Key::MetadataV1,
        update_authority: token_metadata
            .update_authority
            .map(|pubkey| Pubkey::new(&pubkey.0))
            .unwrap_or_default(),
        mint: Pubkey::new(&token_metadata.mint.0),
        data,
        primary_sale_happened: false, // Default for Token-2022
        is_mutable: token_metadata.update_authority.is_some(),
        edition_nonce: None,
        token_standard: None,
        collection: None,
        uses: None,
        collection_details: None,
        programmable_config: None,
    }
}

/// This method removes code duplication when checking token metadata. When metadata is read for
/// attestation and transfers, Token Bridge does not invoke Metaplex's Token Metadata program, so
/// it must validate the account the same way Token Metadata program does to ensure the correct
/// account is passed into Token Bridge's instruction context.
pub fn deserialize_and_verify_metadata(
    mint: &Info,
    metadata: &Info,
    derivation_data: SplTokenMetaDerivationData,
) -> Result<spl_token_metadata::state::Metadata> {
    let mint_metadata = token_metadata_parser::parse_token2022_metadata(
        token_metadata_parser::Pubkey::new(mint.key.to_bytes()),
        &mint.data.borrow(),
    )
    .map_err(|_| TokenBridgeError::InvalidMetadata)?;

    // we constrain the `metadata` account in every case.
    // 1. if mint is token-2022 with embedded metadata, we return that metadata (in this case, `metadata` == `mint`. `token_metadata_parser` ensures this)
    // 2. if mint is token-2022 with external metadata pointer, we verify `metadata` matches the pointer
    //    a. if `metadata` is owned by spl-token-metadata, we verify the pda and deserialise it as standard Metaplex metadata
    //    b. if `metadata` is not owned by spl-token-metadata, we don't verify that it's a pda (we know it matches the pointer already)
    // 3. if mint doesn't include a metadata pointer, we ensure `metadata` is the metaplex pda.
    //    this is the legacy case, but it applies to token2022 tokens as well (that have no metadata pointer extension)
    //
    // Note that in every case other than 1 (which is a well-defined spec via
    // the token metadata extension), we parse the `metadata` account following
    // the standard metaplex format.
    //
    // In case of 2b, this is a best-effort guess, because the metadata pointer
    // extension makes no guarantees about the shape of the metadata account. However, a common practice is to just follow the metaplex format.
    // What this means is that if the metadata account is not owned by the metaplex program, and is not in the metaplex format, the deserialisation will fail.
    // We just don't support these tokens.

    match mint_metadata {
        // 1.
        token_metadata_parser::MintMetadata::Embedded(token_metadata) => {
            // token-2022 mint with embedded metadata
            return Ok(convert_token2022_to_metaplex_metadata(&token_metadata));
        }
        // 2.
        token_metadata_parser::MintMetadata::External(pointer) => {
            if pointer.metadata_address
                != token_metadata_parser::Pubkey::new(metadata.key.to_bytes())
            {
                return Err(TokenBridgeError::WrongMetadataAccount.into());
            }

            // 2a.
            if *metadata.owner == spl_token_metadata::id() {
                // Standard Metaplex metadata verification and parsing
                // Verify pda.
                metadata.verify_derivation(&spl_token_metadata::id(), &derivation_data)?;
            // 2b.
            } else {
                // fall through
            }
        }
        // 3.
        token_metadata_parser::MintMetadata::None => {
            // Standard Metaplex metadata verification and parsing
            // Verify pda.
            metadata.verify_derivation(&spl_token_metadata::id(), &derivation_data)?;
        }
    }

    // There must be account data for token's metadata.
    if metadata.data_is_empty() {
        return Err(TokenBridgeError::NonexistentTokenMetadataAccount.into());
    }

    // Account must belong to Metaplex Token Metadata program.
    if *metadata.owner != spl_token_metadata::id() {
        return Err(TokenBridgeError::WrongAccountOwner.into());
    }

    // Account must be the expected Metadata length.
    if metadata.data_len() != spl_token_metadata::state::MAX_METADATA_LEN
        && metadata.data_len() != NEW_MAX_METADATA_LEN
    {
        return Err(TokenBridgeError::InvalidMetadata.into());
    }

    let mut data: &[u8] = &metadata.data.borrow_mut();

    // Unfortunately we cannot use `map_err` easily, so we will match certain deserialization conditions.
    match spl_token_metadata::utils::meta_deser_unchecked(&mut data) {
        Ok(deserialized) => {
            if deserialized.key == MetadataV1 {
                Ok(deserialized)
            } else {
                Err(TokenBridgeError::NotMetadataV1Account.into())
            }
        }
        _ => Err(TokenBridgeError::InvalidMetadata.into()),
    }
}
