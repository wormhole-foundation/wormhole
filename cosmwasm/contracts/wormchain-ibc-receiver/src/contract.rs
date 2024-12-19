use crate::error::ContractError;
use crate::msg::{AllChannelChainsResponse, ChannelChainResponse, ExecuteMsg, QueryMsg};
use crate::state::{CHANNEL_CHAIN, VAA_ARCHIVE};
use anyhow::{bail, ensure, Context};
use cosmwasm_std::{entry_point, to_json_binary, Binary, Deps, Empty, Event, StdResult};
use cosmwasm_std::{DepsMut, Env, MessageInfo, Order, Response};
use serde_wormhole::RawMessage;
use std::str;
use wormhole_bindings::WormholeQuery;
use wormhole_sdk::ibc_receiver::{Action, GovernancePacket};
use wormhole_sdk::vaa::{Body, Header};
use wormhole_sdk::Chain;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    _msg: Empty,
) -> Result<Response, anyhow::Error> {
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
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, anyhow::Error> {
    match msg {
        ExecuteMsg::SubmitUpdateChannelChain { vaas } => submit_vaas(deps, info, vaas),
    }
}

fn submit_vaas(
    mut deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    vaas: Vec<Binary>,
) -> Result<Response, anyhow::Error> {
    let evts = vaas
        .into_iter()
        .map(|v| handle_vaa(deps.branch(), v))
        .collect::<anyhow::Result<Vec<_>>>()?;
    Ok(Response::new()
        .add_attribute("action", "submit_vaas")
        .add_attribute("owner", info.sender)
        .add_events(evts))
}

fn handle_vaa(deps: DepsMut<WormholeQuery>, vaa: Binary) -> anyhow::Result<Event> {
    // parse the VAA header and data
    let (header, data) = serde_wormhole::from_slice::<(Header, &RawMessage)>(&vaa)
        .context("failed to parse VAA header")?;

    // Must be a version 1 VAA
    ensure!(header.version == 1, "unsupported VAA version");

    // call into wormchain to verify the VAA
    deps.querier
        .query::<Empty>(&WormholeQuery::VerifyVaa { vaa: vaa.clone() }.into())
        .context(ContractError::VerifyQuorum)?;

    // parse the VAA body
    let body = serde_wormhole::from_slice::<Body<&RawMessage>>(data)
        .context("failed to parse VAA body")?;

    // validate this is a governance VAA
    ensure!(
        body.emitter_chain == Chain::Solana
            && body.emitter_address == wormhole_sdk::GOVERNANCE_EMITTER,
        "not a governance VAA"
    );

    // parse the governance packet
    let govpacket: GovernancePacket =
        serde_wormhole::from_slice(body.payload).context("failed to parse governance packet")?;

    // validate the governance VAA is directed to wormchain
    ensure!(
        govpacket.chain == Chain::Wormchain,
        "this governance VAA is for another chain"
    );

    // governance VAA replay protection
    let digest = body
        .digest()
        .context("failed to compute governance VAA digest")?;

    if VAA_ARCHIVE.has(deps.storage, &digest.hash) {
        bail!("governance vaa already executed");
    }
    VAA_ARCHIVE
        .save(deps.storage, &digest.hash, &true)
        .context("failed to save governance VAA to archive")?;

    // match the governance action and execute the corresponding logic
    match govpacket.action {
        Action::UpdateChannelChain {
            channel_id,
            chain_id,
        } => {
            ensure!(chain_id != Chain::Wormchain, "the wormchain-ibc-receiver contract should not maintain channel mappings to wormchain");

            let channel_id_str =
                str::from_utf8(&channel_id).context("failed to parse channel-id as utf-8")?;
            let channel_id_trimmed = channel_id_str.trim_start_matches(char::from(0));

            // update storage with the mapping
            CHANNEL_CHAIN
                .save(
                    deps.storage,
                    channel_id_trimmed.to_string(),
                    &chain_id.into(),
                )
                .context("failed to save channel chain")?;
            Ok(Event::new("UpdateChannelChain")
                .add_attribute("chain_id", chain_id.to_string())
                .add_attribute("channel_id", channel_id_trimmed))
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::ChannelChain { channel_id } => {
            query_channel_chain(deps, channel_id).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::AllChannelChains {} => {
            query_all_channel_chains(deps).and_then(|resp| to_json_binary(&resp))
        }
    }
}

fn query_channel_chain(deps: Deps, channel_id: Binary) -> StdResult<ChannelChainResponse> {
    CHANNEL_CHAIN
        .load(deps.storage, channel_id.to_string())
        .map(|chain_id| ChannelChainResponse { chain_id })
}

fn query_all_channel_chains(deps: Deps) -> StdResult<AllChannelChainsResponse> {
    CHANNEL_CHAIN
        .range(deps.storage, None, None, Order::Ascending)
        .map(|res| {
            res.map(|(channel_id, chain_id)| (Binary::from(Vec::<u8>::from(channel_id)), chain_id))
        })
        .collect::<StdResult<Vec<_>>>()
        .map(|channels_chains| AllChannelChainsResponse { channels_chains })
}
