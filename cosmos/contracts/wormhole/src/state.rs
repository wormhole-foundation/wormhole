use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{Storage};
use cosmwasm_storage::{Bucket, ReadonlyBucket, bucket, bucket_read, Singleton, ReadonlySingleton, singleton, singleton_read};

pub static CONFIG_KEY: &[u8] = b"config";
pub static GUARDIAN_SET_KEY: &[u8] = b"guardian_set";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // Current active guardian set
    pub guardian_set_index: u32,

    // Period for which a guardian set stays active after it has been replaced
    pub guardian_set_expirity: u64,
}

// Guardian address
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianAddress {
    pub bytes: Vec<u8>,                   // 20-byte addresses
}

#[cfg(test)]
use hex;
#[cfg(test)]
impl GuardianAddress {
    pub fn from(string: &str) -> GuardianAddress {
        GuardianAddress {
            bytes: hex::decode(string).expect("Decoding failed")
        }
    }
}

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianSetInfo {
    pub addresses: Vec<GuardianAddress>,      // List of guardian addresses
    pub expiration_time: u64,                 // Guardian set expiration time
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