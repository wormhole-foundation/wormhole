#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use crate::ibc::PACKET_LIFETIME;
use anyhow::Context;
use cosmwasm_std::{
    to_binary, DepsMut, Env, IbcChannel, IbcMsg, IbcQuery, ListChannelsResponse, MessageInfo,
    Response, StdError, StdResult,
};
use cw2::{get_contract_version, set_contract_version};
use semver::Version;
use wormhole::msg::{ExecuteMsg, InstantiateMsg, MigrateMsg};

use crate::msg::WormholeIbcPacketMsg;

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:wormhole-ibc";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

// TODO: Set this based on an env variable
const WORMCHAIN_IBC_RECEIVER_PORT: &str =
    "wasm.wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj";

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, anyhow::Error> {
    // save the contract name and version
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)
        .context("failed to set contract version")?;

    // execute the wormhole core contract instantiation
    wormhole::contract::instantiate(deps, env, info, msg)
        .context("wormhole core instantiation failed")
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(deps: DepsMut, env: Env, msg: MigrateMsg) -> Result<Response, anyhow::Error> {
    let ver = get_contract_version(deps.storage)?;
    // ensure we are migrating from an allowed contract
    if ver.contract != CONTRACT_NAME {
        return Err(StdError::generic_err("Can only upgrade from same type").into());
    }

    // ensure we are migrating to a newer version
    let saved_version =
        Version::parse(&ver.version).context("could not parse saved contract version")?;
    let new_version =
        Version::parse(CONTRACT_VERSION).context("could not parse new contract version")?;
    if saved_version >= new_version {
        return Err(StdError::generic_err("Cannot upgrade from a newer version").into());
    }

    // set the new version
    cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    // call the core contract migrate function
    wormhole::contract::migrate(deps, env, msg).context("wormhole core migration failed")
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, anyhow::Error> {
    match msg {
        ExecuteMsg::SubmitVAA { .. } => wormhole::contract::execute(deps, env, info, msg)
            .context("failed core submit_vaa execution"),
        ExecuteMsg::PostMessage { .. } => post_message_ibc(deps, env, info, msg),
    }
}

fn post_message_ibc(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> anyhow::Result<Response> {
    // search for a channel bound to counterparty with the port "wasm.<wormchain_addr>"
    let ibc_channels = deps
        .querier
        .query::<ListChannelsResponse>(&IbcQuery::ListChannels { port_id: None }.into())
        .context("failed to query ibc channels")?;

    let channel_id = find_wormchain_channel_id(ibc_channels.channels)?;

    // compute the packet timeout (infinite timeout)
    let packet_timeout = env.block.time.plus_seconds(PACKET_LIFETIME).into();

    // compute the block height
    let block_height = env.block.height.to_string();

    // compute the transaction index
    // (this is an optional since not all messages are executed as part of txns)
    // (they may be executed part of the pre/post block handlers)
    let tx_index = env.transaction.as_ref().map(|tx_info| tx_info.index);

    // actually execute the postMessage call on the core contract
    let res = wormhole::contract::execute(deps, env, info, msg)
        .context("wormhole core execution failed")?;

    let res_with_tx_index = match tx_index {
        Some(index) => res.add_attribute("message.tx_index", index.to_string()),
        None => res,
    };
    let res_with_block_height =
        res_with_tx_index.add_attribute("message.block_height", block_height);

    // Send the result attributes over IBC on this channel
    let packet = WormholeIbcPacketMsg::Publish {
        msg: res_with_block_height.clone(),
    };
    let ibc_msg = IbcMsg::SendPacket {
        channel_id,
        data: to_binary(&packet)?,
        timeout: packet_timeout,
    };

    // add the IBC message to the response
    Ok(res_with_block_height
        .add_attribute("is_ibc", true.to_string())
        .add_message(ibc_msg))
}

/// Find any IBC channel that is connected to the wormchain integrator contract
fn find_wormchain_channel_id(channels: Vec<IbcChannel>) -> StdResult<String> {
    for c in channels {
        if c.counterparty_endpoint.port_id == WORMCHAIN_IBC_RECEIVER_PORT {
            return Ok(c.endpoint.channel_id);
        }
    }

    Err(StdError::not_found(format!(
        "no channel connecting to wormchain contract port {WORMCHAIN_IBC_RECEIVER_PORT}"
    )))
}

#[cfg(test)]
mod tests;
