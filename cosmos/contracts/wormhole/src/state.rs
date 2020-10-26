use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{HumanAddr, Storage};
use cosmwasm_storage::{
    bucket, bucket_read, singleton, singleton_read, Bucket, ReadonlyBucket, ReadonlySingleton,
    Singleton,
};

pub static CONFIG_KEY: &[u8] = b"config";
pub static GUARDIAN_SET_KEY: &[u8] = b"guardian_set";
pub static WRAPPED_ASSET_KEY: &[u8] = b"wrapped_asset";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // Current active guardian set
    pub guardian_set_index: u32,

    // Period for which a guardian set stays active after it has been replaced
    pub guardian_set_expirity: u64,

    // Code id for wrapped asset contract
    pub wrapped_asset_code_id: u64,
}

// Guardian address
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianAddress {
    pub bytes: Vec<u8>, // 20-byte addresses
}

#[cfg(test)]
use hex;
#[cfg(test)]
impl GuardianAddress {
    pub fn from(string: &str) -> GuardianAddress {
        GuardianAddress {
            bytes: hex::decode(string).expect("Decoding failed"),
        }
    }
}

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianSetInfo {
    pub addresses: Vec<GuardianAddress>, // List of guardian addresses
    pub expiration_time: u64,            // Guardian set expiration time
}

// Wormhole contract generic information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct WormholeInfo {
    // Period for which a guardian set stays active after it has been replaced
    pub guardian_set_expirity: u64,
}

pub fn config<S: Storage>(storage: &mut S) -> Singleton<S, ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read<S: Storage>(storage: &S) -> ReadonlySingleton<S, ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn guardian_set<S: Storage>(storage: &mut S) -> Bucket<S, GuardianSetInfo> {
    bucket(GUARDIAN_SET_KEY, storage)
}

pub fn guardian_set_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, GuardianSetInfo> {
    bucket_read(GUARDIAN_SET_KEY, storage)
}

pub fn wrapped_asset<S: Storage>(storage: &mut S) -> Bucket<S, HumanAddr> {
    bucket(WRAPPED_ASSET_KEY, storage)
}

pub fn wrapped_asset_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, HumanAddr> {
    bucket_read(WRAPPED_ASSET_KEY, storage)
}
