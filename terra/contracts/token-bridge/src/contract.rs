use cw20::{BalanceResponse, TokenInfoResponse};
use cw20_base::msg::{ExecuteMsg as TokenMsg, QueryMsg as TokenQuery};
use cw20_wrapped::msg::{
    ExecuteMsg as WrappedMsg, InitHook, InstantiateMsg as WrappedInit, QueryMsg as WrappedQuery,
    WrappedAssetInfoResponse,
};
use sha3::{Digest, Keccak256};
use std::{
    cmp::{max, min},
    str::FromStr,
};
use terraswap::asset::{Asset, AssetInfo};

use classic_bindings::{TerraQuerier, TerraQuery};
use wormhole::{
    byte_utils::{
        extend_address_to_32, extend_address_to_32_array, extend_string_to_32, get_string_from_32,
        ByteUtils,
    },
    error::ContractError,
    msg::{ExecuteMsg as WormholeExecuteMsg, QueryMsg as WormholeQueryMsg},
    state::{vaa_archive_add, vaa_archive_check, GovernancePacket, ParsedVAA},
};

#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;

use cosmwasm_std::{
    coin, to_binary, BankMsg, Binary, CanonicalAddr, Coin, CosmosMsg, CustomQuery, Decimal, Deps,
    DepsMut, Env, MessageInfo, QuerierWrapper, QueryRequest, Reply, Response, StdError, StdResult,
    SubMsg, Uint128, WasmMsg, WasmQuery,
};

use crate::{
    msg::{
        ExecuteMsg, InstantiateMsg, MigrateMsg, QueryMsg, TransferInfoResponse,
        WrappedRegistryResponse,
    },
    state::{
        bridge_contracts, bridge_contracts_read, bridge_deposit, config, config_read,
        receive_native, send_native, wrapped_asset, wrapped_asset_address,
        wrapped_asset_address_read, wrapped_asset_read, wrapped_asset_seq, wrapped_asset_seq_read,
        wrapped_transfer_tmp, Action, AssetMeta, ConfigInfo, RegisterChain, TokenBridgeMessage,
        TransferInfo, TransferState, TransferWithPayloadInfo, UpgradeContract,
    },
};

type HumanAddr = String;

// Chain ID of Terra
const CHAIN_ID: u16 = 3;

const WRAPPED_ASSET_UPDATING: &str = "updating";
const WRAPPED_ASSET_LABEL: &str = "WrappedCW20";

pub enum TransferType<A> {
    WithoutPayload,
    WithPayload { payload: A },
}

/// Migration code that runs the next time the contract is upgraded.
/// This function will contain ephemeral code that we want to run once, and thus
/// can (and should be) safely deleted after the upgrade happened successfully.
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    // see [the token upgrades](../../../docs/token_upgrades.md) document for
    // information on upgrading the wrapped token contract.
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
            contract_addr: state.token_address.clone(),
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

    send_native(deps.storage, &state.token_canonical, real_amount)?;
    Ok(Response::default()
        .add_message(message)
        .add_attribute("action", "reply_handler"))
}

pub fn coins_after_tax(deps: DepsMut<TerraQuery>, coins: Vec<Coin>) -> StdResult<Vec<Coin>> {
    let mut res = vec![];
    for coin in coins {
        let asset = Asset {
            amount: coin.amount,
            info: AssetInfo::NativeToken {
                denom: coin.denom.clone(),
            },
        };
        res.push(deduct_tax(&asset, &deps.querier)?);
    }
    Ok(res)
}

fn parse_vaa<C: CustomQuery>(
    deps: Deps<C>,
    block_time: u64,
    data: &Binary,
) -> StdResult<ParsedVAA> {
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
pub fn execute(
    deps: DepsMut<TerraQuery>,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::RegisterAssetHook { asset_id } => {
            handle_register_asset(deps, env, info, asset_id.as_slice())
        }
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

fn deposit_tokens<C: CustomQuery>(
    deps: DepsMut<C>,
    _env: Env,
    info: MessageInfo,
) -> StdResult<Response> {
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
    deps: DepsMut<TerraQuery>,
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
            amount: coins_after_tax(deps, vec![coin(deposited_amount, &denom)])?,
        }));
    }

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("action", "withdraw_tokens"))
}

/// Handle wrapped asset registration messages
fn handle_register_asset<C: CustomQuery>(
    deps: DepsMut<C>,
    _env: Env,
    info: MessageInfo,
    asset_id: &[u8],
) -> StdResult<Response> {
    let mut bucket = wrapped_asset(deps.storage);
    let result = bucket.load(asset_id);
    let result = result.map_err(|_| ContractError::RegistrationForbidden.std())?;
    if result != WRAPPED_ASSET_UPDATING {
        return ContractError::AssetAlreadyRegistered.std_err();
    }

    bucket.save(asset_id, &info.sender.to_string())?;

    let contract_address: CanonicalAddr = deps.api.addr_canonicalize(info.sender.as_str())?;
    wrapped_asset_address(deps.storage).save(contract_address.as_slice(), &asset_id.to_vec())?;

    Ok(Response::new()
        .add_attribute("action", "register_asset")
        .add_attribute("asset_id", format!("{asset_id:?}"))
        .add_attribute("contract_addr", info.sender))
}

fn handle_attest_meta<C: CustomQuery>(
    deps: DepsMut<C>,
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

    if CHAIN_ID == meta.token_chain {
        return Err(StdError::generic_err(
            "this asset is native to this chain and should not be attested",
        ));
    }

    let cfg = config_read(deps.storage).load()?;
    let asset_id = build_asset_id(meta.token_chain, meta.token_address.as_slice());

    // If a CW20 wrapped already exists and this message has a newer sequence ID
    // we allow updating the metadata. If not, we create a brand new token.
    let message = if let Ok(contract) = wrapped_asset_read(deps.storage).load(&asset_id) {
        // Prevent anyone from re-attesting with old VAAs.
        if sequence <= wrapped_asset_seq_read(deps.storage).load(&asset_id)? {
            return Err(StdError::generic_err(
                "this asset has already been attested",
            ));
        }
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: contract,
            msg: to_binary(&WrappedMsg::UpdateMetadata {
                name: get_string_from_32(&meta.name),
                symbol: get_string_from_32(&meta.symbol),
            })?,
            funds: vec![],
        })
    } else {
        wrapped_asset(deps.storage).save(&asset_id, &HumanAddr::from(WRAPPED_ASSET_UPDATING))?;
        CosmosMsg::Wasm(WasmMsg::Instantiate {
            admin: Some(env.contract.address.clone().into_string()),
            code_id: cfg.wrapped_asset_code_id,
            msg: to_binary(&WrappedInit {
                name: get_string_from_32(&meta.name),
                symbol: get_string_from_32(&meta.symbol),
                asset_chain: meta.token_chain,
                asset_address: meta.token_address.to_vec().into(),
                decimals: min(meta.decimals, 8u8),
                mint: None,
                init_hook: Some(InitHook {
                    contract_addr: env.contract.address.to_string(),
                    msg: to_binary(&ExecuteMsg::RegisterAssetHook {
                        asset_id: asset_id.to_vec().into(),
                    })?,
                }),
            })?,
            funds: vec![],
            label: WRAPPED_ASSET_LABEL.to_string(),
        })
    };
    wrapped_asset_seq(deps.storage).save(&asset_id, &sequence)?;
    Ok(Response::new().add_message(message))
}

fn handle_create_asset_meta(
    deps: DepsMut<TerraQuery>,
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
    deps: DepsMut<TerraQuery>,
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

    let asset_canonical = deps.api.addr_canonicalize(&asset_address)?;
    let token_info: TokenInfoResponse = deps.querier.query(&request)?;

    let meta: AssetMeta = AssetMeta {
        token_chain: CHAIN_ID,
        token_address: extend_address_to_32(&asset_canonical),
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
            funds: coins_after_tax(deps, info.funds)?,
        }))
        .add_attribute("meta.token_chain", CHAIN_ID.to_string())
        .add_attribute("meta.token", asset_address)
        .add_attribute("meta.nonce", nonce.to_string())
        .add_attribute("meta.block_time", env.block.time.seconds().to_string()))
}

fn handle_create_asset_meta_native_token(
    deps: DepsMut<TerraQuery>,
    env: Env,
    info: MessageInfo,
    denom: String,
    nonce: u32,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;
    let mut asset_id = extend_address_to_32(&build_native_id(&denom).into());
    asset_id[0] = 1;
    let symbol = format_native_denom_symbol(&denom);
    let meta: AssetMeta = AssetMeta {
        token_chain: CHAIN_ID,
        token_address: asset_id.clone(),
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
            funds: coins_after_tax(deps, info.funds)?,
        }))
        .add_attribute("meta.token_chain", CHAIN_ID.to_string())
        .add_attribute("meta.symbol", symbol)
        .add_attribute("meta.asset_id", hex::encode(asset_id))
        .add_attribute("meta.nonce", nonce.to_string())
        .add_attribute("meta.block_time", env.block.time.seconds().to_string()))
}

fn handle_complete_transfer_with_payload(
    deps: DepsMut<TerraQuery>,
    env: Env,
    info: MessageInfo,
    data: &Binary,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
    let state = config_read(deps.storage).load()?;

    let vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), data)?;
    let data = vaa.payload;

    if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }
    vaa_archive_add(deps.storage, vaa.hash.as_slice())?;

    // check if vaa is from governance
    if is_governance_emitter(&state, vaa.emitter_chain, &vaa.emitter_address) {
        return ContractError::InvalidVAAAction.std_err();
    }

    let message = TokenBridgeMessage::deserialize(&data)?;

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
}

fn submit_vaa(
    deps: DepsMut<TerraQuery>,
    env: Env,
    info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let state = config_read(deps.storage).load()?;

    let vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), data)?;
    let data = vaa.payload;

    if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }
    vaa_archive_add(deps.storage, vaa.hash.as_slice())?;

    // check if vaa is from governance
    if is_governance_emitter(&state, vaa.emitter_chain, &vaa.emitter_address) {
        return handle_governance_payload(deps, env, &data);
    }

    let message = TokenBridgeMessage::deserialize(&data)?;

    match message.action {
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
    }
}

fn handle_governance_payload<C: CustomQuery>(
    deps: DepsMut<C>,
    env: Env,
    data: &[u8],
) -> StdResult<Response> {
    let gov_packet = GovernancePacket::deserialize(data)?;
    let module = get_string_from_32(&gov_packet.module);

    if module != "TokenBridge" {
        return Err(StdError::generic_err("this is not a valid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != CHAIN_ID {
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

fn handle_upgrade_contract<C: CustomQuery>(
    _deps: DepsMut<C>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<Response> {
    let UpgradeContract { new_contract } = UpgradeContract::deserialize(data)?;

    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(WasmMsg::Migrate {
            contract_addr: env.contract.address.to_string(),
            new_code_id: new_contract,
            msg: to_binary(&MigrateMsg {})?,
        }))
        .add_attribute("action", "contract_upgrade"))
}

fn handle_register_chain<C: CustomQuery>(
    deps: DepsMut<C>,
    _env: Env,
    data: &Vec<u8>,
) -> StdResult<Response> {
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
    deps: DepsMut<TerraQuery>,
    env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    transfer_type: TransferType<()>,
    data: &Vec<u8>,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
    let transfer_info = TransferInfo::deserialize(data)?;
    // see [the token id doc](../../../docs/token_id.md) for more info
    if transfer_info.token_chain == CHAIN_ID && is_native_id(transfer_info.token_address.as_slice())
    {
        handle_complete_transfer_token_native(
            deps,
            env,
            info,
            emitter_chain,
            emitter_address,
            transfer_type,
            data,
            relayer_address,
        )
    } else {
        handle_complete_transfer_token(
            deps,
            env,
            info,
            emitter_chain,
            emitter_address,
            transfer_type,
            data,
            relayer_address,
        )
    }
}

#[allow(clippy::too_many_arguments)]
fn handle_complete_transfer_token<C: CustomQuery>(
    deps: DepsMut<C>,
    _env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    transfer_type: TransferType<()>,
    data: &Vec<u8>,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
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

    if transfer_info.recipient_chain != CHAIN_ID {
        return Err(StdError::generic_err(
            "this transfer is not directed at this chain",
        ));
    }

    let token_chain = transfer_info.token_chain;
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

    if token_chain != CHAIN_ID {
        let asset_address = transfer_info.token_address;
        let asset_id = build_asset_id(token_chain, &asset_address);

        // Check if this asset is already deployed
        let contract_addr = wrapped_asset_read(deps.storage).load(&asset_id).ok();

        if let Some(contract_addr) = contract_addr {
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
        } else {
            Err(StdError::generic_err("Wrapped asset not deployed. To deploy, invoke CreateWrapped with the associated AssetMeta"))
        }
    } else {
        let token_address = transfer_info.token_address.as_slice().get_address(0);

        let contract_addr = deps.api.addr_humanize(&token_address)?;

        // note -- here the amount is the amount the recipient will receive;
        // amount + fee is the total sent
        receive_native(deps.storage, &token_address, Uint128::new(amount + fee))?;

        // undo normalization to 8 decimals
        let token_info: TokenInfoResponse =
            deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
                contract_addr: contract_addr.to_string(),
                msg: to_binary(&TokenQuery::TokenInfo {})?,
            }))?;

        let decimals = token_info.decimals;
        let multiplier = 10u128.pow((max(decimals, 8u8) - 8u8) as u32);
        amount = amount.checked_mul(multiplier).unwrap();
        fee = fee.checked_mul(multiplier).unwrap();

        let mut messages = vec![CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: contract_addr.to_string(),
            msg: to_binary(&TokenMsg::Transfer {
                recipient: recipient.to_string(),
                amount: Uint128::from(amount),
            })?,
            funds: vec![],
        })];

        if fee != 0 {
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: contract_addr.to_string(),
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
            .add_attribute("contract", contract_addr)
            .add_attribute("amount", amount.to_string())
            .add_attribute("relayer", relayer_address)
            .add_attribute("fee", fee.to_string()))
    }
}

#[allow(clippy::too_many_arguments)]
fn handle_complete_transfer_token_native(
    mut deps: DepsMut<TerraQuery>,
    _env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    transfer_type: TransferType<()>,
    data: &Vec<u8>,
    relayer_address: &HumanAddr,
) -> StdResult<Response> {
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

    if transfer_info.recipient_chain != CHAIN_ID {
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

    // Wipe the native byte marker and extract the serialized denom.
    let mut token_address = transfer_info.token_address;
    let token_address = token_address.as_mut_slice();
    token_address[0] = 0;

    let mut denom = token_address.to_vec();
    denom.retain(|&c| c != 0);
    let denom = String::from_utf8(denom).unwrap();

    // note -- here the amount is the amount the recipient will receive;
    // amount + fee is the total sent
    let token_address = (&*token_address).get_address(0);
    receive_native(deps.storage, &token_address, Uint128::new(amount + fee))?;

    let mut messages = vec![CosmosMsg::Bank(BankMsg::Send {
        to_address: recipient.to_string(),
        amount: coins_after_tax(deps.branch(), vec![coin(amount, &denom)])?,
    })];

    if fee != 0 {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: relayer_address.to_string(),
            amount: coins_after_tax(deps, vec![coin(fee, &denom)])?,
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
    deps: DepsMut<TerraQuery>,
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
    mut deps: DepsMut<TerraQuery>,
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
    if recipient_chain == CHAIN_ID {
        return ContractError::SameSourceAndTarget.std_err();
    }
    if amount.is_zero() {
        return ContractError::AmountTooLow.std_err();
    }

    let asset_chain: u16;
    let asset_address: [u8; 32];

    let cfg: ConfigInfo = config_read(deps.storage).load()?;
    let asset_canonical: CanonicalAddr = deps.api.addr_canonicalize(&asset)?;

    let mut messages: Vec<CosmosMsg> = vec![];
    let mut submessages: Vec<SubMsg> = vec![];

    // we'll only need this for payload 3 transfers
    let sender_address = deps.api.addr_canonicalize(info.sender.as_ref())?;
    let sender_address = extend_address_to_32_array(&sender_address);

    match wrapped_asset_address_read(deps.storage).load(asset_canonical.as_slice()) {
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
            let request = QueryRequest::Wasm(WasmQuery::Smart {
                contract_addr: asset,
                msg: to_binary(&WrappedQuery::WrappedAssetInfo {})?,
            });
            let wrapped_token_info: WrappedAssetInfoResponse = deps.querier.query(&request)?;
            asset_chain = wrapped_token_info.asset_chain;
            asset_address = wrapped_token_info.asset_address.to_array()?;

            let token_bridge_message: TokenBridgeMessage = match transfer_type {
                TransferType::WithoutPayload => {
                    let transfer_info = TransferInfo {
                        token_chain: asset_chain,
                        token_address: asset_address,
                        amount: (0, amount.u128()),
                        recipient_chain,
                        recipient,
                        fee: (0, fee.u128()),
                    };
                    TokenBridgeMessage {
                        action: Action::TRANSFER,
                        payload: transfer_info.serialize(),
                    }
                }
                TransferType::WithPayload { payload } => {
                    let transfer_info = TransferWithPayloadInfo {
                        token_chain: asset_chain,
                        token_address: asset_address,
                        amount: (0, amount.u128()),
                        recipient_chain,
                        recipient,
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
                funds: coins_after_tax(deps.branch(), info.funds)?,
            }));
        }
        Err(_) => {
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
            asset_chain = CHAIN_ID;

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
                        token_address: asset_address,
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
                        token_address: asset_address,
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

            // Wrap up state to be captured by the submessage reply.
            wrapped_transfer_tmp(deps.storage).save(&TransferState {
                previous_balance: balance.balance.to_string(),
                account: info.sender.to_string(),
                token_address: asset,
                token_canonical: asset_canonical.clone(),
                message: token_bridge_message.serialize(),
                multiplier: Uint128::new(multiplier).to_string(),
                nonce,
            })?;
        }
    };

    // Ensure that the asset's address does not collide with the native
    // address format. This is impossible for legacy CW20 addresses as they are
    // 20 bytes long left padded with 0s, so their first byte can't be 1.
    // However, it's theoretically possible for a new 32 byte CW20 address to have
    // this format. The probability of this happening is 1 / 2^96 ≈ 1.2 * 10^-29,
    // so it is negligible. Regardless, we block such addresses here
    // for the sake of completeness and documentation.
    assert!(!is_native_id(&asset_address));

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

/// All ISO-4217 currency codes are 3 letters, so we can safely slice anything that is not ULUNA.
/// https://www.xe.com/iso4217.php
fn format_native_denom_symbol(denom: &str) -> String {
    if denom == "uluna" {
        return "LUNA".to_string();
    }
    // UUSD -> US -> UST
    denom.to_uppercase()[1..3].to_string() + "T"
}

#[allow(clippy::too_many_arguments)]
fn handle_initiate_transfer_native_token(
    deps: DepsMut<TerraQuery>,
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
    if recipient_chain == CHAIN_ID {
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

    let cfg: ConfigInfo = config_read(deps.storage).load()?;
    let mut messages: Vec<CosmosMsg> = vec![];

    let asset_chain: u16 = CHAIN_ID;
    let asset_address: CanonicalAddr = build_native_id(&denom).into();

    send_native(deps.storage, &asset_address, amount)?;

    // Mark the first byte of the address to distinguish it as native.
    // NOTE: Since the asset's address 20 bytes long, it will get left padded
    // with 12 bytes of zeros, meaning that after the marker byte adjustment,
    // the address is [1] ++ [0; 11], i.e. a single 1 byte followed by eleven 0
    // bytes.  We maintain the global invariant that only native bank denoms
    // have the first 12 bytes in this format. Since there is a theoretical
    // probability that a 32 byte CW20 address could collide with this format,
    // we block such addresses on the way out.
    let mut asset_address = extend_address_to_32_array(&asset_address);
    asset_address[0] = 1;

    // sanity check, this will always pass
    assert!(is_native_id(&asset_address));

    let token_bridge_message: TokenBridgeMessage = match transfer_type {
        TransferType::WithoutPayload => {
            let transfer_info = TransferInfo {
                amount: (0, amount.u128()),
                token_address: asset_address,
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
            let sender_address = deps.api.addr_canonicalize(info.sender.as_ref())?;
            let sender_address = extend_address_to_32_array(&sender_address);
            let transfer_info = TransferWithPayloadInfo {
                amount: (0, amount.u128()),
                token_address: asset_address,
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
        funds: coins_after_tax(deps, info.funds)?,
    }));

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("transfer.token_chain", asset_chain.to_string())
        .add_attribute("transfer.token", hex::encode(asset_address))
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
    }
}

pub fn query_wrapped_registry(
    deps: Deps,
    chain: u16,
    address: &[u8],
) -> StdResult<WrappedRegistryResponse> {
    let asset_id = build_asset_id(chain, address);
    // Check if this asset is already deployed
    match wrapped_asset_read(deps.storage).load(&asset_id) {
        Ok(address) => Ok(WrappedRegistryResponse { address }),
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
                token_address: core.token_address,
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
                token_address: core.token_address,
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

pub fn build_asset_id(chain: u16, address: &[u8]) -> Vec<u8> {
    let chain = &chain.to_be_bytes();
    let mut asset_id = Vec::with_capacity(chain.len() + address.len());
    asset_id.extend_from_slice(chain);
    asset_id.extend_from_slice(address);

    let mut hasher = Keccak256::new();
    hasher.update(asset_id);
    hasher.finalize().to_vec()
}

// Produce a 20 byte asset "address" from a native terra denom.
pub fn build_native_id(denom: &str) -> Vec<u8> {
    let n = denom.len();
    assert!(n < 20);
    let mut asset_address = Vec::with_capacity(20);
    asset_address.resize(20 - n, 0u8);
    asset_address.extend_from_slice(denom.as_bytes());
    asset_address
}

/// Check that the first byte of the address is 1 and the remaining 11 bytes are 0.
/// For more information, see the comment in [`handle_initiate_transfer_native_token`].
fn is_native_id(address: &[u8]) -> bool {
    address[0] == 1 && address[1..12].iter().all(|&x| x == 0)
}

fn is_governance_emitter(cfg: &ConfigInfo, emitter_chain: u16, emitter_address: &[u8]) -> bool {
    cfg.gov_chain == emitter_chain && cfg.gov_address == emitter_address
}

////////////////////////////////////////////////////////////////////////////////
// Tax calculation

// the code below has been lifted from
// https://github.com/terraswap/terraswap/blob/7cf47f5e811fe0c4643a7cd09500702c1e7f3a6b/packages/terraswap/src/asset.rs#L25-L64
// but with luna tax enabled instead of defaulting it to 0

static DECIMAL_FRACTION: Uint128 = Uint128::new(1_000_000_000_000_000_000u128);

pub fn compute_tax(asset: &Asset, querier: &QuerierWrapper<TerraQuery>) -> StdResult<Uint128> {
    let amount = asset.amount;
    if let AssetInfo::NativeToken { denom } = &asset.info {
        let terra_querier = TerraQuerier::new(querier);
        let tax_rate: Decimal = (terra_querier.query_tax_rate()?).rate;
        let tax_cap: Uint128 = (terra_querier.query_tax_cap(denom.to_string())?).cap;
        Ok(std::cmp::min(
            amount.checked_sub(amount.multiply_ratio(
                DECIMAL_FRACTION,
                DECIMAL_FRACTION * tax_rate + DECIMAL_FRACTION,
            ))?,
            tax_cap,
        ))
    } else {
        Ok(Uint128::zero())
    }
}

pub fn deduct_tax(asset: &Asset, querier: &QuerierWrapper<TerraQuery>) -> StdResult<Coin> {
    let amount = asset.amount;
    if let AssetInfo::NativeToken { denom } = &asset.info {
        Ok(Coin {
            denom: denom.to_string(),
            amount: amount.checked_sub(compute_tax(asset, querier)?)?,
        })
    } else {
        Err(StdError::generic_err("cannot deduct tax from token asset"))
    }
}
