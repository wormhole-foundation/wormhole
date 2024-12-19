use std::marker::PhantomData;

use accountant::{
    query_balance, query_modification,
    state::{account, transfer, Kind, Modification, TokenAddress, Transfer},
    validate_transfer,
};
use anyhow::{ensure, Context};
#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    from_json, to_json_binary, Binary, ConversionOverflowError, Deps, DepsMut, Empty, Env, Event,
    MessageInfo, Order, Response, StdError, StdResult, Uint256,
};
use cw2::set_contract_version;
use cw_storage_plus::Bound;
use serde_wormhole::RawMessage;
use tinyvec::{Array, TinyVec};
use wormhole_bindings::WormholeQuery;
use wormhole_sdk::{
    accountant as accountant_module,
    accountant_modification::ModificationKind,
    token,
    vaa::{self, Body, Header, Signature},
    Chain,
};

use crate::{
    bail,
    error::{AnyError, ContractError},
    msg::{
        AllAccountsResponse, AllModificationsResponse, AllPendingTransfersResponse,
        AllTransfersResponse, BatchTransferStatusResponse, ChainRegistrationResponse, ExecuteMsg,
        MigrateMsg, MissingObservation, MissingObservationsResponse, Observation, ObservationError,
        ObservationStatus, QueryMsg, SubmitObservationResponse, TransferDetails, TransferStatus,
        SUBMITTED_OBSERVATIONS_PREFIX,
    },
    state::{Data, PendingTransfer, CHAIN_REGISTRATIONS, DIGESTS, PENDING_TRANSFERS},
};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:global-accountant";
const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut<WormholeQuery>,
    _env: Env,
    info: MessageInfo,
    _msg: Empty,
) -> Result<Response, AnyError> {
    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)
        .context("failed to set contract version")?;

    Ok(Response::new()
        .add_attribute("action", "instantiate")
        .add_attribute("owner", info.sender)
        .add_attribute("version", CONTRACT_VERSION))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut<WormholeQuery>, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut<WormholeQuery>,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, AnyError> {
    match msg {
        ExecuteMsg::SubmitObservations {
            observations,
            guardian_set_index,
            signature,
        } => submit_observations(deps, info, observations, guardian_set_index, signature),

        ExecuteMsg::SubmitVaas { vaas } => submit_vaas(deps, info, vaas),
    }
}

fn submit_observations(
    mut deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    observations: Binary,
    guardian_set_index: u32,
    signature: Signature,
) -> Result<Response, AnyError> {
    // We need to prepend an observation prefix to `observations`, which is the
    // same prefix used by the guardians to sign these observations. This
    // prefix specifies this type as global accountant observations.

    deps.querier
        .query::<Empty>(
            &WormholeQuery::VerifyMessageSignature {
                prefix: SUBMITTED_OBSERVATIONS_PREFIX.into(),
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
        .context("failed to calculate quorum")?;

    let observations: Vec<Observation> =
        from_json(&observations).context("failed to parse `Observations`")?;

    let mut responses = Vec::with_capacity(observations.len());
    let mut events = Vec::with_capacity(observations.len());
    for o in observations {
        let key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
        match handle_observation(deps.branch(), o, guardian_set_index, quorum, signature) {
            Ok((status, event)) => {
                responses.push(SubmitObservationResponse { key, status });
                if let Some(evt) = event {
                    events.push(evt);
                }
            }
            Err(e) => {
                let err = ObservationError {
                    key,
                    error: format!("{e:#}"),
                };
                let evt = cw_transcode::to_event(&err)
                    .context("failed to transcode observation error")?;
                events.push(evt);
                responses.push(SubmitObservationResponse {
                    key: err.key,
                    status: ObservationStatus::Error(err.error),
                });
            }
        }
    }

    let data = to_json_binary(&responses).context("failed to serialize transfer details")?;

    Ok(Response::new()
        .add_attribute("action", "submit_observations")
        .add_attribute("owner", info.sender)
        .set_data(data)
        .add_events(events))
}

fn handle_observation(
    mut deps: DepsMut<WormholeQuery>,
    o: Observation,
    guardian_set_index: u32,
    quorum: u32,
    sig: Signature,
) -> anyhow::Result<(ObservationStatus, Option<Event>)> {
    let registered_emitter = CHAIN_REGISTRATIONS
        .may_load(deps.storage, o.emitter_chain)
        .context("failed to load chain registration")?
        .ok_or_else(|| ContractError::MissingChainRegistration(o.emitter_chain.into()))?;

    ensure!(
        *registered_emitter == o.emitter_address,
        "unknown emitter address"
    );

    let digest = o.digest().context(ContractError::ObservationDigest)?;

    let digest_key = DIGESTS.key((o.emitter_chain, o.emitter_address.to_vec(), o.sequence));
    let tx_key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);

    if let Some(saved_digest) = digest_key
        .may_load(deps.storage)
        .context("failed to load transfer digest")?
    {
        if saved_digest != digest {
            bail!(ContractError::DigestMismatch);
        }

        return Ok((ObservationStatus::Committed, None));
    }

    let key = PENDING_TRANSFERS.key(tx_key.clone());
    let mut pending = key
        .may_load(deps.storage)
        .map(Option::unwrap_or_default)
        .context("failed to load `PendingTransfer`")?;
    let data = match pending.iter_mut().find(|d| {
        d.guardian_set_index() == guardian_set_index
            && d.digest() == &digest
            && d.tx_hash() == &o.tx_hash
    }) {
        Some(d) => d,
        None => {
            pending.push(Data::new(
                digest.clone(),
                o.tx_hash.clone(),
                o.emitter_chain,
                guardian_set_index,
            ));
            let back = pending.len() - 1;
            &mut pending[back]
        }
    };

    data.add_signature(sig.index);

    if data.num_signatures() < quorum {
        // Still need more signatures so just save the pending transfer data and exit.
        key.save(deps.storage, &pending)
            .context("failed to save pending transfers")?;

        return Ok((ObservationStatus::Pending, None));
    }

    let msg = serde_wormhole::from_slice::<token::Message<&RawMessage>>(&o.payload)
        .context("failed to parse observation payload")?;
    let tx_data = match msg {
        token::Message::Transfer {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        }
        | token::Message::TransferWithPayload {
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

    accountant::commit_transfer(
        deps.branch(),
        Transfer {
            key: tx_key,
            data: tx_data,
        },
    )
    .context("failed to commit transfer")?;

    // Save the digest of the observation so that we can check for duplicate transfer keys with
    // mismatched data.
    digest_key
        .save(deps.storage, &digest)
        .context("failed to save transfer digest")?;

    // Now that the transfer has been committed, we don't need to keep it in the pending list.
    key.remove(deps.storage);

    let event = cw_transcode::to_event(&o)
        .map(Some)
        .context("failed to transcode `Observation` to `Event`")?;

    Ok((ObservationStatus::Committed, event))
}

fn modify_balance(
    deps: DepsMut<WormholeQuery>,
    info: &MessageInfo,
    modification: Modification,
) -> Result<Event, AnyError> {
    let mut event = accountant::modify_balance(deps, modification)
        .context("failed to modify account balance")?;

    event = event
        .add_attribute("action", "modify_balance")
        .add_attribute("owner", info.sender.clone());

    Ok(event)
}

fn submit_vaas(
    mut deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    vaas: Vec<Binary>,
) -> Result<Response, AnyError> {
    let evts = vaas
        .into_iter()
        .map(|v| handle_vaa(deps.branch(), &info, v))
        .collect::<anyhow::Result<Vec<_>>>()?;
    Ok(Response::new()
        .add_attribute("action", "submit_vaas")
        .add_attribute("owner", info.sender)
        .add_events(evts))
}

fn handle_vaa(
    mut deps: DepsMut<WormholeQuery>,
    info: &MessageInfo,
    vaa: Binary,
) -> anyhow::Result<Event> {
    let (header, data) = serde_wormhole::from_slice::<(Header, &RawMessage)>(&vaa)
        .context("failed to parse VAA header")?;

    ensure!(header.version == 1, "unsupported VAA version");

    deps.querier
        .query::<Empty>(&WormholeQuery::VerifyVaa { vaa: vaa.clone() }.into())
        .context(ContractError::VerifyQuorum)?;

    let digest = vaa::digest(data)
        .map(|d| d.secp256k_hash.to_vec().into())
        .context("failed to calculate digest for VAA body")?;

    let body = serde_wormhole::from_slice::<Body<&RawMessage>>(data)
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

    // We may also accept governance messages from wormchain in the future
    let mut evt = if body.emitter_chain == Chain::Solana
        && body.emitter_address == wormhole_sdk::GOVERNANCE_EMITTER
    {
        if body.payload.len() < 32 {
            bail!("governance module missing");
        }
        let module = &body.payload[..32];

        if module == token::MODULE {
            let govpacket = serde_wormhole::from_slice(body.payload)
                .context("failed to parse tokenbridge governance packet")?;
            handle_token_governance_vaa(deps.branch(), body.with_payload(govpacket))?
        } else if module == accountant_module::MODULE {
            let govpacket = serde_wormhole::from_slice(body.payload)
                .context("failed to parse accountant governance packet")?;
            handle_accountant_governance_vaa(deps.branch(), info, body.with_payload(govpacket))?
        } else {
            bail!("unknown governance module")
        }
    } else {
        let msg = serde_wormhole::from_slice(body.payload)
            .context("failed to parse tokenbridge message")?;
        handle_tokenbridge_vaa(deps.branch(), body.with_payload(msg))?
    };

    digest_key
        .save(deps.storage, &digest)
        .context("failed to save message digest")?;

    evt = evt.add_attribute("vaa_digest", hex::encode(digest.as_slice()));

    Ok(evt)
}

fn handle_token_governance_vaa(
    deps: DepsMut<WormholeQuery>,
    body: Body<token::GovernancePacket>,
) -> anyhow::Result<Event> {
    ensure!(
        body.payload.chain == Chain::Any || body.payload.chain == Chain::Wormchain,
        "this token governance VAA is for another chain"
    );

    match body.payload.action {
        token::Action::RegisterChain {
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

fn handle_accountant_governance_vaa(
    deps: DepsMut<WormholeQuery>,
    info: &MessageInfo,
    body: Body<accountant_module::GovernancePacket>,
) -> anyhow::Result<Event> {
    ensure!(
        body.payload.chain == Chain::Wormchain,
        "this accountant governance VAA is for another chain"
    );

    match body.payload.action {
        accountant_module::Action::ModifyBalance {
            sequence,
            chain_id,
            token_chain,
            token_address,
            kind,
            amount,
            reason,
        } => {
            let token_address = TokenAddress::new(token_address.0);
            let kind = match kind {
                ModificationKind::Add => Kind::Add,
                ModificationKind::Subtract => Kind::Sub,
                ModificationKind::Unknown => {
                    bail!("unsupported governance action")
                }
            };
            let amount = Uint256::from_be_bytes(amount.0);
            let modification = Modification {
                sequence,
                chain_id,
                token_chain,
                token_address,
                kind,
                amount,
                reason: reason.to_string(),
            };
            modify_balance(deps, info, modification).map_err(|e| e.into())
        }
    }
}

fn handle_tokenbridge_vaa(
    mut deps: DepsMut<WormholeQuery>,
    body: Body<token::Message<&RawMessage>>,
) -> anyhow::Result<Event> {
    let registered_emitter = CHAIN_REGISTRATIONS
        .may_load(deps.storage, body.emitter_chain.into())
        .context("failed to load chain registration")?
        .ok_or(ContractError::MissingChainRegistration(body.emitter_chain))?;

    ensure!(
        *registered_emitter == body.emitter_address.0,
        "unknown emitter address"
    );

    let data = match body.payload {
        token::Message::Transfer {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        }
        | token::Message::TransferWithPayload {
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
    let evt = accountant::commit_transfer(deps.branch(), tx)
        .with_context(|| format!("failed to commit transfer for key {key}"))?;

    PENDING_TRANSFERS.remove(deps.storage, key);

    Ok(evt)
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps<WormholeQuery>, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Balance(key) => query_balance(deps, key).and_then(|resp| to_json_binary(&resp)),
        QueryMsg::AllAccounts { start_after, limit } => {
            query_all_accounts(deps, start_after, limit).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::AllTransfers { start_after, limit } => {
            query_all_transfers(deps, start_after, limit).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::AllPendingTransfers { start_after, limit } => {
            query_all_pending_transfers(deps, start_after, limit)
                .and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::Modification { sequence } => {
            query_modification(deps, sequence).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::AllModifications { start_after, limit } => {
            query_all_modifications(deps, start_after, limit).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::ValidateTransfer { transfer } => validate_transfer(deps, &transfer)
            .map_err(|e| {
                e.downcast().unwrap_or_else(|e| StdError::GenericErr {
                    msg: format!("{e:#}"),
                })
            })
            .and_then(|()| to_json_binary(&Empty {})),
        QueryMsg::ChainRegistration { chain } => {
            query_chain_registration(deps, chain).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::MissingObservations {
            guardian_set,
            index,
        } => query_missing_observations(deps, guardian_set, index)
            .and_then(|resp| to_json_binary(&resp)),
        QueryMsg::TransferStatus(key) => {
            query_transfer_status(deps, &key).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::BatchTransferStatus(keys) => {
            query_batch_transfer_status(deps, keys).and_then(|resp| to_json_binary(&resp))
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
        accountant::query_all_accounts(deps, start_after)
            .take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|accounts| AllAccountsResponse { accounts })
    } else {
        accountant::query_all_accounts(deps, start_after)
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
        accountant::query_all_transfers(deps, start_after)
            .map(|res| {
                res.and_then(|t| {
                    let digest = DIGESTS.load(
                        deps.storage,
                        (
                            t.key.emitter_chain(),
                            t.key.emitter_address().to_vec(),
                            t.key.sequence(),
                        ),
                    )?;

                    Ok((t, digest))
                })
            })
            .take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|transfers| AllTransfersResponse { transfers })
    } else {
        accountant::query_all_transfers(deps, start_after)
            .map(|res| {
                res.and_then(|t| {
                    let digest = DIGESTS.load(
                        deps.storage,
                        (
                            t.key.emitter_chain(),
                            t.key.emitter_address().to_vec(),
                            t.key.sequence(),
                        ),
                    )?;

                    Ok((t, digest))
                })
            })
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
        accountant::query_all_modifications(deps, start_after)
            .take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|modifications| AllModificationsResponse { modifications })
    } else {
        accountant::query_all_modifications(deps, start_after)
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

fn query_missing_observations(
    deps: Deps<WormholeQuery>,
    guardian_set: u32,
    index: u8,
) -> StdResult<MissingObservationsResponse> {
    let mut missing = Vec::new();
    for pending in PENDING_TRANSFERS.range(deps.storage, None, None, Order::Ascending) {
        let (_, v) = pending?;
        for data in v {
            if data.guardian_set_index() == guardian_set && !data.has_signature(index) {
                missing.push(MissingObservation {
                    chain_id: data.emitter_chain(),
                    tx_hash: data.tx_hash().clone(),
                });
            }
        }
    }

    Ok(MissingObservationsResponse { missing })
}

fn query_transfer_status(
    deps: Deps<WormholeQuery>,
    key: &transfer::Key,
) -> StdResult<TransferStatus> {
    if let Some(digest) = DIGESTS.may_load(
        deps.storage,
        (
            key.emitter_chain(),
            key.emitter_address().to_vec(),
            key.sequence(),
        ),
    )? {
        let data = accountant::query_transfer(deps, key.clone())?;
        Ok(TransferStatus::Committed { data, digest })
    } else if let Some(data) = PENDING_TRANSFERS.may_load(deps.storage, key.clone())? {
        Ok(TransferStatus::Pending(tinyvec_to_vec(data)))
    } else {
        Err(StdError::not_found(format!("transfer with key {key}")))
    }
}

fn query_batch_transfer_status(
    deps: Deps<WormholeQuery>,
    keys: Vec<transfer::Key>,
) -> StdResult<BatchTransferStatusResponse> {
    keys.into_iter()
        .map(|key| {
            let status = match query_transfer_status(deps, &key) {
                Ok(s) => Some(s),
                Err(StdError::NotFound { .. }) => None,
                Err(e) => return Err(e),
            };
            Ok(TransferDetails { key, status })
        })
        .collect::<StdResult<Vec<_>>>()
        .map(|details| BatchTransferStatusResponse { details })
}
