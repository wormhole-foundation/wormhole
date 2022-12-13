use std::marker::PhantomData;

use accounting::{
    query_balance, query_modification, query_transfer,
    state::{account, transfer, Modification, TokenAddress, Transfer},
    validate_transfer,
};
use anyhow::{ensure, Context};
#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    from_binary, to_binary, Binary, ConversionOverflowError, CosmosMsg, Deps, DepsMut, Empty, Env,
    Event, MessageInfo, Order, Response, StdError, StdResult, Uint256, WasmMsg,
};
use cw2::set_contract_version;
use cw_storage_plus::Bound;
use tinyvec::{Array, TinyVec};
use wormhole::{token::Message, vaa::Signature};
use wormhole_bindings::WormholeQuery;

use crate::{
    bail,
    error::{AnyError, ContractError},
    msg::{
        AllAccountsResponse, AllModificationsResponse, AllPendingTransfersResponse,
        AllTransfersResponse, ExecuteMsg, Instantiate, InstantiateMsg, MigrateMsg, Observation,
        QueryMsg, Upgrade,
    },
    state::{self, Data, PendingTransfer, PENDING_TRANSFERS, TOKENBRIDGE_ADDR},
};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:wormchain-accounting";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut<WormholeQuery>,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, AnyError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)
        .context("failed to set contract version")?;

    let _: Empty = deps
        .querier
        .query(
            &WormholeQuery::VerifyQuorum {
                data: msg.instantiate.clone(),
                guardian_set_index: msg.guardian_set_index,
                signatures: msg.signatures,
            }
            .into(),
        )
        .context(ContractError::VerifyQuorum)?;

    let init: Instantiate =
        from_binary(&msg.instantiate).context("failed to parse `Instantiate` message")?;

    let tokenbridge_addr = deps
        .api
        .addr_validate(&init.tokenbridge_addr)
        .context("failed to validate tokenbridge address")?;

    TOKENBRIDGE_ADDR
        .save(deps.storage, &tokenbridge_addr)
        .context("failed to save tokenbridge address")?;

    let event =
        accounting::instantiate(deps, init.into()).context("failed to instantiate accounting")?;

    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("owner", info.sender)
        .add_event(event))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut<WormholeQuery>, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut<WormholeQuery>,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, AnyError> {
    match msg {
        ExecuteMsg::SubmitObservations {
            observations,
            guardian_set_index,
            signature,
        } => submit_observations(deps, info, observations, guardian_set_index, signature),
        ExecuteMsg::ModifyBalance {
            modification,
            guardian_set_index,
            signatures,
        } => modify_balance(deps, info, modification, guardian_set_index, signatures),
        ExecuteMsg::UpgradeContract {
            upgrade,
            guardian_set_index,
            signatures,
        } => upgrade_contract(deps, env, info, upgrade, guardian_set_index, signatures),
    }
}

fn submit_observations(
    mut deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    observations: Binary,
    guardian_set_index: u32,
    signature: Signature,
) -> Result<Response, AnyError> {
    deps.querier
        .query::<Empty>(
            &WormholeQuery::VerifySignature {
                data: observations.clone(),
                guardian_set_index,
                signature,
            }
            .into(),
        )
        .context("failed to verify signature")?;

    let quorum = deps
        .querier
        .query::<u32>(&WormholeQuery::CalculateQuorum { guardian_set_index }.into())
        .and_then(|q| {
            usize::try_from(q).map_err(|_| StdError::ConversionOverflow {
                source: ConversionOverflowError::new("u32", "usize", q.to_string()),
            })
        })
        .context("failed to calculate quorum")?;

    let observations: Vec<Observation> =
        from_binary(&observations).context("failed to parse `Observations`")?;

    let events = observations
        .into_iter()
        .map(|o| handle_observation(deps.branch(), o, guardian_set_index, quorum, signature))
        .filter_map(Result::transpose)
        .collect::<anyhow::Result<Vec<_>>>()
        .context("failed to handle `Observation`")?;

    Ok(Response::new()
        .add_attribute("action", "submit_observations")
        .add_attribute("owner", info.sender)
        .add_events(events))
}

fn handle_observation(
    mut deps: DepsMut<WormholeQuery>,
    o: Observation,
    guardian_set_index: u32,
    quorum: usize,
    sig: Signature,
) -> anyhow::Result<Option<Event>> {
    if accounting::has_transfer(deps.as_ref(), o.key.clone()) {
        bail!("transfer for key \"{}\" already committed", o.key);
    }

    let key = PENDING_TRANSFERS.key(o.key.clone());
    let mut pending = key
        .may_load(deps.storage)
        .map(Option::unwrap_or_default)
        .context("failed to load `PendingTransfer`")?;
    let data = match pending
        .iter_mut()
        .find(|d| d.guardian_set_index() == guardian_set_index && d.observation() == &o)
    {
        Some(d) => d,
        None => {
            pending.push(Data::new(o.clone(), guardian_set_index));
            let back = pending.len() - 1;
            &mut pending[back]
        }
    };

    data.add_signature(sig)?;

    if data.signatures().len() < quorum {
        // Still need more signatures so just save the pending transfer data and exit.
        key.save(deps.storage, &pending)
            .context("failed to save pending transfers")?;

        return Ok(None);
    }

    let (msg, _) = serde_wormhole::from_slice_with_payload(&o.payload)
        .context("failed to parse observation payload")?;
    let tx_data = match msg {
        Message::Transfer {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        }
        | Message::TransferWithPayload {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        } => transfer::Data {
            amount: Uint256::from_be_bytes(amount.0),
            token_address: TokenAddress::new(token_address.0),
            token_chain: token_chain.into(),
            recipient_chain: recipient_chain.into(),
        },
        _ => bail!("Unknown tokenbridge payload"),
    };

    let emitter_chain = o.key.emitter_chain();

    let tokenbridge_addr = TOKENBRIDGE_ADDR
        .load(deps.storage)
        .context("failed to load tokenbridge addr")?;

    let registered_emitter: Vec<u8> = deps
        .querier
        .query_wasm_smart(
            tokenbridge_addr,
            &tokenbridge::msg::QueryMsg::ChainRegistration {
                chain: emitter_chain,
            },
        )
        .context("failed to query chain registration")?;
    ensure!(
        *registered_emitter == **o.key.emitter_address(),
        "unknown emitter address"
    );

    accounting::commit_transfer(
        deps.branch(),
        Transfer {
            key: o.key.clone(),
            data: tx_data,
        },
    )
    .context("failed to commit transfer")?;

    // Now that the transfer has been committed, we don't need to keep it in the pending list.
    key.remove(deps.storage);

    Ok(Some(
        Event::new("Transfer")
            .add_attribute("emitter_chain", o.key.emitter_chain().to_string())
            .add_attribute("emitter_address", o.key.emitter_address().to_string())
            .add_attribute("sequence", o.key.sequence().to_string())
            .add_attribute("nonce", o.nonce.to_string())
            .add_attribute("tx_hash", o.tx_hash.to_base64())
            .add_attribute("payload", o.payload.to_base64()),
    ))
}

fn modify_balance(
    deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    modification: Binary,
    guardian_set_index: u32,
    signatures: Vec<Signature>,
) -> Result<Response, AnyError> {
    deps.querier
        .query::<Empty>(
            &WormholeQuery::VerifyQuorum {
                data: modification.clone(),
                guardian_set_index,
                signatures: signatures.into_iter().map(From::from).collect(),
            }
            .into(),
        )
        .context(ContractError::VerifyQuorum)?;

    let msg: Modification = from_binary(&modification).context("failed to parse `Modification`")?;

    let event =
        accounting::modify_balance(deps, msg).context("failed to modify account balance")?;

    Ok(Response::new()
        .add_attribute("action", "modify_balance")
        .add_attribute("owner", info.sender)
        .add_event(event))
}

fn upgrade_contract(
    deps: DepsMut<WormholeQuery>,
    env: Env,
    info: MessageInfo,
    upgrade: Binary,
    guardian_set_index: u32,
    signatures: Vec<Signature>,
) -> Result<Response, AnyError> {
    deps.querier
        .query::<Empty>(
            &WormholeQuery::VerifyQuorum {
                data: upgrade.clone(),
                guardian_set_index,
                signatures: signatures.into_iter().map(From::from).collect(),
            }
            .into(),
        )
        .context(ContractError::VerifyQuorum)?;

    let Upgrade { new_addr } = from_binary(&upgrade).context("failed to parse `Upgrade`")?;

    let mut buf = 0u64.to_ne_bytes();
    buf.copy_from_slice(&new_addr[24..]);
    let new_contract = u64::from_be_bytes(buf);

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(WasmMsg::Migrate {
            contract_addr: env.contract.address.to_string(),
            new_code_id: new_contract,
            msg: to_binary(&MigrateMsg {})?,
        }))
        .add_attribute("action", "contract_upgrade")
        .add_attribute("owner", info.sender))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps<WormholeQuery>, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Balance(key) => query_balance(deps, key).and_then(|resp| to_binary(&resp)),
        QueryMsg::AllAccounts { start_after, limit } => {
            query_all_accounts(deps, start_after, limit).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::Transfer(req) => query_transfer(deps, req).and_then(|resp| to_binary(&resp)),
        QueryMsg::AllTransfers { start_after, limit } => {
            query_all_transfers(deps, start_after, limit).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::PendingTransfer(req) => {
            query_pending_transfer(deps, req).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::AllPendingTransfers { start_after, limit } => {
            query_all_pending_transfers(deps, start_after, limit).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::Modification { sequence } => {
            query_modification(deps, sequence).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::AllModifications { start_after, limit } => {
            query_all_modifications(deps, start_after, limit).and_then(|resp| to_binary(&resp))
        }
        QueryMsg::ValidateTransfer { transfer } => validate_transfer(deps, &transfer)
            .map_err(|e| {
                e.downcast().unwrap_or_else(|e| StdError::GenericErr {
                    msg: format!("{e:#}"),
                })
            })
            .and_then(|()| to_binary(&Empty {})),
    }
}

fn query_all_accounts(
    deps: Deps<WormholeQuery>,
    start_after: Option<account::Key>,
    limit: Option<u32>,
) -> StdResult<AllAccountsResponse> {
    if let Some(lim) = limit {
        let l = lim
            .try_into()
            .map_err(|_| ConversionOverflowError::new("u32", "usize", lim.to_string()))?;
        accounting::query_all_accounts(deps, start_after)
            .take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|accounts| AllAccountsResponse { accounts })
    } else {
        accounting::query_all_accounts(deps, start_after)
            .collect::<StdResult<Vec<_>>>()
            .map(|accounts| AllAccountsResponse { accounts })
    }
}

fn query_all_transfers(
    deps: Deps<WormholeQuery>,
    start_after: Option<transfer::Key>,
    limit: Option<u32>,
) -> StdResult<AllTransfersResponse> {
    if let Some(lim) = limit {
        let l = lim
            .try_into()
            .map_err(|_| ConversionOverflowError::new("u32", "usize", lim.to_string()))?;
        accounting::query_all_transfers(deps, start_after)
            .take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|transfers| AllTransfersResponse { transfers })
    } else {
        accounting::query_all_transfers(deps, start_after)
            .collect::<StdResult<Vec<_>>>()
            .map(|transfers| AllTransfersResponse { transfers })
    }
}

#[inline]
fn tinyvec_to_vec<A: Array>(tv: TinyVec<A>) -> Vec<A::Item> {
    match tv {
        TinyVec::Inline(mut arr) => arr.drain_to_vec(),
        TinyVec::Heap(v) => v,
    }
}

fn query_pending_transfer(
    deps: Deps<WormholeQuery>,
    key: transfer::Key,
) -> StdResult<Vec<state::Data>> {
    PENDING_TRANSFERS
        .load(deps.storage, key)
        .map(tinyvec_to_vec)
}

fn query_all_pending_transfers(
    deps: Deps<WormholeQuery>,
    start_after: Option<transfer::Key>,
    limit: Option<u32>,
) -> StdResult<AllPendingTransfersResponse> {
    let start = start_after.map(|key| Bound::Exclusive((key, PhantomData)));

    let iter = PENDING_TRANSFERS
        .range(deps.storage, start, None, Order::Ascending)
        .map(|item| {
            item.map(|(key, tv)| PendingTransfer {
                key,
                data: tinyvec_to_vec(tv),
            })
        });

    if let Some(lim) = limit {
        let l = lim
            .try_into()
            .map_err(|_| ConversionOverflowError::new("u32", "usize", lim.to_string()))?;
        iter.take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|pending| AllPendingTransfersResponse { pending })
    } else {
        iter.collect::<StdResult<Vec<_>>>()
            .map(|pending| AllPendingTransfersResponse { pending })
    }
}

fn query_all_modifications(
    deps: Deps<WormholeQuery>,
    start_after: Option<u64>,
    limit: Option<u32>,
) -> StdResult<AllModificationsResponse> {
    if let Some(lim) = limit {
        let l = lim
            .try_into()
            .map_err(|_| ConversionOverflowError::new("u32", "usize", lim.to_string()))?;
        accounting::query_all_modifications(deps, start_after)
            .take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|modifications| AllModificationsResponse { modifications })
    } else {
        accounting::query_all_modifications(deps, start_after)
            .collect::<StdResult<Vec<_>>>()
            .map(|modifications| AllModificationsResponse { modifications })
    }
}
