#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use anyhow::Context;
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Empty, Env, MessageInfo, Reply, Response, StdResult,
};
use cw2::set_contract_version;

use crate::{
    bindings::TokenFactoryMsg,
    execute::{
        complete_transfer_and_convert, convert_and_transfer, submit_update_chain_to_channel_map,
        TransferType,
    },
    msg::{
        ExecuteMsg, InstantiateMsg, QueryMsg, COMPLETE_TRANSFER_REPLY_ID, CREATE_DENOM_REPLY_ID,
    },
    query::query_ibc_channel,
    reply::{handle_complete_transfer_reply, handle_create_denom_reply},
    state::{TOKEN_BRIDGE_CONTRACT, WORMHOLE_CONTRACT},
};

// version info for migration info
const CONTRACT_NAME: &str = "ibc-translator";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

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

    WORMHOLE_CONTRACT
        .save(deps.storage, &msg.wormhole_contract)
        .context("failed to save wormhole contract address to storage")?;

    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)
        .context("failed to set contract version")?;

    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("owner", info.sender)
        .add_attribute("version", CONTRACT_VERSION))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: Empty) -> Result<Response, anyhow::Error> {
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    match msg {
        ExecuteMsg::CompleteTransferAndConvert { vaa } => {
            complete_transfer_and_convert(deps, env, info, vaa)
        }
        ExecuteMsg::SimpleConvertAndTransfer {
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
        ExecuteMsg::ContractControlledConvertAndTransfer {
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
            submit_update_chain_to_channel_map(deps, env, info, vaa)
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

    if msg.id == CREATE_DENOM_REPLY_ID {
        return handle_create_denom_reply(deps, env, msg);
    }

    // other cases probably from calling into the burn/mint messages and token factory methods
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::IbcChannel { chain_id } => to_binary(&query_ibc_channel(deps, chain_id)?),
    }
}
