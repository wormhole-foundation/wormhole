use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, Coin};

use crate::state::{GuardianAddress, GuardianSetInfo, ParsedVAA};

type HumanAddr = String;

/// The instantiation parameters of the core bridge contract. See
/// [`crate::state::ConfigInfo`] for more details on what these fields mean.
#[cw_serde]
pub struct InstantiateMsg {
    pub gov_chain: u16,
    pub gov_address: Binary,

    /// Guardian set to initialise the contract with.
    pub initial_guardian_set: GuardianSetInfo,
    pub guardian_set_expirity: u64,

    pub chain_id: u16,
    pub fee_denom: String,
}

#[cw_serde]
pub enum ExecuteMsg {
    SubmitVAA { vaa: Binary },
    PostMessage { message: Binary, nonce: u32 },
}

#[cw_serde]
pub struct MigrateMsg {}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(GuardianSetInfoResponse)]
    GuardianSetInfo {},
    #[returns(ParsedVAA)]
    VerifyVAA { vaa: Binary, block_time: u64 },
    #[returns(GetStateResponse)]
    GetState {},
    #[returns(GetAddressHexResponse)]
    QueryAddressHex { address: HumanAddr },
}

#[cw_serde]
pub struct GuardianSetInfoResponse {
    pub guardian_set_index: u32,         // Current guardian set index
    pub addresses: Vec<GuardianAddress>, // List of querdian addresses
}

#[cw_serde]
pub struct WrappedRegistryResponse {
    pub address: HumanAddr,
}

#[cw_serde]
pub struct GetStateResponse {
    pub fee: Coin,
}

#[cw_serde]
pub struct GetAddressHexResponse {
    pub hex: String,
}
