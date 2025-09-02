use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::Storage;
use cosmwasm_storage::{singleton, singleton_read, ReadonlySingleton, Singleton};

type HumanAddr = String;

pub static CONFIG_KEY: &[u8] = b"config";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct Config {
    pub token_bridge_contract: HumanAddr,
}

pub fn config(storage: &mut dyn Storage) -> Singleton<'_, Config> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<'_, Config> {
    singleton_read(storage, CONFIG_KEY)
}
