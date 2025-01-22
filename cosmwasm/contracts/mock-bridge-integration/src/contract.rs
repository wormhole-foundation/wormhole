use cosmwasm_std::{
    entry_point, to_json_binary, Binary, CosmosMsg, Deps, DepsMut, Env, MessageInfo, QueryRequest,
    Reply, Response, StdError, StdResult, SubMsg, WasmMsg, WasmQuery,
};

use crate::{
    msg::{ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg},
    state::{config, config_read, Config},
};

use cw_token_bridge::{
    msg::TransferInfoResponse,
    msg::{ExecuteMsg as TokenBridgeExecuteMsg, QueryMsg as TokenBridgeQueryMessage},
};

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    let state = Config {
        token_bridge_contract: msg.token_bridge_contract,
    };
    config(deps.storage).save(&state)?;

    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(deps: DepsMut, env: Env, info: MessageInfo, msg: ExecuteMsg) -> StdResult<Response> {
    match msg {
        ExecuteMsg::CompleteTransferWithPayload { data } => {
            complete_transfer_with_payload(deps, env, info, &data)
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(_deps: DepsMut, _env: Env, _msg: Reply) -> StdResult<Response> {
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(_deps: Deps, _env: Env, _msg: QueryMsg) -> StdResult<Binary> {
    Err(StdError::generic_err("not implemented"))
}

fn complete_transfer_with_payload(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

    let messages = vec![SubMsg::reply_on_success(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cfg.token_bridge_contract,
            msg: to_json_binary(&TokenBridgeExecuteMsg::CompleteTransferWithPayload {
                data: data.clone(),
                relayer: info.sender.to_string(),
            })?,
            funds: vec![],
        }),
        1,
    )];

    let transfer_info = parse_transfer_vaa(deps.as_ref(), data)?;

    Ok(Response::new()
        .add_submessages(messages)
        .add_attribute("action", "complete_transfer_with_payload")
        .add_attribute(
            "transfer_payload",
            Binary::from(transfer_info.payload).to_base64(),
        ))
}

fn parse_transfer_vaa(deps: Deps, data: &Binary) -> StdResult<TransferInfoResponse> {
    let cfg = config_read(deps.storage).load()?;
    let transfer_info: TransferInfoResponse =
        deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: cfg.token_bridge_contract,
            msg: to_json_binary(&TokenBridgeQueryMessage::TransferInfo { vaa: data.clone() })?,
        }))?;
    Ok(transfer_info)
}
