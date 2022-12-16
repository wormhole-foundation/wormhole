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
use wormhole::{
    token::{Action, GovernancePacket, Message},
    vaa::{self, Body, Header, Signature},
    Address, Chain,
};
use wormhole_bindings::WormholeQuery;

use crate::{
    bail,
    error::{AnyError, ContractError},
    msg::{
        AllAccountsResponse, AllModificationsResponse, AllPendingTransfersResponse,
        AllTransfersResponse, ChainRegistrationResponse, ExecuteMsg, Instantiate, InstantiateMsg,
        MigrateMsg, Observation, QueryMsg, Upgrade,
    },
    state::{self, Data, PendingTransfer, CHAIN_REGISTRATIONS, DIGESTS, PENDING_TRANSFERS},
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
        ExecuteMsg::SubmitVAAs { vaas } => submit_vaas(deps, info, vaas),
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
    let digest_key = DIGESTS.key((o.emitter_chain, o.emitter_address.to_vec(), o.sequence));
    if let Some(saved_digest) = digest_key
        .may_load(deps.storage)
        .context("failed to load transfer digest")?
    {
        let digest = o.digest().context(ContractError::ObservationDigest)?;
        if saved_digest != digest {
            bail!(ContractError::DigestMismatch);
        }

        bail!(ContractError::DuplicateMessage);
    }

    let tx_key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);

    let key = PENDING_TRANSFERS.key(tx_key.clone());
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

    let registered_emitter = CHAIN_REGISTRATIONS
        .may_load(deps.storage, o.emitter_chain)
        .context("failed to load chain registration")?
        .ok_or_else(|| ContractError::MissingChainRegistration(o.emitter_chain.into()))?;

    ensure!(
        *registered_emitter == o.emitter_address,
        "unknown emitter address"
    );

    accounting::commit_transfer(
        deps.branch(),
        Transfer {
            key: tx_key,
            data: tx_data,
        },
    )
    .context("failed to commit transfer")?;

    // Save the digest of the observation so that we can check for duplicate transfer keys with
    // mismatched data.
    let digest = o.digest().context(ContractError::ObservationDigest)?;
    digest_key
        .save(deps.storage, &digest)
        .context("failed to save transfer digest")?;

    // Now that the transfer has been committed, we don't need to keep it in the pending list.
    key.remove(deps.storage);

    Ok(Some(
        Event::new("Transfer")
            .add_attribute("tx_hash", o.tx_hash.to_base64())
            .add_attribute("timestamp", o.timestamp.to_string())
            .add_attribute("nonce", o.nonce.to_string())
            .add_attribute("emitter_chain", o.emitter_chain.to_string())
            .add_attribute("emitter_address", Address(o.emitter_address).to_string())
            .add_attribute("sequence", o.sequence.to_string())
            .add_attribute("consistency_level", o.consistency_level.to_string())
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

fn submit_vaas(
    mut deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    vaas: Vec<Binary>,
) -> Result<Response, AnyError> {
    let evts = vaas
        .into_iter()
        .map(|v| handle_vaa(deps.branch(), v))
        .collect::<anyhow::Result<Vec<_>>>()?;
    Ok(Response::new()
        .add_attribute("action", "submit_vaas")
        .add_attribute("owner", info.sender)
        .add_events(evts))
}

fn handle_vaa(mut deps: DepsMut<WormholeQuery>, vaa: Binary) -> anyhow::Result<Event> {
    let (header, data) = serde_wormhole::from_slice_with_payload::<Header>(&vaa)
        .context("failed to parse VAA header")?;

    ensure!(header.version == 1, "unsupported VAA version");

    deps.querier
        .query::<Empty>(
            &WormholeQuery::VerifyQuorum {
                data: data.to_vec().into(),
                guardian_set_index: header.guardian_set_index,
                signatures: header.signatures,
            }
            .into(),
        )
        .context(ContractError::VerifyQuorum)?;

    let digest = vaa::digest(data)
        .map(|d| d.secp256k_hash.to_vec().into())
        .context("failed to calculate digest for VAA body")?;

    let (body, payload) = serde_wormhole::from_slice_with_payload::<Body<()>>(data)
        .context("failed to parse VAA body")?;

    let digest_key = DIGESTS.key((
        body.emitter_chain.into(),
        body.emitter_address.0.to_vec(),
        body.sequence,
    ));

    if let Some(saved_digest) = digest_key
        .may_load(deps.storage)
        .context("failed to load transfer digest")?
    {
        if saved_digest != digest {
            bail!(ContractError::DigestMismatch);
        }

        bail!(ContractError::DuplicateMessage);
    }

    let evt = if body.emitter_chain == Chain::Solana
        && body.emitter_address == wormhole::GOVERNANCE_EMITTER
    {
        let govpacket =
            serde_wormhole::from_slice(payload).context("failed to parse governance packet")?;
        handle_governance_vaa(deps.branch(), body.with_payload(govpacket))?
    } else {
        let (msg, _) = serde_wormhole::from_slice_with_payload(payload)
            .context("failed to parse tokenbridge message")?;
        handle_tokenbridge_vaa(deps.branch(), body.with_payload(msg))?
    };

    digest_key
        .save(deps.storage, &digest)
        .context("failed to save message digest")?;

    Ok(evt)
}

fn handle_governance_vaa(
    deps: DepsMut<WormholeQuery>,
    body: Body<GovernancePacket>,
) -> anyhow::Result<Event> {
    ensure!(
        body.payload.chain == Chain::Any || body.payload.chain == Chain::Wormchain,
        "this governance VAA is for another chain"
    );

    match body.payload.action {
        Action::RegisterChain {
            chain,
            emitter_address,
        } => {
            CHAIN_REGISTRATIONS
                .save(
                    deps.storage,
                    chain.into(),
                    &emitter_address.0.to_vec().into(),
                )
                .context("failed to save chain registration")?;
            Ok(Event::new("RegisterChain")
                .add_attribute("chain", chain.to_string())
                .add_attribute("emitter_address", emitter_address.to_string()))
        }
        _ => bail!("unsupported governance action"),
    }
}

fn handle_tokenbridge_vaa(
    mut deps: DepsMut<WormholeQuery>,
    body: Body<Message>,
) -> anyhow::Result<Event> {
    let data = match body.payload {
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

    let key = transfer::Key::new(
        body.emitter_chain.into(),
        TokenAddress::new(body.emitter_address.0),
        body.sequence,
    );

    let tx = Transfer {
        key: key.clone(),
        data,
    };
    let evt = accounting::commit_transfer(deps.branch(), tx)
        .with_context(|| format!("failed to commit transfer for key {key}"))?;

    PENDING_TRANSFERS.remove(deps.storage, key);

    Ok(evt)
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
        QueryMsg::ChainRegistration { chain } => {
            query_chain_registration(deps, chain).and_then(|resp| to_binary(&resp))
        }
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

fn query_chain_registration(
    deps: Deps<WormholeQuery>,
    chain: u16,
) -> StdResult<ChainRegistrationResponse> {
    CHAIN_REGISTRATIONS
        .load(deps.storage, chain)
        .map(|address| ChainRegistrationResponse { address })
}
