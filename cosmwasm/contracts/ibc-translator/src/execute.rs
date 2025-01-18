use anyhow::{bail, ensure, Context};
use cosmwasm_std::{
    to_json_binary, Binary, Coin, CosmosMsg, Deps, DepsMut, Empty, Env, Event, MessageInfo,
    QueryRequest, Response, SubMsg, Uint128, WasmMsg, WasmQuery,
};
use cw_token_bridge::msg::{
    Asset, AssetInfo, ExecuteMsg as TokenBridgeExecuteMsg, QueryMsg as TokenBridgeQueryMsg,
    TransferInfoResponse,
};
use cw_wormhole::byte_utils::ByteUtils;

use cw20_wrapped_2::msg::ExecuteMsg as Cw20WrappedExecuteMsg;
use serde_wormhole::RawMessage;
use std::str;
use wormhole_bindings::{
    tokenfactory::{TokenFactoryMsg, TokenMsg},
    WormholeQuery,
};
use wormhole_sdk::{
    ibc_translator::{Action, GovernancePacket},
    vaa::{Body, Header},
    Chain,
};

use crate::{
    msg::COMPLETE_TRANSFER_REPLY_ID,
    state::{
        CHAIN_TO_CHANNEL_MAP, CURRENT_TRANSFER, CW_DENOMS, TOKEN_BRIDGE_CONTRACT, VAA_ARCHIVE,
    },
};

pub enum TransferType {
    Simple { fee: Uint128 },
    ContractControlled { payload: Binary },
}

/// Calls into the wormhole token bridge to complete the payload3 transfer.
pub fn complete_transfer_and_convert(
    deps: DepsMut<WormholeQuery>,
    env: Env,
    info: MessageInfo,
    vaa: Binary,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    // get the token bridge contract address from storage
    let token_bridge_contract = TOKEN_BRIDGE_CONTRACT
        .load(deps.storage)
        .context("could not load token bridge contract address")?;

    // craft the token bridge execute message
    // this will be added as a submessage to the response
    let token_bridge_execute_msg =
        to_json_binary(&TokenBridgeExecuteMsg::CompleteTransferWithPayload {
            data: vaa.clone(),
            relayer: info.sender.to_string(),
        })
        .context("could not serialize token bridge execute msg")?;

    let sub_msg = SubMsg::reply_on_success(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: token_bridge_contract.clone(),
            msg: token_bridge_execute_msg,
            funds: vec![],
        }),
        COMPLETE_TRANSFER_REPLY_ID,
    );

    // craft the token bridge query message to parse the payload3 vaa
    let token_bridge_query_msg = to_json_binary(&TokenBridgeQueryMsg::TransferInfo { vaa })
        .context("could not serialize token bridge transfer_info query msg")?;

    let transfer_info: TransferInfoResponse = deps
        .querier
        .query(&QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: token_bridge_contract,
            msg: token_bridge_query_msg,
        }))
        .context("could not parse token bridge payload3 vaa")?;

    // DEFENSE IN-DEPTH CHECK FOR PAYLOAD3 VAAs
    // ensure that the transfer vaa recipient is this contract.
    // we should never process any VAAs that are not directed to this contract.
    let target_address = (&transfer_info.recipient.as_slice()).get_address(0);
    let recipient = deps.api.addr_humanize(&target_address)?;
    ensure!(
        recipient == env.contract.address,
        "vaa recipient must be this contract"
    );

    // save interim state
    CURRENT_TRANSFER
        .save(deps.storage, &transfer_info)
        .context("failed to save current transfer to storage")?;

    // return the response which will callback to the reply handler on success
    Ok(Response::new()
        .add_submessage(sub_msg)
        .add_attribute("action", "complete_transfer_with_payload")
        .add_attribute(
            "transfer_payload",
            Binary::from(transfer_info.payload).to_base64(),
        ))
}

pub fn convert_and_transfer(
    deps: DepsMut<WormholeQuery>,
    info: MessageInfo,
    env: Env,
    recipient: Binary,
    chain: u16,
    transfer_type: TransferType,
    nonce: u32,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    // load the token bridge contract address
    let token_bridge_contract = TOKEN_BRIDGE_CONTRACT
        .load(deps.storage)
        .context("could not load token bridge contract address")?;

    ensure!(
        info.funds.len() == 1,
        "info.funds should contain only 1 coin"
    );
    let bridging_coin = info.funds[0].clone();
    let cw20_contract_addr = parse_bank_token_factory_contract(deps, env, bridging_coin.clone())?;

    // batch calls together
    let mut response: Response<TokenFactoryMsg> = Response::new();

    // 1. tokenfactorymsg::burn for the bank tokens
    response = response.add_message(TokenMsg::BurnTokens {
        denom: bridging_coin.denom.clone(),
        amount: bridging_coin.amount.u128(),
        burn_from_address: "".to_string(),
    });

    // 2. cw20::increaseAllowance to the contract address for the token bridge to spend the amount of tokens
    let increase_allowance_msg = to_json_binary(&Cw20WrappedExecuteMsg::IncreaseAllowance {
        spender: token_bridge_contract.clone(),
        amount: bridging_coin.amount,
        expires: None,
    })
    .context("could not serialize cw20 increase_allowance msg")?;
    response = response.add_message(CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: cw20_contract_addr.clone(),
        msg: increase_allowance_msg,
        funds: vec![],
    }));

    // 3. token_bridge::initiate_transfer -- the cw20 tokens will be either burned or transferred to the token_bridge
    let token_bridge_transfer: TokenBridgeExecuteMsg = match transfer_type {
        TransferType::Simple { fee } => TokenBridgeExecuteMsg::InitiateTransfer {
            asset: Asset {
                info: AssetInfo::Token {
                    contract_addr: cw20_contract_addr,
                },
                amount: bridging_coin.amount,
            },
            recipient_chain: chain,
            recipient,
            fee,
            nonce,
        },
        TransferType::ContractControlled { payload } => {
            TokenBridgeExecuteMsg::InitiateTransferWithPayload {
                asset: Asset {
                    info: AssetInfo::Token {
                        contract_addr: cw20_contract_addr,
                    },
                    amount: bridging_coin.amount,
                },
                recipient_chain: chain,
                recipient,
                fee: Uint128::from(0u128),
                payload,
                nonce,
            }
        }
    };
    let initiate_transfer_msg = to_json_binary(&token_bridge_transfer)
        .context("could not serialize token bridge initiate_transfer msg")?;
    response = response.add_message(CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: token_bridge_contract,
        msg: initiate_transfer_msg,
        funds: vec![],
    }));

    Ok(response)
}

pub fn parse_bank_token_factory_contract(
    deps: DepsMut<WormholeQuery>,
    env: Env,
    coin: Coin,
) -> Result<String, anyhow::Error> {
    // extract the contract address from the denom of the token that was sent to us
    // if the token is not a factory token created by this contract, return error
    let parsed_denom = coin.denom.split('/').collect::<Vec<_>>();
    ensure!(
        parsed_denom.len() == 3
            && parsed_denom[0] == "factory"
            && parsed_denom[1] == env.contract.address,
        "coin is not from the token factory"
    );

    // decode subdenom from base64 => encode as cosmos addr to get contract addr
    let cw20_contract_addr = contract_addr_from_base58(deps.as_ref(), parsed_denom[2])?;

    // validate that the contract does indeed match the stored denom we have for it
    let stored_denom = CW_DENOMS
        .load(deps.storage, cw20_contract_addr.clone())
        .context(
            "a corresponding denom for the extracted contract addr is not contained in storage",
        )?;
    ensure!(
        stored_denom == coin.denom,
        "the stored denom for the contract does not match the actual coin denom"
    );

    Ok(cw20_contract_addr)
}

pub fn contract_addr_from_base58(
    deps: Deps<WormholeQuery>,
    subdenom: &str,
) -> Result<String, anyhow::Error> {
    let decoded_addr = bs58::decode(subdenom)
        .into_vec()
        .context(format!("failed to decode base58 subdenom {subdenom}"))?;
    let canonical_addr = Binary::from(decoded_addr);
    deps.api
        .addr_humanize(&canonical_addr.into())
        .map(|a| a.to_string())
        .context(format!("failed to humanize cosmos address {subdenom}"))
}

pub fn submit_update_chain_to_channel_map(
    deps: DepsMut<WormholeQuery>,
    vaa: Binary,
) -> Result<Response<TokenFactoryMsg>, anyhow::Error> {
    // parse the VAA header and data
    let (header, data) = serde_wormhole::from_slice::<(Header, &RawMessage)>(&vaa)
        .context("failed to parse VAA header")?;

    // Must be a version 1 VAA
    ensure!(header.version == 1, "unsupported VAA version");

    // call into wormchain to verify the VAA
    deps.querier
        .query::<Empty>(&WormholeQuery::VerifyVaa { vaa: vaa.clone() }.into())
        .context("failed to verify vaa")?;

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
        govpacket.chain == Chain::Wormchain || govpacket.chain == Chain::Any,
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
            ensure!(
                chain_id != Chain::Wormchain,
                "the ibc-translator contract should not maintain channel mappings to wormchain"
            );

            let channel_id_str =
                str::from_utf8(&channel_id).context("failed to parse channel-id as utf-8")?;
            let channel_id_trimmed = channel_id_str.trim_start_matches(char::from(0));

            // update storage with the mapping
            CHAIN_TO_CHANNEL_MAP
                .save(
                    deps.storage,
                    chain_id.into(),
                    &channel_id_trimmed.to_string(),
                )
                .context("failed to save channel chain")?;
            Ok(Response::new().add_event(
                Event::new("UpdateChainToChannelMap")
                    .add_attribute("chain_id", chain_id.to_string())
                    .add_attribute("channel_id", channel_id_trimmed),
            ))
        }
    }
}
