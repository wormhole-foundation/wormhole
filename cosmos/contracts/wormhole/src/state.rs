use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{CanonicalAddr, Storage, Api, StdResult};
use cosmwasm_storage::{Bucket, ReadonlyBucket, bucket, bucket_read, Singleton, ReadonlySingleton, singleton, singleton_read};
use crate::msg::GuardianSetMsg;

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

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianSetInfo {
    pub addresses_raw: Vec<CanonicalAddr>,      // List of guardian addresses
    pub expiration_time: u64,                   // Guardian set expiration time
}

impl GuardianSetInfo {
    pub fn from<A: Api>(api: &A, guardian_set: &GuardianSetMsg) -> StdResult<GuardianSetInfo> {
        let mut addresses_raw: Vec<CanonicalAddr> = Vec::new();
        for human_addr in &guardian_set.addresses {
            addresses_raw.push(api.canonical_address(&human_addr)?);
        }
        Ok(GuardianSetInfo {addresses_raw, expiration_time: guardian_set.expiration_time})
    }
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

pub fn config_read<S: Storage>(storage: &mut S) -> ReadonlySingleton<S, ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn guardian_set<S: Storage>(storage: &mut S) -> Bucket<S, GuardianSetInfo> {
    bucket(GUARDIAN_SET_KEY, storage)
}

pub fn guardian_set_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, GuardianSetInfo> {
    bucket_read(GUARDIAN_SET_KEY, storage)
}