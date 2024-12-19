#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use anyhow::{bail, Context};
use cosmwasm_std::{
    to_json_binary, Binary, Deps, DepsMut, Empty, Env, MessageInfo, Reply, Response, StdResult,
};
use wormhole_bindings::{tokenfactory::TokenFactoryMsg, WormholeQuery};

use crate::{
    execute::{
        complete_transfer_and_convert, convert_and_transfer, submit_update_chain_to_channel_map,
        TransferType,
    },
    msg::{ExecuteMsg, InstantiateMsg, QueryMsg, COMPLETE_TRANSFER_REPLY_ID},
    query::query_ibc_channel,
    reply::handle_complete_transfer_reply,
    state::TOKEN_BRIDGE_CONTRACT,
};

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, anyhow::Error> {
    TOKEN_BRIDGE_CONTRACT
        .save(deps.storage, &msg.token_bridge_contract)
        .context("failed to save token bridge contract address to storage")?;

    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("owner", info.sender))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: Empty) -> Result<Response, anyhow::Error> {
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut<WormholeQuery>,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    match msg {
        ExecuteMsg::CompleteTransferAndConvert { vaa } => {
            complete_transfer_and_convert(deps, env, info, vaa)
        }
        ExecuteMsg::GatewayConvertAndTransfer {
            recipient,
            chain,
            fee,
            nonce,
        } => convert_and_transfer(
            deps,
            info,
            env,
            recipient,
            chain,
            TransferType::Simple { fee },
            nonce,
        ),
        ExecuteMsg::GatewayConvertAndTransferWithPayload {
            contract,
            chain,
            payload,
            nonce,
        } => convert_and_transfer(
            deps,
            info,
            env,
            contract,
            chain,
            TransferType::ContractControlled { payload },
            nonce,
        ),
        ExecuteMsg::SubmitUpdateChainToChannelMap { vaa } => {
            submit_update_chain_to_channel_map(deps, vaa)
        }
    }
}

/// Reply handler for various kinds of replies
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(
    deps: DepsMut,
    env: Env,
    msg: Reply,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    if msg.id == COMPLETE_TRANSFER_REPLY_ID {
        return handle_complete_transfer_reply(deps, env, msg);
    }

    // for safety, let's error out if we don't match a reply ID
    bail!("unmatched reply id {}", msg.id);
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::IbcChannel { chain_id } => to_json_binary(&query_ibc_channel(deps, chain_id)?),
    }
}
