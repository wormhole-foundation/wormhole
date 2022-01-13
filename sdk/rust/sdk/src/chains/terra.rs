use cosmwasm_std::{
    to_binary,
    Addr,
    Binary,
    CosmosMsg,
    DepsMut,
    Env,
    QueryRequest,
    StdResult,
    WasmMsg,
    WasmQuery,
};
use serde::Serialize;

use bridge::msg::{
    ExecuteMsg,
    QueryMsg,
};
use bridge::state::ParsedVAA;

/// Export Core Mainnet Contract Address
#[cfg(feature = "mainnet")]
pub fn id() -> Addr {
    Addr::unchecked("terra1dq03ugtd40zu9hcgdzrsq6z2z4hwhc9tqk2uy5")
}

/// Export Core Devnet Contract Address
#[cfg(feature = "devnet")]
pub fn id() -> Addr {
    Addr::unchecked("terra1pd65m0q9tl3v8znnz5f5ltsfegyzah7g42cx5v")
}

pub fn post_message<T>(wormhole: Addr, nonce: u32, message: &T) -> StdResult<CosmosMsg>
where
    T: Serialize,
    T: ?Sized,
{
    Ok(CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: wormhole.to_string(),
        funds:         vec![],
        msg:           to_binary(&ExecuteMsg::PostMessage {
            message: to_binary(message)?,
            nonce,
        })?,
    }))
}

/// Parse a VAA using the Wormhole contract Query interface.
pub fn parse_vaa(
    wormhole: Addr,
    deps: DepsMut,
    env: Env,
    data: &Binary,
) -> StdResult<ParsedVAA> {
    let vaa: ParsedVAA = deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: wormhole.to_string(),
        msg:           to_binary(&QueryMsg::VerifyVAA {
            vaa: data.clone(),
            block_time: env.block.time.seconds(),
        })?,
    }))?;
    Ok(vaa)
}
