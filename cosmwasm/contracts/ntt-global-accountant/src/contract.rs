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
use cw_storage_plus::{Bound, KeyDeserialize};
use ntt_messages::{
    mode::Mode,
    ntt::NativeTokenTransfer,
    transceiver::{Transceiver, TransceiverMessage},
    transceivers::wormhole::{
        WormholeTransceiver, WormholeTransceiverInfo, WormholeTransceiverRegistration,
    },
    trimmed_amount::{TrimmedAmount, TRIMMED_DECIMALS},
};
use serde_wormhole::RawMessage;
use tinyvec::{Array, TinyVec};
use wormhole_bindings::WormholeQuery;
use wormhole_io::TypePrefixedPayload;
use wormhole_sdk::{
    accountant_modification::ModificationKind,
    ntt_accountant as ntt_accountant_module, relayer,
    vaa::{self, Body, Header, Signature},
    Chain,
};

use crate::{
    bail,
    error::{AnyError, ContractError},
    msg::{
        AllAccountsResponse, AllModificationsResponse, AllPendingTransfersResponse,
        AllTransceiverHubsResponse, AllTransceiverPeersResponse, AllTransfersResponse,
        BatchTransferStatusResponse, ExecuteMsg, MigrateMsg, MissingObservation,
        MissingObservationsResponse, Observation, ObservationError, ObservationStatus, QueryMsg,
        RelayerChainRegistrationResponse, SubmitObservationResponse, TransferDetails,
        TransferStatus, SUBMITTED_OBSERVATIONS_PREFIX,
    },
    state::{
        Data, PendingTransfer, TransceiverHub, TransceiverPeer, DIGESTS, PENDING_TRANSFERS,
        RELAYER_CHAIN_REGISTRATIONS, TRANSCEIVER_PEER, TRANSCEIVER_TO_HUB,
    },
    structs::DeliveryInstruction,
};

// version info for migration info
const CONTRACT_NAME: &str = "crates.io:ntt-global-accountant";
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
        // this key is for the VAA, which is how the guardian is tracking messages
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
    let relayer_emitter = RELAYER_CHAIN_REGISTRATIONS.may_load(deps.storage, o.emitter_chain)?;

    let digest = o.digest().context(ContractError::ObservationDigest)?;
    let event = cw_transcode::to_event(&o)
        .map(Some)
        .context("failed to transcode `Observation` to `Event`")?;

    let (sender, payload) =
        if relayer_emitter.is_some_and(|relayer_address| relayer_address == o.emitter_address) {
            // if the emitter is a known standard relayer, parse the sender and payload from the delivery instruction
            let delivery_instruction = DeliveryInstruction::deserialize(&o.payload.0)?;
            (
                delivery_instruction.sender_address.into(),
                delivery_instruction.payload,
            )
        } else {
            // otherwise, the sender and payload is the same as the VAA
            (o.emitter_address.into(), o.payload.0)
        };

    let hub = TRANSCEIVER_TO_HUB
        .load(deps.storage, (o.emitter_chain, sender))
        .map_err(|_| ContractError::MissingHubRegistration)?;

    let message: TransceiverMessage<WormholeTransceiver, NativeTokenTransfer> =
        TypePrefixedPayload::read_payload(&mut payload.as_slice())
            .context("failed to parse observation payload")?;

    let destination_chain = message.ntt_manager_payload.payload.to_chain.id;
    let source_peer = TRANSCEIVER_PEER
        .load(deps.storage, (o.emitter_chain, sender, destination_chain))
        .map_err(|_| ContractError::MissingSourcePeerRegistration(destination_chain.into()))?;
    let destination_peer = TRANSCEIVER_PEER
        .load(
            deps.storage,
            (destination_chain, source_peer, o.emitter_chain),
        )
        .map_err(|_| ContractError::MissingDestinationPeerRegistration(o.emitter_chain.into()))?;
    if destination_peer != sender {
        // SECURITY:
        // ensure that both peers are cross-registered
        // this prevents a rogue transceiver from registering with and altering the balance of an existing network
        bail!("peers are not cross-registered")
    }

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

    // !IMPORTANT! the amounts are NOT normalized among different chains / tokens,
    // i.e. the same token belonging to the same locking hub, can have message sourced from one chain that uses 4 decimals "normalized",
    // and another that uses 8... this is maxed at 8, but should be actually normalized to 8 for accounting purposes.
    let tx_data = transfer::Data {
        amount: normalize_transfer_amount(message.ntt_manager_payload.payload.amount),
        token_address: hub.1,
        token_chain: hub.0,
        recipient_chain: message.ntt_manager_payload.payload.to_chain.id,
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

        if module == relayer::MODULE {
            let govpacket = serde_wormhole::from_slice(body.payload)
                .context("failed to parse standardized relayer governance packet")?;
            handle_relayer_governance_vaa(deps.branch(), body.with_payload(govpacket))?
        } else if module == ntt_accountant_module::MODULE {
            let govpacket = serde_wormhole::from_slice(body.payload)
                .context("failed to parse accountant governance packet")?;
            handle_accountant_governance_vaa(deps.branch(), info, body.with_payload(govpacket))?
        } else {
            bail!("unknown governance module")
        }
    } else {
        let msg =
            serde_wormhole::from_slice(body.payload).context("failed to parse raw message")?;
        handle_ntt_vaa(deps.branch(), body.with_payload(msg))?
    };

    digest_key
        .save(deps.storage, &digest)
        .context("failed to save message digest")?;

    evt = evt.add_attribute("vaa_digest", hex::encode(digest.as_slice()));

    Ok(evt)
}

fn handle_relayer_governance_vaa(
    deps: DepsMut<WormholeQuery>,
    body: Body<relayer::GovernancePacket>,
) -> anyhow::Result<Event> {
    ensure!(
        body.payload.chain == Chain::Any || body.payload.chain == Chain::Wormchain,
        "this relayer governance VAA is for another chain"
    );

    match body.payload.action {
        relayer::Action::RegisterChain {
            chain,
            emitter_address,
        } => {
            RELAYER_CHAIN_REGISTRATIONS
                .save(
                    deps.storage,
                    chain.into(),
                    &emitter_address.0.to_vec().into(),
                )
                .context("failed to save chain registration")?;
            Ok(Event::new("RegisterRelayer")
                .add_attribute("chain", chain.to_string())
                .add_attribute("emitter_address", emitter_address.to_string()))
        }
        _ => bail!("unsupported governance action"),
    }
}

fn handle_accountant_governance_vaa(
    deps: DepsMut<WormholeQuery>,
    info: &MessageInfo,
    body: Body<ntt_accountant_module::GovernancePacket>,
) -> anyhow::Result<Event> {
    ensure!(
        body.payload.chain == Chain::Wormchain,
        "this accountant governance VAA is for another chain"
    );

    match body.payload.action {
        ntt_accountant_module::Action::ModifyBalance {
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

fn handle_ntt_vaa(
    mut deps: DepsMut<WormholeQuery>,
    body: Body<&RawMessage>,
) -> anyhow::Result<Event> {
    let relayer_emitter =
        RELAYER_CHAIN_REGISTRATIONS.may_load(deps.storage, body.emitter_chain.into())?;

    let (sender, payload) = if relayer_emitter
        .is_some_and(|relayer_address| relayer_address == body.emitter_address.0)
    {
        // if the emitter is a known standard relayer, parse the sender and payload from the delivery instruction
        let delivery_instruction =
            DeliveryInstruction::deserialize(&Vec::from_slice(body.payload)?)?;
        (
            delivery_instruction.sender_address.into(),
            delivery_instruction.payload,
        )
    } else {
        // otherwise, the sender and payload is the same as the VAA
        (
            body.emitter_address.0.into(),
            Vec::from_slice(body.payload)?,
        )
    };

    if payload.len() < 4 {
        bail!("payload prefix missing");
    }
    let prefix = &payload[..4];

    if prefix == WormholeTransceiver::PREFIX {
        let source_chain = body.emitter_chain.into();

        let hub = TRANSCEIVER_TO_HUB
            .load(deps.storage, (source_chain, sender))
            .map_err(|_| ContractError::MissingHubRegistration)?;

        let message: TransceiverMessage<WormholeTransceiver, NativeTokenTransfer> =
            TypePrefixedPayload::read_payload(&mut payload.as_slice())
                .context("failed to parse NTT transfer payload")?;

        let destination_chain = message.ntt_manager_payload.payload.to_chain.id;
        let source_peer = TRANSCEIVER_PEER
            .load(deps.storage, (source_chain, sender, destination_chain))
            .map_err(|_| ContractError::MissingSourcePeerRegistration(destination_chain.into()))?;
        let destination_peer = TRANSCEIVER_PEER
            .load(deps.storage, (destination_chain, source_peer, source_chain))
            .map_err(|_| ContractError::MissingDestinationPeerRegistration(source_chain.into()))?;
        if destination_peer != sender {
            // SECURITY:
            // ensure that both peers are cross-registered
            // this prevents a rogue transceiver from registering with and altering the balance of an existing network
            bail!("peers are not cross-registered")
        }

        // !IMPORTANT! the amounts are NOT normalized among different chains / tokens,
        // i.e. the same token belonging to the same locking hub, can have message sourced from one chain that uses 4 decimals "normalized",
        // and another that uses 8... this is maxed at 8, but should be actually normalized to 8 for accounting purposes.
        let data = transfer::Data {
            amount: normalize_transfer_amount(message.ntt_manager_payload.payload.amount),
            token_address: hub.1,
            token_chain: hub.0,
            recipient_chain: message.ntt_manager_payload.payload.to_chain.id,
        };

        let key = transfer::Key::new(
            source_chain,
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
    } else if prefix == WormholeTransceiver::INFO_PREFIX {
        // only process init messages for locking hubs, setting their hub mapping to themselves
        let message: WormholeTransceiverInfo =
            TypePrefixedPayload::read_payload(&mut payload.as_slice())
                .context("failed to parse NTT info payload")?;
        if message.manager_mode == (Mode::Locking) {
            let chain = body.emitter_chain.into();
            let hub_key = TRANSCEIVER_TO_HUB.key((chain, sender));

            if hub_key
                .may_load(deps.storage)
                .context("failed to load hub")?
                .is_some()
            {
                bail!("hub entry already exists")
            }
            hub_key
                .save(deps.storage, &(chain, sender))
                .context("failed to save hub")?;
            Ok(Event::new("RegisterHub")
                .add_attribute("chain", chain.to_string())
                .add_attribute("emitter_address", hex::encode(sender)))
        } else {
            bail!("ignoring non-locking NTT initialization")
        }
    } else if prefix == WormholeTransceiver::PEER_INFO_PREFIX {
        // for ease of code assurances, all transceivers should register their hub first, followed by other peers
        // this code will only add peers for which it can assure their hubs match so one less key can be loaded on transfers
        let message: WormholeTransceiverRegistration =
            TypePrefixedPayload::read_payload(&mut payload.as_slice())
                .context("failed to parse NTT registration payload")?;

        let peer_hub = TRANSCEIVER_TO_HUB
            .load(
                deps.storage,
                (message.chain_id.id, message.transceiver_address.into()),
            )
            .map_err(|_| ContractError::MissingHubRegistration)?;

        let chain = body.emitter_chain.into();
        let peer_key = TRANSCEIVER_PEER.key((chain, sender, message.chain_id.id));

        if peer_key.may_load(deps.storage)?.is_some() {
            bail!("peer entry for this chain already exists")
        }

        let hub_key = TRANSCEIVER_TO_HUB.key((chain, sender));

        if let Some(transceiver_hub) = hub_key.may_load(deps.storage)? {
            // hubs must match
            if transceiver_hub != peer_hub {
                bail!("peer hub does not match")
            }
        } else {
            // this transceiver does not have a known hub, check if this peer is a hub themselves
            if peer_hub.0 == message.chain_id.id && peer_hub.1 == message.transceiver_address.into()
            {
                // this peer is a hub, so set it as this transceiver's hub
                hub_key
                    .save(deps.storage, &peer_hub.clone())
                    .context("failed to save hub")?;
            } else {
                // this peer is not a hub and we don't want to make indirect assumptions, so do nothing
                bail!("ignoring attempt to register peer before hub")
            }
        }

        peer_key
            .save(deps.storage, &(message.transceiver_address.into()))
            .context("failed to save hub")?;
        Ok(Event::new("RegisterPeer")
            .add_attribute("chain", chain.to_string())
            .add_attribute("emitter_address", hex::encode(sender))
            .add_attribute("transceiver_chain", message.chain_id.id.to_string())
            .add_attribute(
                "transceiver_address",
                hex::encode(message.transceiver_address),
            ))
    } else {
        bail!("unsupported NTT action")
    }
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
        QueryMsg::RelayerChainRegistration { chain } => {
            query_relayer_chain_registration(deps, chain).and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::AllTransceiverHubs { start_after, limit } => {
            query_all_transceiver_hubs(deps, start_after, limit)
                .and_then(|resp| to_json_binary(&resp))
        }
        QueryMsg::AllTransceiverPeers { start_after, limit } => {
            query_all_transceiver_peers(deps, start_after, limit)
                .and_then(|resp| to_json_binary(&resp))
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

fn query_relayer_chain_registration(
    deps: Deps<WormholeQuery>,
    chain: u16,
) -> StdResult<RelayerChainRegistrationResponse> {
    RELAYER_CHAIN_REGISTRATIONS
        .load(deps.storage, chain)
        .map(|address| RelayerChainRegistrationResponse { address })
}

fn query_all_transceiver_hubs(
    deps: Deps<WormholeQuery>,
    start_after: Option<(u16, TokenAddress)>,
    limit: Option<u32>,
) -> StdResult<AllTransceiverHubsResponse> {
    let start: Option<Bound<'_, (u16, TokenAddress)>> =
        start_after.map(|key| Bound::Exclusive((key, PhantomData)));

    let iter = TRANSCEIVER_TO_HUB
        .range(deps.storage, start, None, Order::Ascending)
        .map(|item| item.map(|(key, data)| TransceiverHub { key, data }));

    if let Some(lim) = limit {
        let l = lim
            .try_into()
            .map_err(|_| ConversionOverflowError::new("u32", "usize", lim.to_string()))?;
        iter.take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|hubs| AllTransceiverHubsResponse { hubs })
    } else {
        iter.collect::<StdResult<Vec<_>>>()
            .map(|hubs| AllTransceiverHubsResponse { hubs })
    }
}

fn query_all_transceiver_peers(
    deps: Deps<WormholeQuery>,
    start_after: Option<(u16, TokenAddress, u16)>,
    limit: Option<u32>,
) -> StdResult<AllTransceiverPeersResponse> {
    let start = start_after.map(|key| Bound::Exclusive((key, PhantomData)));

    let iter = TRANSCEIVER_PEER
        .range(deps.storage, start, None, Order::Ascending)
        .map(|item| item.map(|(key, data)| TransceiverPeer { key, data }));

    if let Some(lim) = limit {
        let l = lim
            .try_into()
            .map_err(|_| ConversionOverflowError::new("u32", "usize", lim.to_string()))?;
        iter.take(l)
            .collect::<StdResult<Vec<_>>>()
            .map(|peers| AllTransceiverPeersResponse { peers })
    } else {
        iter.collect::<StdResult<Vec<_>>>()
            .map(|peers| AllTransceiverPeersResponse { peers })
    }
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

fn normalize_transfer_amount(trimmed_amount: TrimmedAmount) -> Uint256 {
    let to_decimals = TRIMMED_DECIMALS;
    let from_decimals = trimmed_amount.decimals;
    let amount = Uint256::from(trimmed_amount.amount);
    if from_decimals == to_decimals {
        return amount;
    }
    if from_decimals > to_decimals {
        amount / Uint256::from(10u64).pow((from_decimals - to_decimals).into())
    } else {
        amount * Uint256::from(10u64).pow((to_decimals - from_decimals).into())
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn consts() {
        // the accountant contract would need this to never change, and if it did change, would require a migration
        assert_eq!(TRIMMED_DECIMALS, 8)
    }

    #[test]
    fn normalize() {
        assert_eq!(
            normalize_transfer_amount(TrimmedAmount {
                amount: 1_000,
                decimals: 3
            }),
            Uint256::from(100_000_000u64)
        );
        assert_eq!(
            normalize_transfer_amount(TrimmedAmount {
                amount: 1_000_000_000_000_000_000,
                decimals: 18
            }),
            Uint256::from(100_000_000u64)
        );
        assert_eq!(
            normalize_transfer_amount(TrimmedAmount {
                amount: 10_000_000_000,
                decimals: 18
            }),
            Uint256::from(1u64)
        );
        assert_eq!(
            normalize_transfer_amount(TrimmedAmount {
                amount: 1,
                decimals: 18
            }),
            Uint256::from(0u64)
        );
    }
}
