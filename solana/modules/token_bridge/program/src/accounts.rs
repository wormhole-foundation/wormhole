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

/// This method removes code duplication when checking token metadata. When metadata is read for
/// attestation and transfers, Token Bridge does not invoke Metaplex's Token Metadata program, so
/// it must validate the account the same way Token Metadata program does to ensure the correct
/// account is passed into Token Bridge's instruction context.
pub fn deserialize_and_verify_metadata(
    info: &Info,
    derivation_data: SplTokenMetaDerivationData,
) -> Result<spl_token_metadata::state::Metadata> {
    // Verify pda.
    info.verify_derivation(&spl_token_metadata::id(), &derivation_data)?;

    // There must be account data for token's metadata.
    if info.data_is_empty() {
        return Err(TokenBridgeError::NonexistentTokenMetadataAccount.into());
    }

    // Account must belong to Metaplex Token Metadata program.
    if *info.owner != spl_token_metadata::id() {
        return Err(TokenBridgeError::WrongAccountOwner.into());
    }

    // Account must be the expected Metadata length.
    if info.data_len() != spl_token_metadata::state::MAX_METADATA_LEN {
        return Err(TokenBridgeError::InvalidMetadata.into());
    }

    let mut data: &[u8] = &info.data.borrow_mut();

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
