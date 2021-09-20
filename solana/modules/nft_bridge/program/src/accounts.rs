use crate::types::*;
use bridge::{
    accounts::BridgeData,
    api::ForeignAddress,
};
use primitive_types::U256;
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    *,
};

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
    pub token_id: U256,
}

impl<'b, const STATE: AccountState> Seeded<&WrappedDerivationData> for WrappedMint<'b, { STATE }> {
    fn seeds(data: &WrappedDerivationData) -> Vec<Vec<u8>> {
        let mut token_id = vec![0u8; 32];
        data.token_id.to_big_endian(&mut token_id);
        vec![
            String::from("wrapped").as_bytes().to_vec(),
            data.token_chain.to_be_bytes().to_vec(),
            data.token_address.to_vec(),
            token_id,
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
