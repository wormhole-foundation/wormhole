use cosmwasm_std::{Binary, HumanAddr, Uint128, Coin};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::state::{GuardianAddress, GuardianSetInfo};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InitMsg {
    pub initial_guardian_set: GuardianSetInfo,
    pub guardian_set_expirity: u64,
    pub wrapped_asset_code_id: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum HandleMsg {
    SubmitVAA {
        vaa: Binary,
    },
    RegisterAssetHook {
        asset_id: Binary,
    },
    LockAssets {
        asset: HumanAddr,
        amount: Uint128,
        recipient: Binary,
        target_chain: u8,
        nonce: u32,
    },
    TransferFee {
        amount: Coin,
        recipient: HumanAddr,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    GuardianSetInfo {},
    WrappedRegistry { chain: u8, address: Binary },
    VerifyVAA { vaa: Binary, block_time: u64 },
    GetState {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct GuardianSetInfoResponse {
    pub guardian_set_index: u32,         // Current guardian set index
    pub addresses: Vec<GuardianAddress>, // List of querdian addresses
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct WrappedRegistryResponse {
    pub address: HumanAddr,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct GetStateResponse {
    pub fee: Coin,
}
