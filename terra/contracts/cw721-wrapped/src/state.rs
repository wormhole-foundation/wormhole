use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};

use cosmwasm_std::{
    Binary,
    CanonicalAddr,
    Storage,
};
use cosmwasm_storage::{
    singleton,
    singleton_read,
    ReadonlySingleton,
    Singleton,
};

pub const KEY_WRAPPED_ASSET: &[u8] = b"wrappedAsset";

// Created at initialization and reference original asset and bridge address
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct WrappedAssetInfo {
    pub asset_chain: u16,      // Asset chain id
    pub asset_address: Binary, // Asset smart contract address on the original chain
    pub bridge: CanonicalAddr, // Bridge address, authorized to mint and burn wrapped tokens
}

pub fn wrapped_asset_info(storage: &mut dyn Storage) -> Singleton<WrappedAssetInfo> {
    singleton(storage, KEY_WRAPPED_ASSET)
}

pub fn wrapped_asset_info_read(storage: &dyn Storage) -> ReadonlySingleton<WrappedAssetInfo> {
    singleton_read(storage, KEY_WRAPPED_ASSET)
}
