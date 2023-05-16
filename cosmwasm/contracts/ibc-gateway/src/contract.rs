use cw_wormhole::{
    byte_utils::{ByteUtils, get_string_from_32},
    error::ContractError,
    msg::{QueryMsg as WormholeQueryMsg},
    state::{vaa_archive_add, vaa_archive_check, GovernancePacket, ParsedVAA},
};

use cw_token_bridge::{
    msg::{ExecuteMsg as TokenBridgeExecuteMsg, QueryMsg as TokenBridgeQueryMsg, TransferInfoResponse as TokenBridgeTransferInfoResponse},
};

#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use cosmwasm_std::{
    to_binary, Binary, CosmosMsg, Deps, DepsMut, Env,
    MessageInfo, Order, QueryRequest, Reply, Response, StdError, StdResult,
    WasmMsg, WasmQuery, SubMsg,
};

use crate::{
    msg::{
        AllChainChannelsResponse, ExecuteMsg,
        InstantiateMsg, IsVaaRedeemedResponse, MigrateMsg, QueryMsg,
    },
    state::{
        chain_channels, chain_channels_read, config, config_read,
        ConfigInfo, current_transfer, RegisterChainChannel, TransferPayload, TransferState,
        UpgradeContract,
    },
};

type HumanAddr = String;

const COMPLETE_TRANSFER_REPLY_ID: u64 = 1;

pub enum TransferType<A> {
    WithoutPayload,
    WithPayload { payload: A },
}

/// Migration code that runs the next time the contract is upgraded.
/// This function will contain ephemeral code that we want to run once, and thus
/// can (and should be) safely deleted after the upgrade happened successfully.
///
/// Most upgrades won't require any special migration logic. In those cases,
/// this function can safely be implemented as:
/// ```ignore
/// Ok(Response::default())
/// ```
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    // // This migration adds a new field to the [`ConfigInfo`] struct. The
    // // state stored on chain has the old version, so we first parse it as
    // // [`ConfigInfoLegacy`], then add the new fields, and write it back as [`ConfigInfo`].
    // // Since the only place the contract with the legacy state is deployed is
    // // terra2, we just hardcode the new value here for that chain.

    // // 1. make sure this contract doesn't already have the new ConfigInfo struct
    // // in storage. Note that this check is not strictly necessary, as the
    // // upgrade will only be issued for terra2, and no new chains. However, it is
    // // good practice to ensure that migration code cannot be run twice, which
    // // this check achieves.
    // if config_read(deps.storage).load().is_ok() {
    //     return Err(StdError::generic_err(
    //         "Can't migrate; this contract already has a new ConfigInfo struct",
    //     ));
    // }

    // // 2. parse old state
    // let ConfigInfoLegacy {
    //     gov_chain,
    //     gov_address,
    //     wormhole_contract,
    //     wrapped_asset_code_id,
    // } = config_read_legacy(deps.storage).load()?;

    // // 3. store new state with terra2 values hardcoded
    // let chain_id = 18;
    // let native_denom = "uluna".to_string();
    // let native_symbol = "LUNA".to_string();
    // let native_decimals = 6;

    // let config_info = ConfigInfo {
    //     gov_chain,
    //     gov_address,
    //     wormhole_contract,
    //     wrapped_asset_code_id,
    //     chain_id,
    //     native_denom,
    //     native_symbol,
    //     native_decimals
    // };

    // config(deps.storage).save(&config_info)?;
    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    // Save general wormhole info
    let state = ConfigInfo {
        gov_chain: msg.gov_chain,
        gov_address: msg.gov_address.into(),
        wormhole_contract: msg.wormhole_contract,
        token_bridge_contract: msg.token_bridge_contract,
        wrapped_asset_code_id: msg.wrapped_asset_code_id,
        chain_id: msg.chain_id,
        native_denom: msg.native_denom,
        native_symbol: msg.native_symbol,
        native_decimals: msg.native_decimals,
    };
    config(deps.storage).save(&state)?;

    // CHAIN_CHANNELS.save(deps.storage, 18, &"channel-0".into())?;

    Ok(Response::default())
}

// When CW20 transfers complete, we need to verify the actual amount that is being transferred out
// of the bridge. This is to handle fee tokens where the amount expected to be transferred may be
// less due to burns, fees, etc.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(deps: DepsMut, env: Env, msg: Reply) -> StdResult<Response> {
    let state: TransferState = current_transfer(deps.storage).load()?;

    // NOTE: Reentrancy protection. See note in `post_complete_transfer_with_payload` for why this is necessary.
    current_transfer(deps.storage).remove();

    // handle submessage cases based on the reply id
    if msg.id != COMPLETE_TRANSFER_REPLY_ID {
        return Err(StdError::generic_err("reply called for unexpected message type"));
    }

    return handle_complete_transfer_reply(deps, env, msg, state);
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(deps: DepsMut, env: Env, info: MessageInfo, msg: ExecuteMsg) -> StdResult<Response> {
    match msg {
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),

        #[cfg(feature = "full")]
        ExecuteMsg::CompleteTransferWithPayload { data, relayer } => {
            handle_complete_transfer_with_payload(deps, env, &data, &relayer)
        }

        // When in "shutdown" mode, we reject any other action
        #[cfg(not(feature = "full"))]
        _ => Err(StdError::generic_err("Invalid during shutdown mode")),
    }
}

fn submit_vaa(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let state = config_read(deps.storage).load()?;
    let vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), data)?;

    // The only VAAs we expect are governance VAAs.
    if ! is_governance_emitter(&state, vaa.emitter_chain, &vaa.emitter_address) {
        return Err(StdError::generic_err("expected a governance VAA"));
    }

    // Make sure we haven't already processed this VAA, and add it to the archive.
    if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
        return Err(StdError::generic_err("VAA already executed"));
    }

    vaa_archive_add(deps.storage, vaa.hash.as_slice())?;

    // Parse the governance payload and execute it.
    let gov_packet = GovernancePacket::deserialize(&vaa.payload)?;
    handle_governance_payload(deps, env, &gov_packet)
}

fn parse_vaa(deps: Deps, block_time: u64, data: &Binary) -> StdResult<ParsedVAA> {
    let cfg = config_read(deps.storage).load()?;
    let vaa: ParsedVAA = deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: cfg.wormhole_contract,
        msg: to_binary(&WormholeQueryMsg::VerifyVAA {
            vaa: data.clone(),
            block_time,
        })?,
    }))?;
    Ok(vaa)
}

fn handle_governance_payload(
    deps: DepsMut,
    env: Env,
    gov_packet: &GovernancePacket,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;
    let module = get_string_from_32(&gov_packet.module);

    if module != "IbcTranslator" {
        return Err(StdError::generic_err("governance VAA is for an invalid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != cfg.chain_id {
        return Err(StdError::generic_err("governance VAA is for another chain"));
    }    

    match gov_packet.action {
        1u8 => handle_register_chain_channel(deps, env, &gov_packet.payload),
        2u8 => handle_upgrade_contract(deps, env, &gov_packet.payload),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
}

fn handle_register_chain_channel(deps: DepsMut, _env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let RegisterChainChannel {
        chain_id,
        channel_id,
    } = RegisterChainChannel::deserialize(data)?;

    // Note that we are allowing updates to change the channel for a chain.

    let mut bucket = chain_channels(deps.storage);
    bucket.save(&chain_id.to_be_bytes(), &channel_id)?;

    if channel_id == "" {
        return Ok(Response::new()
            .add_attribute("chain_id", chain_id.to_string())
            .add_attribute("channel_id", "disabled"))
    }

    Ok(Response::new()
        .add_attribute("chain_id", chain_id.to_string())
        .add_attribute("channel_id", channel_id))
}

fn handle_upgrade_contract(_deps: DepsMut, env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let UpgradeContract { new_contract } = UpgradeContract::deserialize(data)?;

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(WasmMsg::Migrate {
            contract_addr: env.contract.address.to_string(),
            new_code_id: new_contract,
            msg: to_binary(&MigrateMsg {})?,
        }))
        .add_attribute("action", "contract_upgrade"))
}

fn handle_complete_transfer_with_payload(
    deps: DepsMut,
    env: Env,
    vaa: &Binary,
    relayer_address: &HumanAddr, 
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;
    let parsed_vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), vaa)?;

    if parsed_vaa.payload.len() < 1 {
        return Err(StdError::generic_err("payload is missing"));
    }

    if parsed_vaa.payload[0] != 3 {
        return Err(StdError::generic_err("unexpected payload type"));
    }    

    // Query the token bridge to parse the payload3 VAA.
    let token_bridge_query_msg = to_binary(&TokenBridgeQueryMsg::TransferInfo { vaa: vaa.clone() })?;
    let transfer_info: TokenBridgeTransferInfoResponse = deps
        .querier
        .query(&QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: cfg.token_bridge_contract.clone(),
            msg: token_bridge_query_msg,
        }))?;
    
    // The transfer must be destined for this chain.
    if transfer_info.recipient_chain != cfg.chain_id {
        return Err(StdError::generic_err("invalid recipient chain"));
    }

    // The transfer must be destined for this contract.
    let vaa_recipient = deps.api.addr_humanize(&(&transfer_info.recipient.as_slice()).get_address(0))?;
    // let vaa_recipient = env.contract.address.clone(); ////////////////////////////////////// DO NOT COMMIT THIS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
    if vaa_recipient != env.contract.address {
        return Err(StdError::generic_err("invalid recipient address"));
    }

    // Parse the payload three data.
    let payload: TransferPayload = serde_json_wasm::from_slice(&transfer_info.payload).unwrap();
    match payload {
        TransferPayload::BasicTransfer { chain_id, recipient } => {
            post_complete_transfer_with_payload(
                deps,
                env,
                cfg.token_bridge_contract.clone(),
                vaa.clone(),
                relayer_address.clone(),
                transfer_info,
                chain_id,
                recipient,
            )
        }
    }
}

fn post_complete_transfer_with_payload(
    deps: DepsMut,
    _env: Env,
    token_bridge_contract: String,
    vaa: Binary,
    relayer_address: HumanAddr,
    transfer_info: TokenBridgeTransferInfoResponse,
    target_chain_id: u16,
    target_recipient: Binary,
) -> StdResult<Response> {
    // return Err(StdError::generic_err("invalid recipient address"));

    // Look up the target chain ID in our map.
    let target_channel_lookup = chain_channels_read(deps.storage).load(&target_chain_id.to_be_bytes());
    if ! target_channel_lookup.is_ok() {
        return Err(StdError::generic_err("unknown target chain"));
    }

    let target_channel_id = target_channel_lookup.unwrap();

    // If the channel ID is null, that means transfers to that chain are not allowed.
    if target_channel_id == "" {
        return Err(StdError::generic_err("transfers to target chain are disabled"));
    }

    // Build the complete transfer sub message for the token bridge.
    let token_bridge_execute_msg = to_binary(&TokenBridgeExecuteMsg::CompleteTransferWithPayload {
        data: vaa.clone(),
        relayer: relayer_address,
    })?;

    let sub_msg = SubMsg::reply_on_success(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: token_bridge_contract.clone(),
            msg: token_bridge_execute_msg,
            funds: vec![],
        }),
        COMPLETE_TRANSFER_REPLY_ID,
    );
    
    // NOTE: Reentrancy protection. It is crucial that there's no
    // ongoing transfer in progress here, otherwise we would override
    // its state. A simple protection mechanism is to require
    // that there's no execution in progress. The reply handler takes
    // care of clearing out this temporary storage when done.
    assert!(current_transfer(deps.storage).load().is_err());

    // Save our current state to be used by the submessage reply.
    current_transfer(deps.storage).save(&TransferState {
        transfer_info: transfer_info,
        target_chain_id: target_chain_id,
        target_channel_id: target_channel_id,
        target_recipient: target_recipient,
    })?;

    Ok(Response::new()
        .add_submessage(sub_msg))
}

fn handle_complete_transfer_reply(
    _deps: DepsMut,
    _env: Env,
    _msg: Reply,
    _state: TransferState,
) -> StdResult<Response> {
    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::IsVaaRedeemed { vaa } => to_binary(&query_is_vaa_redeemed(deps, env, &vaa)?),
        QueryMsg::AllChainChannels {} => {
            query_all_chain_channels(deps).and_then(|resp| to_binary(&resp))
        }
    }
}

fn query_is_vaa_redeemed(deps: Deps, _env: Env, vaa: &Binary) -> StdResult<IsVaaRedeemedResponse> {
    let vaa = ParsedVAA::deserialize(vaa)?;
    Ok(IsVaaRedeemedResponse {
        is_redeemed: vaa_archive_check(deps.storage, vaa.hash.as_slice()),
    })
}

fn is_governance_emitter(cfg: &ConfigInfo, emitter_chain: u16, emitter_address: &[u8]) -> bool {
    cfg.gov_chain == emitter_chain && cfg.gov_address == emitter_address
}

fn query_all_chain_channels(deps: Deps) -> StdResult<AllChainChannelsResponse> {
    chain_channels_read(deps.storage)
        .range(None, None, Order::Ascending)
        .map(|res| {
           res.map(|(chain_id, channel_id)| {
                (u16::from_be_bytes([chain_id[0], chain_id[1]]), Binary::from(Vec::<u8>::from(channel_id))) // TODO: There must be a better way!
            })
        })
        .collect::<StdResult<Vec<_>>>()
        .map(|chain_channels| AllChainChannelsResponse { chain_channels })
}
