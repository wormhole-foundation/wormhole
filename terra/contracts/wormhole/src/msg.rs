use cosmwasm_std::{HumanAddr, Uint128};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::state::{GuardianSetInfo, GuardianAddress};

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
        vaa: Vec<u8>,
    },
    RegisterAssetHook {
        asset_id: Vec<u8>,
    },
    LockAssets {
        asset: HumanAddr,
        amount: Uint128,
        recipient: Vec<u8>,
        target_chain: u8,
        nonce: u32,
    },
    TokensLocked {
        target_chain: u8,
        token_chain: u8,
        token_decimals: u8,
        token: Vec<u8>,
        sender: Vec<u8>,
        recipient: Vec<u8>,
        amount: Uint128,
        nonce: u32,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    GuardianSetInfo {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianSetInfoResponse {
    pub guardian_set_index: u32, // Current guardian set index
    pub addresses: Vec<GuardianAddress>,  // List of querdian addresses
}