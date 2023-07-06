#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use anyhow::{Context, ensure};
use cosmwasm_std::{
    BankMsg, CosmosMsg, DepsMut, Empty, Env, MessageInfo, Response,
};

use crate::{
    msg::{ExecuteMsg, InstantiateMsg},
};

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, anyhow::Error> {
    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: Empty) -> Result<Response, anyhow::Error> {
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    _deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, anyhow::Error> {
    match msg {
        ExecuteMsg::ForwardTokens { recipient } => {
            // bank tokens sent to the contract will be in info.funds
            ensure!(
                info.funds.len() == 1,
                "info.funds should contain only 1 coin"
            );

            // batch calls together
            let mut response: Response = Response::new();
            response = response.add_message(BankMsg::Send {
                to_address: recipient,
                amount: info.funds,
                //amount: vec![amount],
            });
            
            Ok(response)
        },
    }
}
