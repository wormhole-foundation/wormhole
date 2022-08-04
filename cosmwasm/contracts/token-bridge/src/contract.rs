use cw20::{
    BalanceResponse,
    TokenInfoResponse,
};
use cw20_base::msg::{
    ExecuteMsg as TokenMsg,
    QueryMsg as TokenQuery,
};
use cw20_wrapped_2::msg::{
    ExecuteMsg as WrappedMsg,
    InitHook,
    InstantiateMsg as WrappedInit,
    QueryMsg as WrappedQuery,
    WrappedAssetInfoResponse,
};
use std::{
    cmp::{
        max,
        min,
    },
    str::FromStr,
};
use terraswap::asset::{
    Asset,
    AssetInfo,
};

use wormhole::{
    byte_utils::{
        extend_address_to_32,
        extend_address_to_32_array,
        extend_string_to_32,
        get_string_from_32,
        ByteUtils,
    },
    error::ContractError,
    msg::{
        ExecuteMsg as WormholeExecuteMsg,
        QueryMsg as WormholeQueryMsg,
    },
    state::{
        vaa_archive_add,
        vaa_archive_check,
        GovernancePacket,
        ParsedVAA,
    },
};

#[allow(unused_imports)]
use cosmwasm_std::entry_point;

use cosmwasm_std::{
    coin,
    to_binary,
    BankMsg,
    Binary,
    CanonicalAddr,
    CosmosMsg,
    Deps,
    DepsMut,
    Empty,
    Env,
    MessageInfo,
    QueryRequest,
    Reply,
    Response,
    StdError,
    StdResult,
    SubMsg,
    Uint128,
    WasmMsg,
    WasmQuery,
};

use crate::{
    msg::{
        ExecuteMsg,
        ExternalIdResponse,
        InstantiateMsg,
        MigrateMsg,
        QueryMsg,
        TransferInfoResponse,
        WrappedRegistryResponse,
    },
    state::{
        bridge_contracts,
        bridge_contracts_read,
        bridge_deposit,
        config,
        config_read,
        config_read_legacy,
        is_wrapped_asset,
        is_wrapped_asset_read,
        receive_native,
        send_native,
        wrapped_asset,
        wrapped_asset_read,
        wrapped_asset_seq,
        wrapped_asset_seq_read,
        wrapped_transfer_tmp,
        Action,
        AssetMeta,
        ConfigInfo,
        ConfigInfoLegacy,
        RegisterChain,
        TokenBridgeMessage,
        TransferInfo,
        TransferState,
        TransferWithPayloadInfo,
        UpgradeContract,
    },
    token_address::{
        ContractId,
        ExternalTokenId,
        TokenId,
        WrappedCW20,
    },
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
pub fn migrate(deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    // This migration adds a new field to the [`ConfigInfo`] struct. The
    // state stored on chain has the old version, so we first parse it as
    // [`ConfigInfoLegacy`], then add the new fields, and write it back as [`ConfigInfo`].
    // Since the only place the contract with the legacy state is deployed is
    // terra2, we just hardcode the new value here for that chain.

    // 1. make sure this contract doesn't already have the new ConfigInfo struct
    // in storage. Note that this check is not strictly necessary, as the
    // upgrade will only be issued for terra2, and no new chains. However, it is
    // good practice to ensure that migration code cannot be run twice, which
    // this check achieves.
    if config_read(deps.storage).load().is_ok() {
        return Err(StdError::generic_err(
            "Can't migrate; this contract already has a new ConfigInfo struct",
        ));
    }

    // 2. parse old state
    let ConfigInfoLegacy {
        gov_chain,
        gov_address,
        wormhole_contract,
        wrapped_asset_code_id,
    } = config_read_legacy(deps.storage).load()?;

    // 3. store new state with terra2 values hardcoded
    let chain_id = 18;

    let config_info = ConfigInfo {
        gov_chain,
        gov_address,
        wormhole_contract,
        wrapped_asset_code_id,
        chain_id,
    };

    config(deps.storage).save(&config_info)?;
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
        wrapped_asset_code_id: msg.wrapped_asset_code_id,
        chain_id: msg.chain_id,
    };
    config(deps.storage).save(&state)?;

    Ok(Response::default())
}

// When CW20 transfers complete, we need to verify the actual amount that is being transferred out
// of the bridge. This is to handle fee tokens where the amount expected to be transferred may be
// less due to burns, fees, etc.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn reply(deps: DepsMut, env: Env, _msg: Reply) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

    let state = wrapped_transfer_tmp(deps.storage).load()?;
    // NOTE: Reentrancy protection. See note in `handle_initiate_transfer_token`
    // for why this is necessary.
    wrapped_transfer_tmp(deps.storage).remove();

    let token_bridge_message = TokenBridgeMessage::deserialize(&state.message)?;

    let (mut transfer_info, transfer_type) = match token_bridge_message.action {
        Action::TRANSFER => {
            let info = TransferInfo::deserialize(&token_bridge_message.payload)?;
            Ok((info, TransferType::WithoutPayload))
        }
        Action::TRANSFER_WITH_PAYLOAD => {
            let info = TransferWithPayloadInfo::deserialize(&token_bridge_message.payload)?;
            Ok((
                info.as_transfer_info(),
                TransferType::WithPayload {
                    // put both the payload and sender_address into the payload
                    // field here (which we can do, since [`TransferType`] is
                    // parametric)
                    payload: (info.payload, info.sender_address),
                },
            ))
        }
        _ => Err(StdError::generic_err("Unreachable")),
    }?;

    // Fetch CW20 Balance post-transfer.
    let new_balance: BalanceResponse =
        deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
            contract_addr: state.token_address.to_string(),
            msg: to_binary(&TokenQuery::Balance {
                address: env.contract.address.to_string(),
            })?,
        }))?;

    // Actual amount should be the difference in balance of the CW20 account in question to account
    // for fee tokens.
    let multiplier = Uint128::from_str(&state.multiplier)?;
    let real_amount = new_balance.balance - Uint128::from_str(&state.previous_balance)?;
    let real_amount = real_amount / multiplier;

    // If the fee is too large the user would receive nothing.
    if transfer_info.fee.1 > real_amount.u128() {
        return Err(StdError::generic_err("fee greater than sent amount"));
    }

    // Update Wormhole message to correct amount.
    transfer_info.amount.1 = real_amount.u128();

    let token_bridge_message = match transfer_type {
        TransferType::WithoutPayload => TokenBridgeMessage {
            action: Action::TRANSFER,
            payload: transfer_info.serialize(),
        },
        TransferType::WithPayload { payload } => TokenBridgeMessage {
            action: Action::TRANSFER_WITH_PAYLOAD,
            payload: TransferWithPayloadInfo {
                amount: transfer_info.amount,
                token_address: transfer_info.token_address,
                token_chain: transfer_info.token_chain,
                recipient: transfer_info.recipient,
                recipient_chain: transfer_info.recipient_chain,
                sender_address: payload.1,
                payload: payload.0,
            }
            .serialize(),
        },
    };

    // Post Wormhole Message
    let message = CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: cfg.wormhole_contract,
        funds: vec![],
        msg: to_binary(&WormholeExecuteMsg::PostMessage {
            message: Binary::from(token_bridge_message.serialize()),
            nonce: state.nonce,
        })?,
    });

    let external_id = ExternalTokenId::from_native_cw20(&state.token_address)?;
    send_native(deps.storage, &external_id, real_amount)?;
    Ok(Response::default()
        .add_message(message)
        .add_attribute("action", "reply_handler"))
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

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(deps: DepsMut, env: Env, info: MessageInfo, msg: ExecuteMsg) -> StdResult<Response> {
    match msg {
        ExecuteMsg::RegisterAssetHook {
            chain,
            token_address,
        } => handle_register_asset(deps, env, info, chain, token_address),
        ExecuteMsg::InitiateTransfer {
            asset,
            recipient_chain,
            recipient,
            fee,
            nonce,
        } => handle_initiate_transfer(
            deps,
            env,
            info,
            asset,
            recipient_chain,
            recipient.to_array()?,
            fee,
            TransferType::WithoutPayload,
            nonce,
        ),
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
        ExecuteMsg::DepositTokens {} => deposit_tokens(deps, env, info),
        ExecuteMsg::WithdrawTokens { asset } => withdraw_tokens(deps, env, info, asset),
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),
        ExecuteMsg::CreateAssetMeta { asset_info, nonce } => {
            handle_create_asset_meta(deps, env, info, asset_info, nonce)
        }
        ExecuteMsg::CompleteTransferWithPayload { data, relayer } => {
            handle_complete_transfer_with_payload(deps, env, info, &data, &relayer)
        }
    }
}

fn deposit_tokens(deps: DepsMut, _env: Env, info: MessageInfo) -> StdResult<Response> {
    for coin in info.funds {
        let deposit_key = format!("{}:{}", info.sender, coin.denom);
        bridge_deposit(deps.storage).update(
            deposit_key.as_bytes(),
            |amount: Option<Uint128>| -> StdResult<Uint128> {
                Ok(amount.unwrap_or(Uint128::new(0)) + coin.amount)
            },
        )?;
    }

    Ok(Response::new().add_attribute("action", "deposit_tokens"))
}

fn withdraw_tokens(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    data: AssetInfo,
) -> StdResult<Response> {
    let mut messages: Vec<CosmosMsg> = vec![];
    if let AssetInfo::NativeToken { denom } = data {
        let deposit_key = format!("{}:{}", info.sender, denom);
        let mut deposited_amount: u128 = 0;
        bridge_deposit(deps.storage).update(
            deposit_key.as_bytes(),
            |current: Option<Uint128>| match current {
                Some(v) => {
                    deposited_amount = v.u128();
                    Ok(Uint128::new(0))
                }
                None => Err(StdError::generic_err("no deposit found to withdraw")),
            },
        )?;
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: info.sender.to_string(),
            amount: vec![coin(deposited_amount, &denom)],
        }));
    }

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("action", "withdraw_tokens"))
}

/// Handle wrapped asset registration messages
fn handle_register_asset(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    chain: u16,
    token_address: ExternalTokenId,
) -> StdResult<Response> {
    // We need to ensure that this registration request was initiated by the
    // token bridge contract. We do this by checking that the wrapped asset
    // already has an associated sequence number, but no address entry yet.
    // This is a necessary and sufficient condition, because having the sequence
    // number means that [`handle_attest_meta`] has been called, and not having
    // an address entry yet means precisely that the callback hasn't finished
    // yet.

    let _ = wrapped_asset_seq_read(deps.storage, chain)
        .load(&token_address.serialize())
        .map_err(|_| ContractError::RegistrationForbidden.std())?;

    let mut bucket = wrapped_asset(deps.storage, chain);
    let result = bucket.load(&token_address.serialize()).ok();
    if result.is_some() {
        return ContractError::AssetAlreadyRegistered.std_err();
    }

    bucket.save(
        &token_address.serialize(),
        &WrappedCW20 {
            human_address: info.sender.clone(),
        },
    )?;

    let contract_address: CanonicalAddr = deps.api.addr_canonicalize(info.sender.as_str())?;
    is_wrapped_asset(deps.storage).save(contract_address.as_slice(), &())?;

    Ok(Response::new()
        .add_attribute("action", "register_asset")
        .add_attribute("token_chain", format!("{:?}", chain))
        .add_attribute("token_address", format!("{:?}", token_address))
        .add_attribute("contract_addr", info.sender))
}

fn handle_attest_meta(
    deps: DepsMut,
    env: Env,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    sequence: u64,
    data: &Vec<u8>,
) -> StdResult<Response> {
    let meta = AssetMeta::deserialize(data)?;

    let expected_contract =
        bridge_contracts_read(deps.storage).load(&emitter_chain.to_be_bytes())?;

    // must be sent by a registered token bridge contract
    if expected_contract != emitter_address {
        return Err(StdError::generic_err("invalid emitter"));
    }

    let token_id = meta
        .token_address
        .to_token_id(deps.storage, emitter_chain)?;

    let token_address: [u8; 32] = match token_id {
        TokenId::Bank { denom } => Err(StdError::generic_err(format!(
            "{} is native to this chain and should not be attested",
            denom
        ))),
        TokenId::Contract(ContractId::NativeCW20 { contract_address }) => {
            Err(StdError::generic_err(format!(
                "Contract {} is native to this chain and should not be attested",
                contract_address
            )))
        }
        TokenId::Contract(ContractId::ForeignToken {
            chain_id: _,
            foreign_address,
        }) => Ok(foreign_address),
    }?;

    let cfg = config_read(deps.storage).load()?;
    // If a CW20 wrapped already exists and this message has a newer sequence ID
    // we allow updating the metadata. If not, we create a brand new token.
    let message = if let Ok(contract) =
        wrapped_asset_read(deps.storage, meta.token_chain).load(token_address.as_slice())
    {
        // Prevent anyone from re-attesting with old VAAs.
        if sequence
            <= wrapped_asset_seq_read(deps.storage, meta.token_chain)
                .load(token_address.as_slice())?
        {
            return Err(StdError::generic_err(
                "this asset has already been attested",
            ));
        }
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: contract.into_string(),
            msg: to_binary(&WrappedMsg::UpdateMetadata {
                name: get_string_from_32(&meta.name),
                symbol: get_string_from_32(&meta.symbol),
            })?,
            funds: vec![],
        })
    } else {
        CosmosMsg::Wasm(WasmMsg::Instantiate {
            admin: Some(env.contract.address.clone().into_string()),
            code_id: cfg.wrapped_asset_code_id,
            msg: to_binary(&WrappedInit {
                name: get_string_from_32(&meta.name),
                symbol: get_string_from_32(&meta.symbol),
                asset_chain: meta.token_chain,
                asset_address: token_address.into(),
                decimals: min(meta.decimals, 8u8),
                mint: None,
                init_hook: Some(InitHook {
                    contract_addr: env.contract.address.to_string(),
                    msg: to_binary(&ExecuteMsg::RegisterAssetHook {
                        chain: meta.token_chain,
                        token_address: meta.token_address.clone(),
                    })?,
                }),
            })?,
            funds: vec![],
            label: "Wormhole Wrapped CW20".to_string(),
        })
    };
    wrapped_asset_seq(deps.storage, meta.token_chain).save(&token_address, &sequence)?;
    Ok(Response::new().add_message(message))
}

fn handle_create_asset_meta(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset_info: AssetInfo,
    nonce: u32,
) -> StdResult<Response> {
    match asset_info {
        AssetInfo::Token { contract_addr } => {
            handle_create_asset_meta_token(deps, env, info, contract_addr, nonce)
        }
        AssetInfo::NativeToken { ref denom } => {
            handle_create_asset_meta_native_token(deps, env, info, denom.clone(), nonce)
        }
    }
}

fn handle_create_asset_meta_token(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset_address: HumanAddr,
    nonce: u32,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

    let request = QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: asset_address.clone(),
        msg: to_binary(&TokenQuery::TokenInfo {})?,
    });

    let asset_human = deps.api.addr_validate(&asset_address)?;
    let token_id = TokenId::Contract(ContractId::NativeCW20 {
        contract_address: asset_human,
    });
    let external_id = token_id.store(deps.storage)?;
    let token_info: TokenInfoResponse = deps.querier.query(&request)?;

    let meta: AssetMeta = AssetMeta {
        token_chain: cfg.chain_id,
        token_address: external_id,
        decimals: token_info.decimals,
        symbol: extend_string_to_32(&token_info.symbol),
        name: extend_string_to_32(&token_info.name),
    };

    let token_bridge_message = TokenBridgeMessage {
        action: Action::ATTEST_META,
        payload: meta.serialize().to_vec(),
    };

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cfg.wormhole_contract,
            msg: to_binary(&WormholeExecuteMsg::PostMessage {
                message: Binary::from(token_bridge_message.serialize()),
                nonce,
            })?,
            // forward coins sent to this message
            funds: info.funds,
        }))
        .add_attribute("meta.token_chain", cfg.chain_id.to_string())
        .add_attribute("meta.token", asset_address)
        .add_attribute("meta.nonce", nonce.to_string())
        .add_attribute("meta.block_time", env.block.time.seconds().to_string()))
}

fn handle_create_asset_meta_native_token(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    denom: String,
    nonce: u32,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

    let symbol = format_native_denom_symbol(&denom);
    let token_id = TokenId::Bank { denom };
    let external_id = token_id.store(deps.storage)?;

    let meta: AssetMeta = AssetMeta {
        token_chain: cfg.chain_id,
        token_address: external_id.clone(),
        decimals: 6,
        symbol: extend_string_to_32(&symbol),
        name: extend_string_to_32(&symbol),
    };
    let token_bridge_message = TokenBridgeMessage {
        action: Action::ATTEST_META,
        payload: meta.serialize().to_vec(),
    };
    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cfg.wormhole_contract,
            msg: to_binary(&WormholeExecuteMsg::PostMessage {
                message: Binary::from(token_bridge_message.serialize()),
                nonce,
            })?,
            // forward coins sent to this message
            funds: info.funds,
        }))
        .add_attribute("meta.token_chain", cfg.chain_id.to_string())
        .add_attribute("meta.symbol", symbol)
        .add_attribute("meta.asset_id", hex::encode(external_id.serialize()))
        .add_attribute("meta.nonce", nonce.to_string())
        .add_attribute("meta.block_time", env.block.time.seconds().to_string()))
}

fn handle_complete_transfer_with_payload(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    data: &Binary,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
    let (vaa, payload) = parse_and_archive_vaa(deps.branch(), env.clone(), data)?;

    if let Either::Right(message) = payload {
        match message.action {
            Action::TRANSFER_WITH_PAYLOAD => handle_complete_transfer(
                deps,
                env,
                info,
                vaa.emitter_chain,
                vaa.emitter_address,
                TransferType::WithPayload { payload: () },
                &message.payload,
                relayer_address,
            ),
            _ => ContractError::InvalidVAAAction.std_err(),
        }
    } else {
        ContractError::InvalidVAAAction.std_err()
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

fn submit_vaa(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let (vaa, payload) = parse_and_archive_vaa(deps.branch(), env.clone(), data)?;
    match payload {
        Either::Left(governance_packet) => handle_governance_payload(deps, env, &governance_packet),
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
            Action::ATTEST_META => handle_attest_meta(
                deps,
                env,
                vaa.emitter_chain,
                vaa.emitter_address,
                vaa.sequence,
                &message.payload,
            ),
            _ => ContractError::InvalidVAAAction.std_err(),
        },
    }
}

fn handle_governance_payload(
    deps: DepsMut,
    env: Env,
    gov_packet: &GovernancePacket,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;
    let module = get_string_from_32(&gov_packet.module);

    if module != "TokenBridge" {
        return Err(StdError::generic_err("this is not a valid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != cfg.chain_id {
        return Err(StdError::generic_err(
            "the governance VAA is for another chain",
        ));
    }

    match gov_packet.action {
        1u8 => handle_register_chain(deps, env, &gov_packet.payload),
        2u8 => handle_upgrade_contract(deps, env, &gov_packet.payload),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
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

fn handle_register_chain(deps: DepsMut, _env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let RegisterChain {
        chain_id,
        chain_address,
    } = RegisterChain::deserialize(data)?;

    let existing = bridge_contracts_read(deps.storage).load(&chain_id.to_be_bytes());
    if existing.is_ok() {
        return Err(StdError::generic_err(
            "bridge contract already exists for this chain",
        ));
    }

    let mut bucket = bridge_contracts(deps.storage);
    bucket.save(&chain_id.to_be_bytes(), &chain_address)?;

    Ok(Response::new()
        .add_attribute("chain_id", chain_id.to_string())
        .add_attribute("chain_address", hex::encode(chain_address)))
}

#[allow(clippy::too_many_arguments)]
fn handle_complete_transfer(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    transfer_type: TransferType<()>,
    data: &Vec<u8>,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
    let transfer_info = TransferInfo::deserialize(data)?;
    let token_id = transfer_info
        .token_address
        .to_token_id(deps.storage, transfer_info.token_chain)?;
    match token_id {
        TokenId::Bank { denom } => handle_complete_transfer_token_native(
            deps,
            env,
            info,
            emitter_chain,
            emitter_address,
            denom,
            transfer_type,
            data,
            relayer_address,
        ),
        TokenId::Contract(contract) => handle_complete_transfer_token(
            deps,
            env,
            info,
            emitter_chain,
            emitter_address,
            contract,
            transfer_type,
            data,
            relayer_address,
        ),
    }
}

#[allow(clippy::too_many_arguments)]
#[allow(clippy::bind_instead_of_map)]
fn handle_complete_transfer_token(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    token_contract: ContractId,
    transfer_type: TransferType<()>,
    data: &Vec<u8>,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;
    let transfer_info = match transfer_type {
        TransferType::WithoutPayload => TransferInfo::deserialize(data)?,
        TransferType::WithPayload { payload: _ } => {
            TransferWithPayloadInfo::deserialize(data)?.as_transfer_info()
        }
    };

    let expected_contract =
        bridge_contracts_read(deps.storage).load(&emitter_chain.to_be_bytes())?;

    // must be sent by a registered token bridge contract
    if expected_contract != emitter_address {
        return Err(StdError::generic_err("invalid emitter"));
    }

    if transfer_info.recipient_chain != cfg.chain_id {
        return Err(StdError::generic_err(
            "this transfer is not directed at this chain",
        ));
    }

    let target_address = (&transfer_info.recipient.as_slice()).get_address(0);
    let recipient = deps.api.addr_humanize(&target_address)?;

    if let TransferType::WithPayload { payload: _ } = transfer_type {
        if recipient != info.sender {
            return Err(StdError::generic_err(
                "transfers with payload can only be redeemed by the recipient",
            ));
        }
    };

    let (not_supported_amount, mut amount) = transfer_info.amount;
    let (not_supported_fee, mut fee) = transfer_info.fee;

    amount = amount.checked_sub(fee).unwrap();

    // Check high 128 bit of amount value to be empty
    if not_supported_amount != 0 || not_supported_fee != 0 {
        return ContractError::AmountTooHigh.std_err();
    }

    let external_id = ExternalTokenId::from_token_id(&TokenId::Contract(token_contract.clone()))?;

    match token_contract {
        ContractId::ForeignToken {
            chain_id,
            foreign_address,
        } => {
            // Check if this asset is already deployed
            let contract_addr = wrapped_asset_read(deps.storage, chain_id).load(&foreign_address).
                or_else(|_| Err(StdError::generic_err("Wrapped asset not deployed. To deploy, invoke CreateWrapped with the associated AssetMeta")))?;

            let contract_addr = contract_addr.into_string();

            // Asset already deployed, just mint
            let mut messages = vec![CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: contract_addr.clone(),
                msg: to_binary(&WrappedMsg::Mint {
                    recipient: recipient.to_string(),
                    amount: Uint128::from(amount),
                })?,
                funds: vec![],
            })];
            if fee != 0 {
                messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                    contract_addr: contract_addr.clone(),
                    msg: to_binary(&WrappedMsg::Mint {
                        recipient: relayer_address.to_string(),
                        amount: Uint128::from(fee),
                    })?,
                    funds: vec![],
                }))
            }

            Ok(Response::new()
                .add_messages(messages)
                .add_attribute("action", "complete_transfer_wrapped")
                .add_attribute("contract", contract_addr)
                .add_attribute("recipient", recipient)
                .add_attribute("amount", amount.to_string())
                .add_attribute("relayer", relayer_address)
                .add_attribute("fee", fee.to_string()))
        }
        ContractId::NativeCW20 { contract_address } => {
            // note -- here the amount is the amount the recipient will receive;
            // amount + fee is the total sent
            receive_native(deps.storage, &external_id, Uint128::new(amount + fee))?;

            // undo normalization to 8 decimals
            let token_info: TokenInfoResponse =
                deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
                    contract_addr: contract_address.to_string(),
                    msg: to_binary(&TokenQuery::TokenInfo {})?,
                }))?;

            let decimals = token_info.decimals;
            let multiplier = 10u128.pow((max(decimals, 8u8) - 8u8) as u32);
            amount = amount.checked_mul(multiplier).unwrap();
            fee = fee.checked_mul(multiplier).unwrap();

            let mut messages = vec![CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: contract_address.to_string(),
                msg: to_binary(&TokenMsg::Transfer {
                    recipient: recipient.to_string(),
                    amount: Uint128::from(amount),
                })?,
                funds: vec![],
            })];

            if fee != 0 {
                messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                    contract_addr: contract_address.to_string(),
                    msg: to_binary(&TokenMsg::Transfer {
                        recipient: relayer_address.to_string(),
                        amount: Uint128::from(fee),
                    })?,
                    funds: vec![],
                }))
            }

            Ok(Response::new()
                .add_messages(messages)
                .add_attribute("action", "complete_transfer_native")
                .add_attribute("recipient", recipient)
                .add_attribute("contract", contract_address)
                .add_attribute("amount", amount.to_string())
                .add_attribute("relayer", relayer_address)
                .add_attribute("fee", fee.to_string()))
        }
    }
}

#[allow(clippy::too_many_arguments)]
fn handle_complete_transfer_token_native(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    denom: String,
    transfer_type: TransferType<()>,
    data: &Vec<u8>,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;
    let transfer_info = match transfer_type {
        TransferType::WithoutPayload => TransferInfo::deserialize(data)?,
        TransferType::WithPayload { payload: () } => {
            TransferWithPayloadInfo::deserialize(data)?.as_transfer_info()
        }
    };

    let expected_contract =
        bridge_contracts_read(deps.storage).load(&emitter_chain.to_be_bytes())?;

    // must be sent by a registered token bridge contract
    if expected_contract != emitter_address {
        return Err(StdError::generic_err("invalid emitter"));
    }

    if transfer_info.recipient_chain != cfg.chain_id {
        return Err(StdError::generic_err(
            "this transfer is not directed at this chain",
        ));
    }

    let target_address = (&transfer_info.recipient.as_slice()).get_address(0);
    let recipient = deps.api.addr_humanize(&target_address)?;

    if let TransferType::WithPayload { payload: _ } = transfer_type {
        if recipient != info.sender {
            return Err(StdError::generic_err(
                "transfers with payload can only be redeemed by the recipient",
            ));
        }
    };

    let (not_supported_amount, mut amount) = transfer_info.amount;
    let (not_supported_fee, fee) = transfer_info.fee;

    amount = amount.checked_sub(fee).unwrap();

    // Check high 128 bit of amount value to be empty
    if not_supported_amount != 0 || not_supported_fee != 0 {
        return ContractError::AmountTooHigh.std_err();
    }

    let external_address = ExternalTokenId::from_bank_token(&denom)?;
    receive_native(deps.storage, &external_address, Uint128::new(amount + fee))?;

    let mut messages = vec![CosmosMsg::Bank(BankMsg::Send {
        to_address: recipient.to_string(),
        amount: vec![coin(amount, &denom)],
    })];

    if fee != 0 {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: relayer_address.to_string(),
            amount: vec![coin(fee, &denom)],
        }));
    }

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("action", "complete_transfer_terra_native")
        .add_attribute("recipient", recipient)
        .add_attribute("denom", denom)
        .add_attribute("amount", amount.to_string())
        .add_attribute("relayer", relayer_address)
        .add_attribute("fee", fee.to_string()))
}

#[allow(clippy::too_many_arguments)]
fn handle_initiate_transfer(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset: Asset,
    recipient_chain: u16,
    recipient: [u8; 32],
    fee: Uint128,
    transfer_type: TransferType<Vec<u8>>,
    nonce: u32,
) -> StdResult<Response> {
    match asset.info {
        AssetInfo::Token { contract_addr } => handle_initiate_transfer_token(
            deps,
            env,
            info,
            contract_addr,
            asset.amount,
            recipient_chain,
            recipient,
            fee,
            transfer_type,
            nonce,
        ),
        AssetInfo::NativeToken { ref denom } => handle_initiate_transfer_native_token(
            deps,
            env,
            info,
            denom.clone(),
            asset.amount,
            recipient_chain,
            recipient,
            fee,
            transfer_type,
            nonce,
        ),
    }
}

#[allow(clippy::too_many_arguments)]
fn handle_initiate_transfer_token(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset: HumanAddr,
    mut amount: Uint128,
    recipient_chain: u16,
    recipient: [u8; 32],
    mut fee: Uint128,
    transfer_type: TransferType<Vec<u8>>,
    nonce: u32,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

    if recipient_chain == cfg.chain_id {
        return ContractError::SameSourceAndTarget.std_err();
    }
    if amount.is_zero() {
        return ContractError::AmountTooLow.std_err();
    }

    let asset_chain: u16;
    let asset_address: [u8; 32];

    let asset_canonical: CanonicalAddr = deps.api.addr_canonicalize(&asset)?;

    let mut messages: Vec<CosmosMsg> = vec![];
    let mut submessages: Vec<SubMsg> = vec![];

    // we'll only need this for payload 3 transfers
    let sender_address = deps.api.addr_canonicalize(&info.sender.to_string())?;
    let sender_address = extend_address_to_32_array(&sender_address);

    match is_wrapped_asset_read(deps.storage).load(asset_canonical.as_slice()) {
        Ok(_) => {
            // If the fee is too large the user will receive nothing.
            if fee > amount {
                return Err(StdError::generic_err("fee greater than sent amount"));
            }

            // This is a deployed wrapped asset, burn it
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: asset.clone(),
                msg: to_binary(&WrappedMsg::Burn {
                    account: info.sender.to_string(),
                    amount,
                })?,
                funds: vec![],
            }));
            let request = QueryRequest::<Empty>::Wasm(WasmQuery::Smart {
                contract_addr: asset,
                msg: to_binary(&WrappedQuery::WrappedAssetInfo {})?,
            });
            let wrapped_token_info: WrappedAssetInfoResponse = deps.querier.query(&request)?;
            asset_chain = wrapped_token_info.asset_chain;
            assert!(asset_chain != cfg.chain_id, "Expected a foreign chain id.");
            asset_address = wrapped_token_info.asset_address.to_array()?;

            let external_id = ExternalTokenId::from_foreign_token(asset_address);

            let token_bridge_message: TokenBridgeMessage = match transfer_type {
                TransferType::WithoutPayload => {
                    let transfer_info = TransferInfo {
                        amount: (0, amount.u128()),
                        token_address: external_id,
                        token_chain: asset_chain,
                        recipient,
                        recipient_chain,
                        fee: (0, fee.u128()),
                    };
                    TokenBridgeMessage {
                        action: Action::TRANSFER,
                        payload: transfer_info.serialize(),
                    }
                }
                TransferType::WithPayload { payload } => {
                    let transfer_info = TransferWithPayloadInfo {
                        amount: (0, amount.u128()),
                        token_address: external_id,
                        token_chain: asset_chain,
                        recipient,
                        recipient_chain,
                        sender_address,
                        payload,
                    };
                    TokenBridgeMessage {
                        action: Action::TRANSFER_WITH_PAYLOAD,
                        payload: transfer_info.serialize(),
                    }
                }
            };

            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: cfg.wormhole_contract,
                msg: to_binary(&WormholeExecuteMsg::PostMessage {
                    message: Binary::from(token_bridge_message.serialize()),
                    nonce,
                })?,
                // forward coins sent to this message
                funds: info.funds.clone(),
            }));
        }
        Err(_) => {
            asset_chain = cfg.chain_id;

            // normalize amount to 8 decimals when it sent over the wormhole
            let token_info: TokenInfoResponse =
                deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
                    contract_addr: asset.clone(),
                    msg: to_binary(&TokenQuery::TokenInfo {})?,
                }))?;

            let decimals = token_info.decimals;
            let multiplier = 10u128.pow((max(decimals, 8u8) - 8u8) as u32);

            // chop off dust
            amount = Uint128::new(
                amount
                    .u128()
                    .checked_sub(amount.u128().checked_rem(multiplier).unwrap())
                    .unwrap(),
            );

            fee = Uint128::new(
                fee.u128()
                    .checked_sub(fee.u128().checked_rem(multiplier).unwrap())
                    .unwrap(),
            );

            // This is a regular asset, transfer its balance
            submessages.push(SubMsg::reply_on_success(
                CosmosMsg::Wasm(WasmMsg::Execute {
                    contract_addr: asset.clone(),
                    msg: to_binary(&TokenMsg::TransferFrom {
                        owner: info.sender.to_string(),
                        recipient: env.contract.address.to_string(),
                        amount,
                    })?,
                    funds: vec![],
                }),
                1,
            ));

            asset_address = extend_address_to_32_array(&asset_canonical);
            let address_human = deps.api.addr_humanize(&asset_canonical)?;
            // we store here just in case the token is transferred out before it's attested
            let external_id = TokenId::Contract(ContractId::NativeCW20 {
                contract_address: address_human,
            })
            .store(deps.storage)?;

            // convert to normalized amounts before recording & posting vaa
            amount = Uint128::new(amount.u128().checked_div(multiplier).unwrap());
            fee = Uint128::new(fee.u128().checked_div(multiplier).unwrap());

            // Fetch current CW20 Balance pre-transfer.
            let balance: BalanceResponse =
                deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
                    contract_addr: asset.to_string(),
                    msg: to_binary(&TokenQuery::Balance {
                        address: env.contract.address.to_string(),
                    })?,
                }))?;

            // NOTE: Reentrancy protection. It is crucial that there's no
            // ongoing transfer in progress here, otherwise we would override
            // its state.  This could happen if the asset's TransferFrom handler
            // sends us an InitiateTransfer message, which would be executed
            // before the reply handler due the the depth-first semantics of
            // message execution.  A simple protection mechanism is to require
            // that there's no execution in progress. The reply handler takes
            // care of clearing out this temporary storage when done.
            assert!(wrapped_transfer_tmp(deps.storage).load().is_err());

            let token_bridge_message: TokenBridgeMessage = match transfer_type {
                TransferType::WithoutPayload => {
                    let transfer_info = TransferInfo {
                        amount: (0, amount.u128()),
                        token_address: external_id,
                        token_chain: asset_chain,
                        recipient,
                        recipient_chain,
                        fee: (0, fee.u128()),
                    };
                    TokenBridgeMessage {
                        action: Action::TRANSFER,
                        payload: transfer_info.serialize(),
                    }
                }
                TransferType::WithPayload { payload } => {
                    let transfer_info = TransferWithPayloadInfo {
                        amount: (0, amount.u128()),
                        token_address: external_id,
                        token_chain: asset_chain,
                        recipient,
                        recipient_chain,
                        sender_address,
                        payload,
                    };
                    TokenBridgeMessage {
                        action: Action::TRANSFER_WITH_PAYLOAD,
                        payload: transfer_info.serialize(),
                    }
                }
            };

            let token_address = deps.api.addr_validate(&asset)?;

            // Wrap up state to be captured by the submessage reply.
            wrapped_transfer_tmp(deps.storage).save(&TransferState {
                previous_balance: balance.balance.to_string(),
                account: info.sender.to_string(),
                token_address,
                message: token_bridge_message.serialize(),
                multiplier: Uint128::new(multiplier).to_string(),
                nonce,
            })?;
        }
    };

    Ok(Response::new()
        .add_messages(messages)
        .add_submessages(submessages)
        .add_attribute("transfer.token_chain", asset_chain.to_string())
        .add_attribute("transfer.token", hex::encode(asset_address))
        .add_attribute(
            "transfer.sender",
            hex::encode(extend_address_to_32(
                &deps.api.addr_canonicalize(info.sender.as_str())?,
            )),
        )
        .add_attribute("transfer.recipient_chain", recipient_chain.to_string())
        .add_attribute("transfer.recipient", hex::encode(recipient))
        .add_attribute("transfer.amount", amount.to_string())
        .add_attribute("transfer.nonce", nonce.to_string())
        .add_attribute("transfer.block_time", env.block.time.seconds().to_string()))
}

fn format_native_denom_symbol(denom: &str) -> String {
    if denom == "uluna" {
        return "LUNA".to_string();
    }
    //TODO: is there better formatting to do here?
    denom.to_uppercase()
}

#[allow(clippy::too_many_arguments)]
fn handle_initiate_transfer_native_token(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    denom: String,
    amount: Uint128,
    recipient_chain: u16,
    recipient: [u8; 32],
    fee: Uint128,
    transfer_type: TransferType<Vec<u8>>,
    nonce: u32,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

    if recipient_chain == cfg.chain_id {
        return ContractError::SameSourceAndTarget.std_err();
    }
    if amount.is_zero() {
        return ContractError::AmountTooLow.std_err();
    }
    if fee > amount {
        return Err(StdError::generic_err("fee greater than sent amount"));
    }

    let deposit_key = format!("{}:{}", info.sender, denom);
    bridge_deposit(deps.storage).update(deposit_key.as_bytes(), |current: Option<Uint128>| {
        match current {
            Some(v) => Ok(v.checked_sub(amount)?),
            None => Err(StdError::generic_err("no deposit found to transfer")),
        }
    })?;

    let mut messages: Vec<CosmosMsg> = vec![];

    let asset_chain: u16 = cfg.chain_id;
    // we store here just in case the token is transferred out before it's attested
    let asset_address = TokenId::Bank { denom }.store(deps.storage)?;

    send_native(deps.storage, &asset_address, amount)?;

    let token_bridge_message: TokenBridgeMessage = match transfer_type {
        TransferType::WithoutPayload => {
            let transfer_info = TransferInfo {
                amount: (0, amount.u128()),
                token_address: asset_address.clone(),
                token_chain: asset_chain,
                recipient,
                recipient_chain,
                fee: (0, fee.u128()),
            };
            TokenBridgeMessage {
                action: Action::TRANSFER,
                payload: transfer_info.serialize(),
            }
        }
        TransferType::WithPayload { payload } => {
            let sender_address = deps.api.addr_canonicalize(&info.sender.to_string())?;
            let sender_address = extend_address_to_32_array(&sender_address);
            let transfer_info = TransferWithPayloadInfo {
                amount: (0, amount.u128()),
                token_address: asset_address.clone(),
                token_chain: asset_chain,
                recipient,
                recipient_chain,
                sender_address,
                payload,
            };
            TokenBridgeMessage {
                action: Action::TRANSFER_WITH_PAYLOAD,
                payload: transfer_info.serialize(),
            }
        }
    };

    let sender = deps.api.addr_canonicalize(info.sender.as_str())?;
    messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: cfg.wormhole_contract,
        msg: to_binary(&WormholeExecuteMsg::PostMessage {
            message: Binary::from(token_bridge_message.serialize()),
            nonce,
        })?,
        funds: info.funds,
    }));

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("transfer.token_chain", asset_chain.to_string())
        .add_attribute("transfer.token", hex::encode(asset_address.serialize()))
        .add_attribute(
            "transfer.sender",
            hex::encode(extend_address_to_32(&sender)),
        )
        .add_attribute("transfer.recipient_chain", recipient_chain.to_string())
        .add_attribute("transfer.recipient", hex::encode(recipient))
        .add_attribute("transfer.amount", amount.to_string())
        .add_attribute("transfer.nonce", nonce.to_string())
        .add_attribute("transfer.block_time", env.block.time.seconds().to_string()))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::WrappedRegistry { chain, address } => {
            to_binary(&query_wrapped_registry(deps, chain, address.as_slice())?)
        }
        QueryMsg::TransferInfo { vaa } => to_binary(&query_transfer_info(deps, env, &vaa)?),
        QueryMsg::ExternalId { external_id } => to_binary(&query_external_id(deps, external_id)?),
    }
}

fn query_external_id(deps: Deps, external_id: Binary) -> StdResult<ExternalIdResponse> {
    let cfg = config_read(deps.storage).load()?;
    let external_id = ExternalTokenId::deserialize(external_id.to_array()?);
    Ok(ExternalIdResponse {
        token_id: external_id.to_token_id(deps.storage, cfg.chain_id)?,
    })
}

pub fn query_wrapped_registry(
    deps: Deps,
    chain: u16,
    address: &[u8],
) -> StdResult<WrappedRegistryResponse> {
    // Check if this asset is already deployed
    match wrapped_asset_read(deps.storage, chain).load(address) {
        Ok(address) => Ok(WrappedRegistryResponse {
            address: address.into_string(),
        }),
        Err(_) => ContractError::AssetNotFound.std_err(),
    }
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
        other => Err(StdError::generic_err(format!("Invalid action: {}", other))),
    }
}

fn is_governance_emitter(cfg: &ConfigInfo, emitter_chain: u16, emitter_address: &[u8]) -> bool {
    cfg.gov_chain == emitter_chain && cfg.gov_address == emitter_address
}
