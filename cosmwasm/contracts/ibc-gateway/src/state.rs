use schemars::JsonSchema;
use serde::{Deserialize, Serialize};
use std::str;

use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Addr, Binary, StdResult, Storage};
use cosmwasm_storage::{
    singleton, singleton_read, ReadonlySingleton,
    Singleton,
};

use cw_storage_plus::Map;

use cw_wormhole::byte_utils::{ByteUtils, get_string_from_32};

type HumanAddr = String;
static CONFIG_KEY: &[u8] = b"config";

pub const CHAIN_CHANNELS: Map<u16, String> = Map::new("chain_channels");

/// Information about this contract's general parameters.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct ConfigInfo {
    /// Governance chain (typically Solana, i.e. chain id 1)
    pub gov_chain: u16,

    /// Address of governance contract (typically 0x0000000000000000000000000000000000000000000000000000000000000004)
    pub gov_address: Vec<u8>,

    /// Address of the core bridge contract
    pub wormhole_contract: HumanAddr,
        
    /// Address of the token bridge contract
    pub token_bridge_contract: HumanAddr,

    /// Code id of the wrapped token contract. When a new token is attested, the
    /// token bridge instantiates a new contract from this code id.
    pub wrapped_asset_code_id: u64,

    /// The wormhole id of the current chain.
    pub chain_id: u16,

    /// The native denom info of the current chain
    /// Other tokens will not be allowed to be attested
    pub native_denom: String,
    pub native_symbol: String,
    pub native_decimals: u8,
}

pub fn config(storage: &mut dyn Storage) -> Singleton<ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

type Serialized128 = String;

/// Structure to keep track of an active CW20 transfer, required to pass state through to the reply
/// handler for submessages during a transfer.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct TransferState {
    pub account: String,
    pub message: Vec<u8>,
    pub multiplier: Serialized128,
    pub nonce: u32,
    pub previous_balance: Serialized128,
    pub token_address: Addr,
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

pub struct RegisterChainChannel {
    pub chain_id: u16,
    pub channel_id: String,
}

impl RegisterChainChannel {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let chain_id = data.get_u16(0);
        // Note that get_string_from_32 actually handles longer strings.
        let channel_id = get_string_from_32(&data[2..]);

        Ok(RegisterChainChannel {
            chain_id,
            channel_id,
        })
    }
}

#[cw_serde]
pub enum TransferPayload {
    BasicTransfer { chain_id: u16, recipient: Binary },
}
