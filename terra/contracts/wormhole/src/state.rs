use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{Binary, CanonicalAddr, HumanAddr, StdResult, Storage};
use cosmwasm_storage::{
    bucket, bucket_read, singleton, singleton_read, Bucket, ReadonlyBucket, ReadonlySingleton,
    Singleton,
};

pub static CONFIG_KEY: &[u8] = b"config";
pub static GUARDIAN_SET_KEY: &[u8] = b"guardian_set";
pub static WRAPPED_ASSET_KEY: &[u8] = b"wrapped_asset";
pub static WRAPPED_ASSET_ADDRESS_KEY: &[u8] = b"wrapped_asset_address";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // Current active guardian set
    pub guardian_set_index: u32,

    // Period for which a guardian set stays active after it has been replaced
    pub guardian_set_expirity: u64,

    // Code id for wrapped asset contract
    pub wrapped_asset_code_id: u64,

    // Contract owner address, it can make contract active/inactive
    pub owner: CanonicalAddr,

    // If true the contract is active and functioning
    pub is_active: bool,
}

// Guardian address
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianAddress {
    pub bytes: Binary, // 20-byte addresses
}

#[cfg(test)]
use hex;
#[cfg(test)]
impl GuardianAddress {
    pub fn from(string: &str) -> GuardianAddress {
        GuardianAddress {
            bytes: hex::decode(string).expect("Decoding failed").into(),
        }
    }
}

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianSetInfo {
    pub addresses: Vec<GuardianAddress>, // List of guardian addresses
    pub expiration_time: u64,            // Guardian set expiration time
}

impl GuardianSetInfo {
    pub fn quorum(&self) -> usize {
        ((self.addresses.len() * 10 / 3) * 2) / 10 + 1
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

pub fn config_read<S: Storage>(storage: &S) -> ReadonlySingleton<S, ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn guardian_set_set<S: Storage>(
    storage: &mut S,
    index: u32,
    data: &GuardianSetInfo,
) -> StdResult<()> {
    bucket(GUARDIAN_SET_KEY, storage).save(&index.to_le_bytes(), data)
}

pub fn guardian_set_get<S: Storage>(storage: &S, index: u32) -> StdResult<GuardianSetInfo> {
    bucket_read(GUARDIAN_SET_KEY, storage).load(&index.to_le_bytes())
}

pub fn vaa_archive_add<S: Storage>(storage: &mut S, hash: &[u8]) -> StdResult<()> {
    bucket(GUARDIAN_SET_KEY, storage).save(hash, &true)
}

pub fn vaa_archive_check<S: Storage>(storage: &S, hash: &[u8]) -> bool {
    bucket_read(GUARDIAN_SET_KEY, storage)
        .load(&hash)
        .or::<bool>(Ok(false))
        .unwrap()
}

pub fn wrapped_asset<S: Storage>(storage: &mut S) -> Bucket<S, HumanAddr> {
    bucket(WRAPPED_ASSET_KEY, storage)
}

pub fn wrapped_asset_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, HumanAddr> {
    bucket_read(WRAPPED_ASSET_KEY, storage)
}

pub fn wrapped_asset_address<S: Storage>(storage: &mut S) -> Bucket<S, Vec<u8>> {
    bucket(WRAPPED_ASSET_ADDRESS_KEY, storage)
}

pub fn wrapped_asset_address_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, Vec<u8>> {
    bucket_read(WRAPPED_ASSET_ADDRESS_KEY, storage)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn build_guardian_set(length: usize) -> GuardianSetInfo {
        let mut addresses: Vec<GuardianAddress> = Vec::with_capacity(length);
        for _ in 0..length {
            addresses.push(GuardianAddress {
                bytes: vec![].into(),
            });
        }

        GuardianSetInfo {
            addresses,
            expiration_time: 0,
        }
    }

    #[test]
    fn quardian_set_quorum() {
        assert_eq!(build_guardian_set(1).quorum(), 1);
        assert_eq!(build_guardian_set(2).quorum(), 2);
        assert_eq!(build_guardian_set(3).quorum(), 3);
        assert_eq!(build_guardian_set(4).quorum(), 3);
        assert_eq!(build_guardian_set(5).quorum(), 4);
        assert_eq!(build_guardian_set(6).quorum(), 5);
        assert_eq!(build_guardian_set(7).quorum(), 5);
        assert_eq!(build_guardian_set(8).quorum(), 6);
        assert_eq!(build_guardian_set(9).quorum(), 7);
        assert_eq!(build_guardian_set(10).quorum(), 7);
        assert_eq!(build_guardian_set(11).quorum(), 8);
        assert_eq!(build_guardian_set(12).quorum(), 9);
        assert_eq!(build_guardian_set(20).quorum(), 14);
        assert_eq!(build_guardian_set(25).quorum(), 17);
        assert_eq!(build_guardian_set(100).quorum(), 67);
    }
}
