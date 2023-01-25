use anyhow::{ensure, Context};
use cosmwasm_std::{entry_point, to_binary, Binary, Deps, Empty, Event, StdError, StdResult};
use cosmwasm_std::{DepsMut, Env, MessageInfo, Order, Response};
use cw2::{get_contract_version, set_contract_version};
use semver::Version;
use serde_wormhole::RawMessage;
use wormhole::ibc_receiver::{Action, GovernancePacket};
use wormhole::vaa::{Body, Header};
use wormhole::Chain;
use wormhole_bindings::WormholeQuery;

use crate::error::ContractError;
use crate::msg::{AllChainConnectionsResponse, ChainConnectionResponse, ExecuteMsg, QueryMsg};
use crate::state::CHAIN_CONNECTIONS;

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:wormchain-ibc-receiver";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    _msg: Empty,
) -> Result<Response, anyhow::Error> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)
        .context("failed to set contract version")?;

    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("owner", info.sender)
        .add_attribute("version", CONTRACT_VERSION))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(deps: DepsMut, _env: Env, _msg: Empty) -> Result<Response, anyhow::Error> {
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
        return Err(StdError::generic_err("Cannot upgrade from a newer or equal version").into());
    }

    // set the new version
    cw2::set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

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
        ExecuteMsg::SubmitUpdateChainConnection { vaas } => submit_vaas(deps, info, vaas),
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
        body.emitter_chain == Chain::Solana && body.emitter_address == wormhole::GOVERNANCE_EMITTER,
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

    // match the governance action and execute the corresponding logic
    match govpacket.action {
        Action::UpdateChainConnection {
            connection_id,
            chain_id,
        } => {
            // update storage with the mapping
            CHAIN_CONNECTIONS
                .save(deps.storage, connection_id.to_string(), &chain_id.into())
                .context("failed to save chain connection")?;
            Ok(Event::new("UpdateChainConnection")
                .add_attribute("chain_id", chain_id.to_string())
                .add_attribute("connection_id", connection_id.to_string()))
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::ChainConnection { connection_id } => {
            query_chain_connection(deps, connection_id).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::AllChainConnections {} => {
            query_all_chain_connections(deps).and_then(|resp| to_binary(&resp))
        }
    }
}

fn query_chain_connection(deps: Deps, connection_id: Binary) -> StdResult<ChainConnectionResponse> {
    CHAIN_CONNECTIONS
        .load(deps.storage, connection_id.to_string())
        .map(|chain_id| ChainConnectionResponse { chain_id })
}

fn query_all_chain_connections(deps: Deps) -> StdResult<AllChainConnectionsResponse> {
    CHAIN_CONNECTIONS
        .range(deps.storage, None, None, Order::Ascending)
        .map(|res| {
            res.map(|(connection_id, chain_id)| {
                (Binary::from(Vec::<u8>::from(connection_id)), chain_id)
            })
        })
        .collect::<StdResult<Vec<_>>>()
        .map(|chain_connections| AllChainConnectionsResponse { chain_connections })
}

#[cfg(test)]
mod tests {
    use cosmwasm_std::{
        testing::{mock_dependencies, mock_env, mock_info},
        Empty,
    };
    use cw2::get_contract_version;

    use super::{instantiate, CONTRACT_NAME, CONTRACT_VERSION};

    #[test]
    fn instantiate_works() {
        let mut deps = mock_dependencies();

        const SENDER: &str = "creator";
        let info = mock_info(SENDER, &[]);
        let res = instantiate(deps.as_mut(), mock_env(), info, Empty {}).unwrap();

        // the response should have 0 messages and 3 attributes
        assert_eq!(0, res.messages.len());
        assert_eq!(3, res.attributes.len());

        // validate the attributes and their values
        res.attributes.iter().for_each(|a| {
            let value = if a.key == "action" {
                "instantiate"
            } else if a.key == "owner" {
                SENDER
            } else if a.key == "version" {
                CONTRACT_VERSION
            } else {
                panic!("invalid attribute key");
            };

            assert_eq!(a.value, value);
        });

        // check that contract version & name have been set
        let contract_version = get_contract_version(deps.as_ref().storage).unwrap();
        assert_eq!(CONTRACT_NAME, contract_version.contract);
        assert_eq!(CONTRACT_VERSION, contract_version.version);
    }
}
