use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, Uint128};

use crate::token_address::{ExternalTokenId, TokenId};

type HumanAddr = String;

/// The instantiation parameters of the token bridge contract. See
/// [`crate::state::ConfigInfo`] for more details on what these fields mean.
#[cw_serde]
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

#[cw_serde]
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

#[cw_serde]
pub struct MigrateMsg {}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(WrappedRegistryResponse)]
    WrappedRegistry { chain: u16, address: Binary },
    #[returns(TransferInfoResponse)]
    TransferInfo { vaa: Binary },
    #[returns(ExternalIdResponse)]
    ExternalId { external_id: Binary },
    #[returns(IsVaaRedeemedResponse)]
    IsVaaRedeemed { vaa: Binary },
    #[returns(ChainRegistrationResponse)]
    ChainRegistration { chain: u16 },
}

#[cw_serde]
pub struct WrappedRegistryResponse {
    pub address: HumanAddr,
}

#[cw_serde]
pub struct TransferInfoResponse {
    pub amount: Uint128,
    pub token_address: [u8; 32],
    pub token_chain: u16,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
    pub fee: Uint128,
    pub payload: Vec<u8>,
}

#[cw_serde]
pub struct ExternalIdResponse {
    pub token_id: TokenId,
}

#[cw_serde]
pub struct IsVaaRedeemedResponse {
    pub is_redeemed: bool,
}

#[cw_serde]
pub struct ChainRegistrationResponse {
    pub address: Binary,
}

#[cw_serde]
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
pub struct Asset {
    pub info: AssetInfo,
    pub amount: Uint128,
}

/// AssetInfo contract_addr is usually passed from the cw20 hook
/// so we can trust the contract_addr is properly validated.
#[cw_serde]
pub enum AssetInfo {
    Token { contract_addr: String },
    NativeToken { denom: String },
}
