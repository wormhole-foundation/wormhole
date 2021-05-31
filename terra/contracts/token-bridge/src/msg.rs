use cosmwasm_std::{Binary, HumanAddr, Uint128};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InitMsg {
    pub owner: HumanAddr,
    pub wormhole_contract: HumanAddr,
    pub wrapped_asset_code_id: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum HandleMsg {


    RegisterAssetHook {
        asset_id: Binary,
    },

    InitiateTransfer {
        asset: HumanAddr,
        amount: Uint128,
        recipient_chain: u16,
        recipient: Binary,
        nonce: u32,
    },

    SubmitVaa {
        data: Binary,
    },

    RegisterChain {
        chain_id: u16,
        chain_address: Binary,
    },

    CreateAssetMeta {
        asset_address: HumanAddr,
        nonce: u32,
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    WrappedRegistry { chain: u16, address: Binary },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct WrappedRegistryResponse {
    pub address: HumanAddr,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum WormholeQueryMsg {
    VerifyVAA { vaa: Binary, block_time: u64 },
}
