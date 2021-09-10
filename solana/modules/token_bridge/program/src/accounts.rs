use crate::types::*;
use bridge::{
    accounts::BridgeData,
    api::ForeignAddress,
    vaa::{
        DeserializePayload,
        PayloadMessage,
    },
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    *,
};

pub type AuthoritySigner<'b> = Derive<Info<'b>, "authority_signer">;
pub type CustodySigner<'b> = Derive<Info<'b>, "custody_signer">;
pub type MintSigner<'b> = Derive<Info<'b>, "mint_signer">;

pub type CoreBridge<'a, const State: AccountState> = Data<'a, BridgeData, { State }>;

pub type EmitterAccount<'b> = Derive<Info<'b>, "emitter">;

pub type ConfigAccount<'b, const State: AccountState> =
    Derive<Data<'b, Config, { State }>, "config">;

pub type CustodyAccount<'b, const State: AccountState> = Data<'b, SplAccount, { State }>;

pub struct CustodyAccountDerivationData {
    pub mint: Pubkey,
}

impl<'b, const State: AccountState> Seeded<&CustodyAccountDerivationData>
    for CustodyAccount<'b, { State }>
{
    fn seeds(accs: &CustodyAccountDerivationData) -> Vec<Vec<u8>> {
        vec![accs.mint.to_bytes().to_vec()]
    }
}

pub type WrappedMint<'b, const State: AccountState> = Data<'b, SplMint, { State }>;

pub struct WrappedDerivationData {
    pub token_chain: ChainID,
    pub token_address: ForeignAddress,
}

impl<'b, const State: AccountState> Seeded<&WrappedDerivationData> for WrappedMint<'b, { State }> {
    fn seeds(data: &WrappedDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("wrapped").as_bytes().to_vec(),
            data.token_chain.to_be_bytes().to_vec(),
            data.token_address.to_vec(),
        ]
    }
}

pub type WrappedTokenMeta<'b, const State: AccountState> = Data<'b, WrappedMeta, { State }>;

pub struct WrappedMetaDerivationData {
    pub mint_key: Pubkey,
}

impl<'b, const State: AccountState> Seeded<&WrappedMetaDerivationData>
    for WrappedTokenMeta<'b, { State }>
{
    fn seeds(data: &WrappedMetaDerivationData) -> Vec<Vec<u8>> {
        vec![
            String::from("meta").as_bytes().to_vec(),
            data.mint_key.to_bytes().to_vec(),
        ]
    }
}

/// Registered chain endpoint
pub type Endpoint<'b, const State: AccountState> = Data<'b, EndpointRegistration, { State }>;

pub struct EndpointDerivationData {
    pub emitter_chain: u16,
    pub emitter_address: ForeignAddress,
}

/// Seeded implementation based on an incoming VAA
impl<'b, const State: AccountState> Seeded<&EndpointDerivationData> for Endpoint<'b, { State }> {
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
