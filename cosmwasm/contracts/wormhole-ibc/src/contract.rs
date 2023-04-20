#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cw_wormhole::{
    contract::{
        execute as core_execute, instantiate as core_instantiate, migrate as core_migrate,
        query as core_query, query_parse_and_verify_vaa,
    },
    state::config_read,
};
use wormhole_sdk::{
    ibc_receiver::{Action, GovernancePacket},
    Chain,
};

use crate::{
    ibc::PACKET_LIFETIME,
    msg::ExecuteMsg,
    state::{VAA_ARCHIVE, WORMCHAIN_CHANNEL_ID},
};
use anyhow::{bail, ensure, Context};
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Env, Event, IbcMsg, MessageInfo, Response, StdResult,
};
use cw_wormhole::msg::{ExecuteMsg as WormholeExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg};

use crate::msg::WormholeIbcPacketMsg;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, anyhow::Error> {
    // execute the wormhole core contract instantiation
    core_instantiate(deps, env, info, msg).context("wormhole core instantiation failed")
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(deps: DepsMut, env: Env, msg: MigrateMsg) -> Result<Response, anyhow::Error> {
    // call the core contract migrate function
    core_migrate(deps, env, msg).context("wormhole core migration failed")
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, anyhow::Error> {
    match msg {
        ExecuteMsg::SubmitVAA { vaa } => {
            core_execute(deps, env, info, WormholeExecuteMsg::SubmitVAA { vaa })
                .context("failed core submit_vaa execution")
        }
        ExecuteMsg::PostMessage { message, nonce } => post_message_ibc(
            deps,
            env,
            info,
            WormholeExecuteMsg::PostMessage { message, nonce },
        ),
        ExecuteMsg::SubmitUpdateChannelChain { vaa } => {
            let evt = handle_vaa(deps, env, vaa)?;
            Ok(Response::new()
                .add_attribute("action", "submit_vaas")
                .add_attribute("owner", info.sender)
                .add_event(evt))
        }
    }
}

fn handle_vaa(deps: DepsMut, env: Env, vaa: Binary) -> anyhow::Result<Event> {
    // parse the VAA header and data
    let vaa = query_parse_and_verify_vaa(deps.as_ref(), vaa.as_slice(), env.block.time.seconds())
        .context("failed to parse vaa")?;

    // validate this is a governance VAA
    ensure!(
        Chain::from(vaa.emitter_chain) == Chain::Solana
            && vaa.emitter_address == wormhole_sdk::GOVERNANCE_EMITTER.0,
        "not a governance VAA"
    );

    // parse the governance packet
    let govpacket = serde_wormhole::from_slice::<GovernancePacket>(&vaa.payload)
        .context("failed to parse governance packet")?;

    // validate the governance VAA is directed to this chain
    let state = config_read(deps.storage)
        .load()
        .context("failed to load contract config")?;
    ensure!(
        govpacket.chain == Chain::from(state.chain_id),
        format!("this governance VAA is for chain {}, which does not match this chain ({})", u16::from(govpacket.chain), state.chain_id)
    );

    // governance VAA replay protection
    if VAA_ARCHIVE.has(deps.storage, vaa.hash.as_slice()) {
        bail!("governance vaa already executed");
    }
    VAA_ARCHIVE
        .save(deps.storage, vaa.hash.as_slice(), &true)
        .context("failed to save governance VAA to archive")?;

    // match the governance action and execute the corresponding logic
    match govpacket.action {
        Action::UpdateChannelChain {
            channel_id,
            chain_id,
        } => {
            // validate that the chain_id for the channel is wormchain
            // we should only be whitelisting IBC connections to wormchain
            ensure!(
                chain_id == Chain::Wormchain,
                "whitelisted ibc channel not for wormchain"
            );

            let channel_id_str = String::from_utf8(channel_id.to_vec())
                .context("failed to parse channel-id as utf-8")?;

            // update the whitelisted wormchain channel id
            WORMCHAIN_CHANNEL_ID
                .save(deps.storage, &channel_id_str)
                .context("failed to save channel chain")?;
            Ok(Event::new("UpdateChannelChain")
                .add_attribute("chain_id", chain_id.to_string())
                .add_attribute("channel_id", channel_id_str))
        }
    }
}

fn post_message_ibc(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: WormholeExecuteMsg,
) -> anyhow::Result<Response> {
    let channel_id = WORMCHAIN_CHANNEL_ID
        .load(deps.storage)
        .context("failed to load whitelisted wormchain channel id")?;

    // compute the packet timeout (infinite timeout)
    let packet_timeout = env.block.time.plus_seconds(PACKET_LIFETIME).into();

    // compute the block height
    let block_height = env.block.height.to_string();

    // compute the transaction index
    // (this is an optional since not all messages are executed as part of txns)
    // (they may be executed part of the pre/post block handlers)
    let tx_index = env.transaction.as_ref().map(|tx_info| tx_info.index);

    // actually execute the postMessage call on the core contract
    let mut res = core_execute(deps, env, info, msg).context("wormhole core execution failed")?;

    res = match tx_index {
        Some(index) => res.add_attribute("message.tx_index", index.to_string()),
        None => res,
    };
    res = res.add_attribute("message.block_height", block_height);

    // Send the result attributes over IBC on this channel
    let packet = WormholeIbcPacketMsg::Publish { msg: res.clone() };
    let ibc_msg = IbcMsg::SendPacket {
        channel_id,
        data: to_binary(&packet)?,
        timeout: packet_timeout,
    };

    // add the IBC message to the response
    Ok(res
        .add_attribute("is_ibc", true.to_string())
        .add_message(ibc_msg))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    // defer to the core contract logic for all query handling
    core_query(deps, env, msg)
}

#[cfg(test)]
mod tests;
