use cw_wormhole::{byte_utils::ByteUtils, msg::QueryMsg as WormholeQueryMsg, state::ParsedVAA};

use cw_token_bridge::msg::{
    ExecuteMsg as TokenBridgeExecuteMsg, QueryMsg as TokenBridgeQueryMsg,
    TransferInfoResponse as TokenBridgeTransferInfoResponse,
};

#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use cosmwasm_std::{
    to_binary, Binary, CosmosMsg, Deps, DepsMut, Env, MessageInfo, QueryRequest, Reply, Response,
    StdError, StdResult, SubMsg, WasmMsg, WasmQuery,
};

use crate::{
    msg::{ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg},
    state::{
        chain_channels_read, config, config_read, current_transfer, ConfigInfo, TransferPayload,
        TransferState,
    },
};

type HumanAddr = String;

const COMPLETE_TRANSFER_REPLY_ID: u64 = 1;

pub enum TransferType<A> {
    WithoutPayload,
    WithPayload { payload: A },
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
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
        wormhole_contract: msg.wormhole_contract,
        token_bridge_contract: msg.token_bridge_contract,
        chain_id: msg.chain_id,
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
        return Err(StdError::generic_err(
            "reply called for unexpected message type",
        ));
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

        #[cfg(feature = "full")]
        ExecuteMsg::CompleteTransferAndConvert { vaa } => {
            complete_transfer_and_convert(deps, info, vaa)
        }

        // When in "shutdown" mode, we reject any other action
        #[cfg(not(feature = "full"))]
        _ => Err(StdError::generic_err("Invalid during shutdown mode")),
    }
}

fn submit_vaa(deps: DepsMut, env: Env, _info: MessageInfo, data: &Binary) -> StdResult<Response> {
    // let state = config_read(deps.storage).load()?;
    let vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), data)?;

    // Only accept payload3 VAAs.
    if vaa.version != 3 {
        return Err(StdError::generic_err("expected payload3 VAA"));
    }
    return Err(StdError::generic_err("submit_vaa not implemented"));
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
/// Calls into the wormhole token bridge to complete the payload3 transfer.
fn complete_transfer_and_convert(
    deps: DepsMut,
    info: MessageInfo,
    vaa: Binary,
) -> StdResult<Response> {
    // get the token bridge contract address from storage
    let cfg = config_read(deps.storage).load()?;
    let token_bridge_contract = cfg.token_bridge_contract;

    // craft the token bridge execute message
    // this will be added as a submessage to the response
    let _token_bridge_execute_msg =
        to_binary(&TokenBridgeExecuteMsg::CompleteTransferWithPayload {
            data: vaa.clone(),
            relayer: info.sender.to_string(),
        })
        .unwrap();
    // .context("could not serialize token bridge execute msg")?;

    // The transfer needs to be:
    // From addr: from_addr
    // From chain: src_chain_id
    // To addr: from_addr
    // To chain: wormchain

    // let sub_msg = SubMsg::reply_on_success(
    //     CosmosMsg::Wasm(WasmMsg::Execute {
    //         contract_addr: token_bridge_contract.clone(),
    //         msg: token_bridge_execute_msg,
    //         funds: vec![],
    //     }),
    //     COMPLETE_TRANSFER_REPLY_ID,
    // );

    // craft the token bridge query message to parse the payload3 vaa
    let token_bridge_query_msg = to_binary(&TokenBridgeQueryMsg::TransferInfo { vaa }).unwrap();
    // .context("could not serialize token bridge transfer_info query msg")?;

    let transfer_info: TokenBridgeTransferInfoResponse = deps
        .querier
        .query(&QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: token_bridge_contract,
            msg: token_bridge_query_msg,
        }))
        .unwrap();
    // .context("could not parse token bridge payload3 vaa")?;

    // convert info to string for logging
    let ti = format!(
        "amount = {}, token_chain = {}, recipient_chain = {}, fee = {}",
        transfer_info.amount,
        transfer_info.token_chain,
        transfer_info.recipient_chain,
        transfer_info.fee
    );

    // Want the fromAddress in the VAA's payload to be copied into the recipient field of the transfer info response

    // save interim state
    // CURRENT_TRANSFER
    //     .save(deps.storage, &transfer_info)
    //     .context("failed to save current transfer to storage")?;

    // return the response which will callback to the reply handler on success
    Ok(Response::new()
        // .add_submessage(sub_msg)
        .add_attribute("action", "complete_transfer_with_payload")
        .add_attribute("transfer_info", ti)
        .add_attribute(
            "token_address",
            Binary::from(transfer_info.token_address).to_base64(),
        )
        .add_attribute(
            "recipient",
            Binary::from(transfer_info.recipient).to_base64(),
        )
        .add_attribute(
            "transfer_payload",
            Binary::from(transfer_info.payload).to_base64(),
        ))
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
    let token_bridge_query_msg =
        to_binary(&TokenBridgeQueryMsg::TransferInfo { vaa: vaa.clone() })?;
    let transfer_info: TokenBridgeTransferInfoResponse =
        deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: cfg.token_bridge_contract.clone(),
            msg: token_bridge_query_msg,
        }))?;

    // The transfer must be destined for this chain.
    if transfer_info.recipient_chain != cfg.chain_id {
        return Err(StdError::generic_err("invalid recipient chain"));
    }

    // The transfer must be destined for this contract.
    let vaa_recipient = deps
        .api
        .addr_humanize(&(&transfer_info.recipient.as_slice()).get_address(0))?;
    // let vaa_recipient = env.contract.address.clone(); ////////////////////////////////////// DO NOT COMMIT THIS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
    if vaa_recipient != env.contract.address {
        let s = format!(
            "Invalid recipient address: vaa_recipient = {}, env.contract.address = {}",
            vaa_recipient, env.contract.address
        );
        return Err(StdError::generic_err(s));
        // return Err(StdError::generic_err("invalid recipient address"));
    }

    // Parse the payload three data.
    let payload: TransferPayload = serde_json_wasm::from_slice(&transfer_info.payload).unwrap();
    match payload {
        TransferPayload::BasicTransfer {
            chain_id,
            recipient,
        } => post_complete_transfer_with_payload(
            deps,
            env,
            cfg.token_bridge_contract.clone(),
            vaa.clone(),
            relayer_address.clone(),
            transfer_info,
            chain_id,
            recipient,
        ),
        TransferPayload::BasicDeposit { amount: _ } =>
        //Err(StdError::generic_err("wrong payload type")),
        {
            post_complete_deposit_with_payload(deps, vaa.clone(), transfer_info)
        }
    }
}

fn post_complete_deposit_with_payload(
    deps: DepsMut,
    vaa: Binary,
    _transfer_info: TokenBridgeTransferInfoResponse,
) -> StdResult<Response> {
    // Check the token is supported.

    // get the token bridge contract address from storage
    let cfg = config_read(deps.storage).load()?;
    let _token_bridge_contract = cfg.token_bridge_contract;

    // TODO: Get the from: address in the VAA
    let _parsed_vaa = ParsedVAA::deserialize(&vaa)?;
    // let fromAddr = parsedVaa.

    return Err(StdError::generic_err("unsupported function"));
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
    // Look up the target chain ID in our map.
    let target_channel_lookup =
        chain_channels_read(deps.storage).load(&target_chain_id.to_be_bytes());
    if !target_channel_lookup.is_ok() {
        let s = format!("Unknown target chain: {}", target_chain_id);
        return Err(StdError::generic_err(s));
        // return Err(StdError::generic_err("unknown target chain"));
    }

    let target_channel_id = target_channel_lookup.unwrap();

    // If the channel ID is null, that means transfers to that chain are not allowed.
    if target_channel_id == "" {
        return Err(StdError::generic_err(
            "transfers to target chain are disabled",
        ));
    }

    // Build the complete transfer sub message for the token bridge.
    let token_bridge_execute_msg =
        to_binary(&TokenBridgeExecuteMsg::CompleteTransferWithPayload {
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

    Ok(Response::new().add_submessage(sub_msg))
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
pub fn query(_deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::PlaceHolder {} => {
            return Err(StdError::generic_err("placeholder not implemented"));
        }
    }
}
