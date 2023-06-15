use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Binary, Uint128};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

type HumanAddr = String;

/// The instantiation parameters of the token bridge contract. See
/// [`crate::state::ConfigInfo`] for more details on what these fields mean.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct InstantiateMsg {
    pub gov_chain: u16,
    pub gov_address: Binary,

    pub wormhole_contract: HumanAddr,
    pub token_bridge_contract: HumanAddr,
    pub wrapped_asset_code_id: u64,

    pub chain_id: u16,
    pub native_denom: String,
    pub native_symbol: String,
    pub native_decimals: u8,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SubmitVaa {
        data: Binary,
    },

    CompleteTransferWithPayload {
        data: Binary,
        relayer: HumanAddr,
    },

    CompleteTransferAndConvert {
        /// VAA to submit. The VAA should be encoded in the standard wormhole
        /// wire format.
        vaa: Binary,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct MigrateMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    PlaceHolder {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct IsVaaRedeemedResponse {
    pub is_redeemed: bool,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct CompleteTransferResponse {
    // All addresses are bech32-encoded strings.

    // contract address if this minted or unlocked a cw20, otherwise none
    pub contract: Option<String>,
    // denom if this unlocked a native token, otherwise none
    pub denom: Option<String>,
    pub recipient: String,
    pub amount: Uint128,
    pub relayer: String,
    pub fee: Uint128,
}

#[cw_serde]
pub struct AllChainChannelsResponse {
    // a tuple of (connectionId, chainId)
    pub chain_channels: Vec<(u16, Binary)>,
}
