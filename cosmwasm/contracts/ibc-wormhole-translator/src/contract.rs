use cw_wormhole::{
    byte_utils::{
        get_string_from_32,
    },
    error::ContractError,
    msg::{QueryMsg as WormholeQueryMsg},
    state::{vaa_archive_add, vaa_archive_check, GovernancePacket, ParsedVAA},
};

// use cw_token_bridge::{
//     msg::{ExecuteMsg as TokenBridgeExecuteMsg, QueryMsg as TokenBridgeQueryMessage},
// };

#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Env,
    MessageInfo, Order, QueryRequest, Reply, Response, StdError, StdResult, Uint128,
    WasmQuery,
};

use crate::{
    msg::{
        AllChainChannelsResponse, Asset, ChainRegistrationResponse, ExecuteMsg,
        ExternalIdResponse, InstantiateMsg, IsVaaRedeemedResponse, MigrateMsg, QueryMsg,
        TransferInfoResponse,
    },
    state::{
        bridge_contracts_read, config, config_read,
        Action, CHAIN_CHANNELS, ConfigInfo, RegisterChainChannel, TokenBridgeMessage, TransferInfo,
        TransferWithPayloadInfo,
    },
    token_address::{ExternalTokenId},
};

type HumanAddr = String;

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
pub fn reply(_deps: DepsMut, _env: Env, _msg: Reply) -> StdResult<Response> {
    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(deps: DepsMut, env: Env, info: MessageInfo, msg: ExecuteMsg) -> StdResult<Response> {
    match msg {
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),

        // // The following actions are disabled in "shutdown" mode
        // #[cfg(feature = "full")]
        // ExecuteMsg::RegisterAssetHook {
        //     chain,
        //     token_address,
        // } => handle_register_asset(deps, env, info, chain, token_address),

        // #[cfg(feature = "full")]
        // ExecuteMsg::InitiateTransfer {
        //     asset,
        //     recipient_chain,
        //     recipient,
        //     fee,
        //     nonce,
        // } => handle_initiate_transfer(
        //     deps,
        //     env,
        //     info,
        //     asset,
        //     recipient_chain,
        //     recipient.to_array()?,
        //     fee,
        //     TransferType::WithoutPayload,
        //     nonce,
        // ),

        #[cfg(feature = "full")]
        ExecuteMsg::InitiateTransferWithPayload {
            asset,
            recipient_chain,
            recipient,
            fee,
            payload,
            nonce,
        } => handle_initiate_transfer(
            deps,
            env,
            info,
            asset,
            recipient_chain,
            recipient.to_array()?,
            fee,
            TransferType::WithPayload {
                payload: payload.into(),
            },
            nonce,
        ),

        // #[cfg(feature = "full")]
        // ExecuteMsg::DepositTokens {} => deposit_tokens(deps, env, info),

        // #[cfg(feature = "full")]
        // ExecuteMsg::WithdrawTokens { asset } => withdraw_tokens(deps, env, info, asset),

        // #[cfg(feature = "full")]
        // ExecuteMsg::CreateAssetMeta { asset_info, nonce } => {
        //     handle_create_asset_meta(deps, env, info, asset_info, nonce)
        // }

        // #[cfg(feature = "full")]
        // ExecuteMsg::CompleteTransferWithPayload { data, relayer } => {
        //     handle_complete_transfer_with_payload(deps, env, info, &data, &relayer)
        // }

        // When in "shutdown" mode, we reject any other action
        // #[cfg(not(feature = "full"))]
        _ => Err(StdError::generic_err("Invalid during shutdown mode")),
    }
}

fn submit_vaa(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let (vaa, payload) = parse_and_archive_vaa(deps.branch(), env.clone(), data)?;
    match payload {
        Either::Left(governance_packet) => handle_governance_payload(deps, env, &governance_packet),

        // In "shutdown" mode, we only handle governance payloads
        #[cfg(feature = "full")]
        Either::Right(message) => match message.action {
            Action::TRANSFER => {
                let sender = info.sender.to_string();
                handle_complete_transfer(
                    deps,
                    env,
                    info,
                    vaa.emitter_chain,
                    vaa.emitter_address,
                    TransferType::WithoutPayload,
                    &message.payload,
                    &sender,
                )
            }
        _ => ContractError::InvalidVAAAction.std_err(),
        },

        #[cfg(not(feature = "full"))]
        _ => ContractError::InvalidVAAAction6.std_err(),
    }
}

enum Either<A, B> {
    Left(A),
    Right(B),
}

fn parse_and_archive_vaa(
    deps: DepsMut,
    env: Env,
    data: &Binary,
) -> StdResult<(ParsedVAA, Either<GovernancePacket, TokenBridgeMessage>)> {
    let state = config_read(deps.storage).load()?;

    let vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), data)?;

    if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }
    vaa_archive_add(deps.storage, vaa.hash.as_slice())?;
    
    // check if vaa is from governance
    if is_governance_emitter(&state, vaa.emitter_chain, &vaa.emitter_address) {
        let gov_packet = GovernancePacket::deserialize(&vaa.payload)?;
        return Ok((vaa, Either::Left(gov_packet)));
    }

    let message = TokenBridgeMessage::deserialize(&vaa.payload)?;
    Ok((vaa, Either::Right(message)))
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
        return Err(StdError::generic_err("this is not a valid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != cfg.chain_id {
        return Err(StdError::generic_err(
            "the governance VAA is for another chain",
        ));
    }

    match gov_packet.action {
        1u8 => handle_register_chain_channel(deps, env, &gov_packet.payload),
        // 1u8 => handle_register_chain(deps, env, &gov_packet.payload),
        // 2u8 => handle_upgrade_contract(deps, env, &gov_packet.payload),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
}

fn handle_register_chain_channel(deps: DepsMut, _env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let RegisterChainChannel {
        chain_id,
        channel_id,
    } = RegisterChainChannel::deserialize(data)?;

    if CHAIN_CHANNELS
        .save(deps.storage, chain_id, &channel_id.to_string()).is_err() {
        return Err(StdError::generic_err("failed to add chain_channel"));
    }   

    Ok(Response::new()
        .add_attribute("chain_id", chain_id.to_string())
        .add_attribute("channel_id", channel_id))
}

// fn handle_upgrade_contract(_deps: DepsMut, env: Env, data: &Vec<u8>) -> StdResult<Response> {
//     let UpgradeContract { new_contract } = UpgradeContract::deserialize(data)?;

//     Ok(Response::new()
//         .add_message(CosmosMsg::Wasm(WasmMsg::Migrate {
//             contract_addr: env.contract.address.to_string(),
//             new_code_id: new_contract,
//             msg: to_binary(&MigrateMsg {})?,
//         }))
//         .add_attribute("action", "contract_upgrade"))
// }

// fn handle_register_chain(deps: DepsMut, _env: Env, data: &Vec<u8>) -> StdResult<Response> {
//     let RegisterChain {
//         chain_id,
//         chain_address,
//     } = RegisterChain::deserialize(data)?;

//     let existing = bridge_contracts_read(deps.storage).load(&chain_id.to_be_bytes());
//     if existing.is_ok() {
//         return Err(StdError::generic_err(
//             "bridge contract already exists for this chain",
//         ));
//     }

//     let mut bucket = bridge_contracts(deps.storage);
//     bucket.save(&chain_id.to_be_bytes(), &chain_address)?;

//     Ok(Response::new()
//         .add_attribute("chain_id", chain_id.to_string())
//         .add_attribute("chain_address", hex::encode(chain_address)))
// }

#[allow(clippy::too_many_arguments)]
fn handle_complete_transfer(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _emitter_chain: u16,
    _emitter_address: Vec<u8>,
    _transfer_type: TransferType<()>,
    _data: &Vec<u8>,
    _relayer_address: &HumanAddr,
) -> StdResult<Response> {
    Ok(Response::new())
}

#[allow(clippy::too_many_arguments)]
fn handle_initiate_transfer(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _asset: Asset,
    _recipient_chain: u16,
    _recipient: [u8; 32],
    _fee: Uint128,
    _transfer_type: TransferType<Vec<u8>>,
    _nonce: u32,
) -> StdResult<Response> {
    Ok(Response::new())

}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::TransferInfo { vaa } => to_binary(&query_transfer_info(deps, env, &vaa)?),
        QueryMsg::ExternalId { external_id } => to_binary(&query_external_id(deps, external_id)?),
        QueryMsg::IsVaaRedeemed { vaa } => to_binary(&query_is_vaa_redeemed(deps, env, &vaa)?),
        QueryMsg::ChainRegistration { chain } => {
            query_chain_registration(deps, chain).and_then(|r| to_binary(&r))
        }
        QueryMsg::AllChainChannels {} => {
            query_all_chain_channels(deps).and_then(|resp| to_binary(&resp))
        }
    }
}

fn query_external_id(deps: Deps, external_id: Binary) -> StdResult<ExternalIdResponse> {
    let cfg = config_read(deps.storage).load()?;
    let external_id = ExternalTokenId::deserialize(external_id.to_array()?);
    Ok(ExternalIdResponse {
        token_id: external_id.to_token_id(deps.storage, cfg.chain_id)?,
    })
}

fn query_transfer_info(deps: Deps, env: Env, vaa: &Binary) -> StdResult<TransferInfoResponse> {
    let cfg = config_read(deps.storage).load()?;

    let parsed = parse_vaa(deps, env.block.time.seconds(), vaa)?;
    let data = parsed.payload;

    // check if vaa is from governance
    if is_governance_emitter(&cfg, parsed.emitter_chain, &parsed.emitter_address) {
        return ContractError::InvalidVAAAction.std_err();
    }

    let message = TokenBridgeMessage::deserialize(&data)?;
    match message.action {
        Action::ATTEST_META => ContractError::InvalidVAAAction.std_err(),
        Action::TRANSFER => {
            let core = TransferInfo::deserialize(&message.payload)?;

            Ok(TransferInfoResponse {
                amount: core.amount.1.into(),
                token_address: core.token_address.serialize(),
                token_chain: core.token_chain,
                recipient: core.recipient,
                recipient_chain: core.recipient_chain,
                fee: core.fee.1.into(),
                payload: vec![],
            })
        }
        Action::TRANSFER_WITH_PAYLOAD => {
            let info = TransferWithPayloadInfo::deserialize(&message.payload)?;
            let core = info.as_transfer_info();

            Ok(TransferInfoResponse {
                amount: core.amount.1.into(),
                token_address: core.token_address.serialize(),
                token_chain: core.token_chain,
                recipient: core.recipient,
                recipient_chain: core.recipient_chain,
                fee: core.fee.1.into(),
                payload: info.payload,
            })
        }
        other => Err(StdError::generic_err(format!("Invalid action: {other}"))),
    }
}

fn query_is_vaa_redeemed(deps: Deps, _env: Env, vaa: &Binary) -> StdResult<IsVaaRedeemedResponse> {
    let vaa = ParsedVAA::deserialize(vaa)?;
    Ok(IsVaaRedeemedResponse {
        is_redeemed: vaa_archive_check(deps.storage, vaa.hash.as_slice()),
    })
}

fn query_chain_registration(deps: Deps, chain: u16) -> StdResult<ChainRegistrationResponse> {
    bridge_contracts_read(deps.storage)
        .load(&chain.to_be_bytes())
        .map(Binary::from)
        .map(|address| ChainRegistrationResponse { address })
}

fn is_governance_emitter(cfg: &ConfigInfo, emitter_chain: u16, emitter_address: &[u8]) -> bool {
    cfg.gov_chain == emitter_chain && cfg.gov_address == emitter_address
}

fn query_all_chain_channels(deps: Deps) -> StdResult<AllChainChannelsResponse> {
    CHAIN_CHANNELS
        .range(deps.storage, None, None, Order::Ascending)
        .map(|res| {
            res.map(|(chain_id, channel_id)| {
                (Binary::from(Vec::<u8>::from(channel_id)), chain_id)
            })
        })
        .collect::<StdResult<Vec<_>>>()
        .map(|chain_channels| AllChainChannelsResponse { chain_channels })
}
