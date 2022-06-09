//! This module contains auxiliary queries against the native cosmos modules that
//! are not covered by cosmwasm-std.
//!
//! The queries are adapted from [1].
//!
//! [1]: https://buf.build/cosmos/cosmos-sdk/file/c03d23cee0a9488c835dee787f2deebb:cosmos/

use cosmwasm_std::{
    from_binary, to_vec, ContractResult, CustomQuery, QuerierWrapper, StdError, StdResult,
    SystemResult,
};
use schemars::JsonSchema;
use serde::{de::DeserializeOwned, Deserialize, Serialize};

pub fn query<U: DeserializeOwned, C: CustomQuery>(
    querier: &QuerierWrapper<C>,
    request: &QueryRequest,
) -> StdResult<U> {
    let raw = to_vec(request).map_err(|serialize_err| {
        StdError::generic_err(format!("Serializing QueryRequest: {}", serialize_err))
    })?;
    match querier.raw_query(&raw) {
        SystemResult::Err(system_err) => Err(StdError::generic_err(format!(
            "Querier system error: {}",
            system_err
        ))),
        SystemResult::Ok(ContractResult::Err(contract_err)) => Err(StdError::generic_err(format!(
            "Querier contract error: {}",
            contract_err
        ))),
        SystemResult::Ok(ContractResult::Ok(value)) => from_binary(&value),
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryRequest {
    Bank(BankQuery),
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
/// Bank queries from
/// https://buf.build/cosmos/cosmos-sdk/file/c03d23cee0a9488c835dee787f2deebb:cosmos/bank/v1beta1/query.proto
pub enum BankQuery {
    /// This calls into the native bank module for one denomination
    /// Return value is [`DenomMetadataResponse`]
    DenomMetadata { denom: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct DenomMetadataResponse {
    /// metadata describes and provides all the client information for the requested token.
    pub metadata: Metadata,
}

/// DenomUnit represents a struct that describes a given
/// denomination unit of the basic token.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct DenomUnit {
    /// denom represents the string name of the given denom unit (e.g uatom).
    pub denom: String,
    /// exponent represents power of 10 exponent that one must
    /// raise the base_denom to in order to equal the given DenomUnit's denom
    /// 1 denom = 1^exponent base_denom
    /// (e.g. with a base_denom of uatom, one can create a DenomUnit of 'atom' with
    /// exponent = 6, thus: 1 atom = 10^6 uatom).
    pub exponent: u32,
    /// aliases is a list of string aliases for the given denom
    pub aliases: Vec<String>,
}

/// Metadata represents a struct that describes
/// a basic token.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct Metadata {
    pub description: String,
    /// denom_units represents the list of DenomUnit's for a given coin
    pub denom_units: Vec<DenomUnit>,
    /// base represents the base denom (should be the DenomUnit with exponent = 0).
    pub base: String,
    /// display indicates the suggested denom that should be
    /// displayed in clients.
    pub display: String,
    /// name defines the name of the token (eg: Cosmos Atom)
    pub name: String,
    /// symbol is the token symbol usually shown on exchanges (eg: ATOM). This can
    /// be the same as the display.
    pub symbol: String,
}
