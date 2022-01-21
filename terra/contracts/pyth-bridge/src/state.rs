use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};

use cosmwasm_std::{
    StdResult,
    Storage,
};
use cosmwasm_storage::{
    bucket,
    bucket_read,
    singleton,
    singleton_read,
    Bucket,
    ReadonlyBucket,
    ReadonlySingleton,
    Singleton,
};

use wormhole::byte_utils::ByteUtils;

type HumanAddr = String;

pub static CONFIG_KEY: &[u8] = b"config";
pub static PRICE_INFO_KEY: &[u8] = b"price_info";
pub static SEQUENCE_KEY: &[u8] = b"sequence";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // governance contract details
    pub gov_chain: u16,
    pub gov_address: Vec<u8>,

    pub wormhole_contract: HumanAddr,
    pub pyth_emitter: Vec<u8>,
    pub pyth_emitter_chain: u16,
}

pub fn config(storage: &mut dyn Storage) -> Singleton<ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn sequence(storage: &mut dyn Storage) -> Singleton<u64> {
    singleton(storage, SEQUENCE_KEY)
}

pub fn sequence_read(storage: &dyn Storage) -> ReadonlySingleton<u64> {
    singleton_read(storage, SEQUENCE_KEY)
}

pub fn price_info(storage: &mut dyn Storage) -> Bucket<Vec<u8>> {
    bucket(storage, PRICE_INFO_KEY)
}

pub fn price_info_read(storage: &dyn Storage) -> ReadonlyBucket<Vec<u8>> {
    bucket_read(storage, PRICE_INFO_KEY)
}

pub struct UpgradeContract {
    pub new_contract: u64,
}

impl UpgradeContract {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let new_contract = data.get_u64(24);
        Ok(UpgradeContract { new_contract })
    }
}
