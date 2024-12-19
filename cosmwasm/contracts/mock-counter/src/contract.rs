use crate::{msg::ReceiveMsg, CountResponse, ExecuteMsg, InstantiateMsg, QueryMsg};
use cosmwasm_std::{
    entry_point, from_json, to_json_binary, Binary, Deps, DepsMut, Env, MessageInfo, Response,
    StdResult,
};
use cw_storage_plus::Item;

pub const COUNT: Item<u32> = Item::new("count");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response> {
    COUNT.save(deps.storage, &0)?;

    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("count", 0.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::Increment {} => increment(deps),
        ExecuteMsg::Reset {} => reset(deps),
        ExecuteMsg::Receive(receive_msg) => match from_json(receive_msg.msg)? {
            ReceiveMsg::Increment {} => increment(deps),
            ReceiveMsg::Reset {} => reset(deps),
        },
    }
}

pub fn increment(deps: DepsMut) -> StdResult<Response> {
    COUNT.update(deps.storage, |mut count| -> StdResult<_> {
        count += 1;
        Ok(count)
    })?;

    Ok(Response::new().add_attribute("method", "increment"))
}

pub fn reset(deps: DepsMut) -> StdResult<Response> {
    COUNT.save(deps.storage, &0)?;

    Ok(Response::new()
        .add_attribute("method", "reset")
        .add_attribute("count", 0u32.to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetCount {} => to_json_binary(&query_count(deps)?),
    }
}

fn query_count(deps: Deps) -> StdResult<CountResponse> {
    let count = COUNT.load(deps.storage)?;
    Ok(CountResponse { count })
}
