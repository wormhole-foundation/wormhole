use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::state::GuardianSetInfo;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InitMsg {
    pub initial_guardian_set: GuardianSetInfo,
    pub guardian_set_expirity: u64,
    pub wrapped_asset_code_id: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum HandleMsg {
    SubmitVAA { vaa: Vec<u8> },
    RegisterAssetHook { asset_id: Vec<u8> },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {}
