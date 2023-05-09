use cosmwasm_std::{Binary, Uint128};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use crate::token_address::{ExternalTokenId, TokenId};

type HumanAddr = String;

/// The instantiation parameters of the token bridge contract. See
/// [`crate::state::ConfigInfo`] for more details on what these fields mean.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct InstantiateMsg {
    pub gov_chain: u16,
    pub gov_address: Binary,

    pub wormhole_contract: HumanAddr,
    pub wrapped_asset_code_id: u64,

    pub chain_id: u16,
    pub native_denom: String,
    pub native_symbol: String,
    pub native_decimals: u8,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    RegisterAssetHook {
        chain: u16,
        token_address: ExternalTokenId,
    },

    DepositTokens {},
    WithdrawTokens {
        asset: AssetInfo,
    },

    InitiateTransfer {
        asset: Asset,
        recipient_chain: u16,
        recipient: Binary,
        fee: Uint128,
        nonce: u32,
    },

    InitiateTransferWithPayload {
        asset: Asset,
        recipient_chain: u16,
        recipient: Binary,
        fee: Uint128,
        payload: Binary,
        nonce: u32,
    },

    SubmitVaa {
        data: Binary,
    },

    CreateAssetMeta {
        asset_info: AssetInfo,
        nonce: u32,
    },

    CompleteTransferWithPayload {
        data: Binary,
        relayer: HumanAddr,
    },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct MigrateMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    WrappedRegistry { chain: u16, address: Binary },
    TransferInfo { vaa: Binary },
    ExternalId { external_id: Binary },
    IsVaaRedeemed { vaa: Binary },
    ChainRegistration { chain: u16 },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct WrappedRegistryResponse {
    pub address: HumanAddr,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct TransferInfoResponse {
    pub amount: Uint128,
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
    pub fee: Uint128,
    pub payload: Vec<u8>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct ExternalIdResponse {
    pub token_id: TokenId,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct IsVaaRedeemedResponse {
    pub is_redeemed: bool,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct ChainRegistrationResponse {
    pub address: Binary,
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

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct Asset {
    pub info: AssetInfo,
    pub amount: Uint128,
}

/// AssetInfo contract_addr is usually passed from the cw20 hook
/// so we can trust the contract_addr is properly validated.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum AssetInfo {
    Token { contract_addr: String },
    NativeToken { denom: String },
}
