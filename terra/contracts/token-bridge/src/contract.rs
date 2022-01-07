use crate::msg::WrappedRegistryResponse;
use cosmwasm_std::{
    coin,
    entry_point,
    to_binary,
    BankMsg,
    Binary,
    CanonicalAddr,
    Coin,
    CosmosMsg,
    Deps,
    DepsMut,
    Empty,
    Env,
    MessageInfo,
    Order,
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
        InstantiateMsg,
        MigrateMsg,
        QueryMsg,
    },
    state::{
        bridge_contracts,
        bridge_contracts_read,
        bridge_deposit,
        config,
        config_read,
        receive_native,
        send_native,
        wrapped_asset,
        wrapped_asset_address,
        wrapped_asset_address_read,
        wrapped_asset_read,
        wrapped_asset_seq,
        wrapped_asset_seq_read,
        wrapped_transfer_tmp,
        Action,
        AssetMeta,
        ConfigInfo,
        RegisterChain,
        TokenBridgeMessage,
        TransferInfo,
        TransferState,
        UpgradeContract,
    },
};
use wormhole::{
    byte_utils::{
        extend_address_to_32,
        extend_string_to_32,
        get_string_from_32,
        ByteUtils,
    },
    error::ContractError,
};

use cw20_base::msg::{
    ExecuteMsg as TokenMsg,
    QueryMsg as TokenQuery,
};

use wormhole::msg::{
    ExecuteMsg as WormholeExecuteMsg,
    QueryMsg as WormholeQueryMsg,
};

use wormhole::state::{
    vaa_archive_add,
    vaa_archive_check,
    GovernancePacket,
    ParsedVAA,
};

use cw20::{
    BalanceResponse,
    TokenInfoResponse,
};

use cw20_wrapped::msg::{
    ExecuteMsg as WrappedMsg,
    InitHook,
    InstantiateMsg as WrappedInit,
    QueryMsg as WrappedQuery,
    WrappedAssetInfoResponse,
};
use terraswap::asset::{
    Asset,
    AssetInfo,
};

use sha3::{
    Digest,
    Keccak256,
};
use std::{
    cmp::{
        max,
        min,
    },
    str::FromStr,
};

type HumanAddr = String;

// Chain ID of Terra
const CHAIN_ID: u16 = 3;

const WRAPPED_ASSET_UPDATING: &str = "updating";

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    let bucket = wrapped_asset_address(deps.storage);
    let mut messages = vec![];
    for item in bucket.range(None, None, Order::Ascending) {
        let contract_address = item?.0;
        messages.push(CosmosMsg::Wasm(WasmMsg::Migrate {
            contract_addr: deps
                .api
                .addr_humanize(&contract_address.into())?
                .to_string(),
            new_code_id: 767,
            msg: to_binary(&MigrateMsg {})?,
        }));
    }

    let count = messages.len();

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("migrate", "upgrade cw20 wrappers")
        .add_attribute("count", count.to_string()))
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
        gov_address: msg.gov_address.as_slice().to_vec(),
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
    let mut info = TransferInfo::deserialize(&state.message)?;

    // Fetch CW20 Balance post-transfer.
    let new_balance: BalanceResponse = deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
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
    if info.fee.1 > real_amount.u128() {
        return Err(StdError::generic_err("fee greater than sent amount"));
    }

    // Update Wormhole message to correct amount.
    info.amount.1 = real_amount.u128();

    let token_bridge_message = TokenBridgeMessage {
        action: Action::TRANSFER,
        payload: info.serialize(),
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

    send_native(deps.storage, &state.token_canonical, info.amount.1.into())?;
    Ok(Response::default()
        .add_message(message)
        .add_attribute("action", "reply_handler"))
}

pub fn coins_after_tax(deps: DepsMut, coins: Vec<Coin>) -> StdResult<Vec<Coin>> {
    let mut res = vec![];
    for coin in coins {
        let asset = Asset {
            amount: coin.amount.clone(),
            info: AssetInfo::NativeToken {
                denom: coin.denom.clone(),
            },
        };
        res.push(asset.deduct_tax(&deps.querier)?);
    }
    Ok(res)
}

pub fn parse_vaa(deps: DepsMut, block_time: u64, data: &Binary) -> StdResult<ParsedVAA> {
    let cfg = config_read(deps.storage).load()?;
    let vaa: ParsedVAA = deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: cfg.wormhole_contract.clone(),
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
        ExecuteMsg::RegisterAssetHook { asset_id } => {
            handle_register_asset(deps, env, info, &asset_id.as_slice())
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
            recipient.as_slice().to_vec(),
            fee,
            nonce,
        ),
        ExecuteMsg::DepositTokens {} => deposit_tokens(deps, env, info),
        ExecuteMsg::WithdrawTokens { asset } => withdraw_tokens(deps, env, info, asset),
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),
        ExecuteMsg::CreateAssetMeta { asset_info, nonce } => {
            handle_create_asset_meta(deps, env, info, asset_info, nonce)
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
            amount: coins_after_tax(deps, vec![coin(deposited_amount, &denom)])?,
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
    asset_id: &[u8],
) -> StdResult<Response> {
    let mut bucket = wrapped_asset(deps.storage);
    let result = bucket.load(asset_id);
    let result = result.map_err(|_| ContractError::RegistrationForbidden.std())?;
    if result != HumanAddr::from(WRAPPED_ASSET_UPDATING) {
        return ContractError::AssetAlreadyRegistered.std_err();
    }

    bucket.save(asset_id, &info.sender.to_string())?;

    let contract_address: CanonicalAddr = deps.api.addr_canonicalize(&info.sender.as_str())?;
    wrapped_asset_address(deps.storage).save(contract_address.as_slice(), &asset_id.to_vec())?;

    Ok(Response::new()
        .add_attribute("action", "register_asset")
        .add_attribute("asset_id", format!("{:?}", asset_id))
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

    if CHAIN_ID == meta.token_chain {
        return Err(StdError::generic_err(
            "this asset is native to this chain and should not be attested",
        ));
    }

    let cfg = config_read(deps.storage).load()?;
    let asset_id = build_asset_id(meta.token_chain, &meta.token_address.as_slice());

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
            label: String::new(),
        })
    };
    wrapped_asset_seq(deps.storage).save(&asset_id, &sequence)?;
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
            funds: coins_after_tax(deps, info.funds.clone())?,
        }))
        .add_attribute("meta.token_chain", CHAIN_ID.to_string())
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
            funds: coins_after_tax(deps, info.funds.clone())?,
        }))
        .add_attribute("meta.token_chain", CHAIN_ID.to_string())
        .add_attribute("meta.symbol", symbol)
        .add_attribute("meta.asset_id", hex::encode(asset_id))
        .add_attribute("meta.nonce", nonce.to_string())
        .add_attribute("meta.block_time", env.block.time.seconds().to_string()))
}

fn submit_vaa(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let state = config_read(deps.storage).load()?;

    let vaa = parse_vaa(deps.branch(), env.block.time.seconds(), data)?;
    let data = vaa.payload;

    if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }
    vaa_archive_add(deps.storage, vaa.hash.as_slice())?;

    // check if vaa is from governance
    if state.gov_chain == vaa.emitter_chain && state.gov_address == vaa.emitter_address {
        return handle_governance_payload(deps, env, &data);
    }

    let message = TokenBridgeMessage::deserialize(&data)?;

    match message.action {
        Action::TRANSFER => handle_complete_transfer(
            deps,
            env,
            info,
            vaa.emitter_chain,
            vaa.emitter_address,
            &message.payload,
        ),
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

fn handle_governance_payload(deps: DepsMut, env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let gov_packet = GovernancePacket::deserialize(&data)?;
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

fn handle_upgrade_contract(_deps: DepsMut, env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let UpgradeContract { new_contract } = UpgradeContract::deserialize(&data)?;

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
    } = RegisterChain::deserialize(&data)?;

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

fn handle_complete_transfer(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    data: &Vec<u8>,
) -> StdResult<Response> {
    let transfer_info = TransferInfo::deserialize(&data)?;
    match transfer_info.token_address.as_slice()[0] {
        1 => handle_complete_transfer_token_native(
            deps,
            env,
            info,
            emitter_chain,
            emitter_address,
            data,
        ),
        _ => handle_complete_transfer_token(deps, env, info, emitter_chain, emitter_address, data),
    }
}

fn handle_complete_transfer_token(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    data: &Vec<u8>,
) -> StdResult<Response> {
    let transfer_info = TransferInfo::deserialize(&data)?;
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

        return if let Some(contract_addr) = contract_addr {
            // Asset already deployed, just mint

            let recipient = deps
                .api
                .addr_humanize(&target_address)
                .or_else(|_| ContractError::WrongTargetAddressFormat.std_err())?;

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
                        recipient: info.sender.to_string(),
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
                .add_attribute("amount", amount.to_string()))
        } else {
            Err(StdError::generic_err("Wrapped asset not deployed. To deploy, invoke CreateWrapped with the associated AssetMeta"))
        };
    } else {
        let token_address = transfer_info.token_address.as_slice().get_address(0);

        let recipient = deps.api.addr_humanize(&target_address)?;
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
                    recipient: info.sender.to_string(),
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
            .add_attribute("amount", amount.to_string()))
    }
}

fn handle_complete_transfer_token_native(
    mut deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    data: &Vec<u8>,
) -> StdResult<Response> {
    let transfer_info = TransferInfo::deserialize(&data)?;

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

    let (not_supported_amount, mut amount) = transfer_info.amount;
    let (not_supported_fee, fee) = transfer_info.fee;

    amount = amount.checked_sub(fee).unwrap();

    // Check high 128 bit of amount value to be empty
    if not_supported_amount != 0 || not_supported_fee != 0 {
        return ContractError::AmountTooHigh.std_err();
    }

    // Wipe the native byte marker and extract the serialized denom.
    let mut token_address = transfer_info.token_address.clone();
    let token_address = token_address.as_mut_slice();
    token_address[0] = 0;

    let mut denom = token_address.to_vec();
    denom.retain(|&c| c != 0);
    let denom = String::from_utf8(denom).unwrap();

    // note -- here the amount is the amount the recipient will receive;
    // amount + fee is the total sent
    let recipient = deps.api.addr_humanize(&target_address)?;
    let token_address = (&*token_address).get_address(0);
    receive_native(deps.storage, &token_address, Uint128::new(amount + fee))?;

    let mut messages = vec![CosmosMsg::Bank(BankMsg::Send {
        to_address: recipient.to_string(),
        amount: coins_after_tax(deps.branch(), vec![coin(amount, &denom)])?,
    })];

    if fee != 0 {
        messages.push(CosmosMsg::Bank(BankMsg::Send {
            to_address: recipient.to_string(),
            amount: coins_after_tax(deps, vec![coin(fee, &denom)])?,
        }));
    }

    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("action", "complete_transfer_terra_native")
        .add_attribute("recipient", recipient)
        .add_attribute("denom", denom)
        .add_attribute("amount", amount.to_string()))
}

fn handle_initiate_transfer(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset: Asset,
    recipient_chain: u16,
    recipient: Vec<u8>,
    fee: Uint128,
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
            nonce,
        ),
    }
}

fn handle_initiate_transfer_token(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset: HumanAddr,
    mut amount: Uint128,
    recipient_chain: u16,
    recipient: Vec<u8>,
    mut fee: Uint128,
    nonce: u32,
) -> StdResult<Response> {
    if recipient_chain == CHAIN_ID {
        return ContractError::SameSourceAndTarget.std_err();
    }
    if amount.is_zero() {
        return ContractError::AmountTooLow.std_err();
    }

    let asset_chain: u16;
    let asset_address: Vec<u8>;

    let cfg: ConfigInfo = config_read(deps.storage).load()?;
    let asset_canonical: CanonicalAddr = deps.api.addr_canonicalize(&asset)?;

    let mut messages: Vec<CosmosMsg> = vec![];
    let mut submessages: Vec<SubMsg> = vec![];

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
            let request = QueryRequest::<Empty>::Wasm(WasmQuery::Smart {
                contract_addr: asset,
                msg: to_binary(&WrappedQuery::WrappedAssetInfo {})?,
            });
            let wrapped_token_info: WrappedAssetInfoResponse =
                deps.querier.custom_query(&request)?;
            asset_chain = wrapped_token_info.asset_chain;
            asset_address = wrapped_token_info.asset_address.as_slice().to_vec();

            let transfer_info = TransferInfo {
                token_chain: asset_chain,
                token_address: asset_address.clone(),
                amount: (0, amount.u128()),
                recipient_chain,
                recipient: recipient.clone(),
                fee: (0, fee.u128()),
            };

            let token_bridge_message = TokenBridgeMessage {
                action: Action::TRANSFER,
                payload: transfer_info.serialize(),
            };

            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: cfg.wormhole_contract,
                msg: to_binary(&WormholeExecuteMsg::PostMessage {
                    message: Binary::from(token_bridge_message.serialize()),
                    nonce,
                })?,
                // forward coins sent to this message
                funds: coins_after_tax(deps.branch(), info.funds.clone())?,
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

            asset_address = extend_address_to_32(&asset_canonical);
            asset_chain = CHAIN_ID;

            // convert to normalized amounts before recording & posting vaa
            amount = Uint128::new(amount.u128().checked_div(multiplier).unwrap());
            fee = Uint128::new(fee.u128().checked_div(multiplier).unwrap());

            let transfer_info = TransferInfo {
                token_chain: asset_chain,
                token_address: asset_address.clone(),
                amount: (0, amount.u128()),
                recipient_chain,
                recipient: recipient.clone(),
                fee: (0, fee.u128()),
            };

            // Fetch current CW20 Balance pre-transfer.
            let balance: BalanceResponse =
                deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
                    contract_addr: asset.to_string(),
                    msg: to_binary(&TokenQuery::Balance {
                        address: env.contract.address.to_string(),
                    })?,
                }))?;

            // Wrap up state to be captured by the submessage reply.
            wrapped_transfer_tmp(deps.storage).save(&TransferState {
                previous_balance: balance.balance.to_string(),
                account: info.sender.to_string(),
                token_address: asset,
                token_canonical: asset_canonical.clone(),
                message: transfer_info.serialize(),
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
                &deps.api.addr_canonicalize(&info.sender.as_str())?,
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

fn handle_initiate_transfer_native_token(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    denom: String,
    amount: Uint128,
    recipient_chain: u16,
    recipient: Vec<u8>,
    fee: Uint128,
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
    let mut asset_address: Vec<u8> = build_native_id(&denom);

    send_native(deps.storage, &asset_address[..].into(), amount)?;

    // Mark the first byte of the address to distinguish it as native.
    asset_address = extend_address_to_32(&asset_address.into());
    asset_address[0] = 1;

    let transfer_info = TransferInfo {
        token_chain: asset_chain,
        token_address: asset_address.to_vec(),
        amount: (0, amount.u128()),
        recipient_chain,
        recipient: recipient.clone(),
        fee: (0, fee.u128()),
    };

    let token_bridge_message = TokenBridgeMessage {
        action: Action::TRANSFER,
        payload: transfer_info.serialize(),
    };

    let sender = deps.api.addr_canonicalize(&info.sender.as_str())?;
    messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: cfg.wormhole_contract,
        msg: to_binary(&WormholeExecuteMsg::PostMessage {
            message: Binary::from(token_bridge_message.serialize()),
            nonce,
        })?,
        funds: coins_after_tax(deps, info.funds.clone())?,
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
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::WrappedRegistry { chain, address } => {
            to_binary(&query_wrapped_registry(deps, chain, address.as_slice())?)
        }
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

fn build_asset_id(chain: u16, address: &[u8]) -> Vec<u8> {
    let mut asset_id: Vec<u8> = vec![];
    asset_id.extend_from_slice(&chain.to_be_bytes());
    asset_id.extend_from_slice(address);

    let mut hasher = Keccak256::new();
    hasher.update(asset_id);
    hasher.finalize().to_vec()
}

// Produce a 20 byte asset "address" from a native terra denom.
fn build_native_id(denom: &str) -> Vec<u8> {
    let mut asset_address: Vec<u8> = denom.clone().as_bytes().to_vec();
    asset_address.reverse();
    asset_address.extend(vec![0u8; 20 - denom.len()]);
    asset_address.reverse();
    assert_eq!(asset_address.len(), 20);
    asset_address
}

#[cfg(test)]
mod tests {
    use cosmwasm_std::{
        to_binary,
        Binary,
        StdResult,
    };

    #[test]
    fn test_me() -> StdResult<()> {
        let x = vec![
            1u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 96u8, 180u8, 94u8, 195u8, 0u8, 0u8,
            0u8, 1u8, 0u8, 3u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 0u8, 38u8,
            229u8, 4u8, 215u8, 149u8, 163u8, 42u8, 54u8, 156u8, 236u8, 173u8, 168u8, 72u8, 220u8,
            100u8, 90u8, 154u8, 159u8, 160u8, 215u8, 0u8, 91u8, 48u8, 44u8, 48u8, 44u8, 51u8, 44u8,
            48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8,
            48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 53u8, 55u8, 44u8, 52u8,
            54u8, 44u8, 50u8, 53u8, 53u8, 44u8, 53u8, 48u8, 44u8, 50u8, 52u8, 51u8, 44u8, 49u8,
            48u8, 54u8, 44u8, 49u8, 50u8, 50u8, 44u8, 49u8, 49u8, 48u8, 44u8, 49u8, 50u8, 53u8,
            44u8, 56u8, 56u8, 44u8, 55u8, 51u8, 44u8, 49u8, 56u8, 57u8, 44u8, 50u8, 48u8, 55u8,
            44u8, 49u8, 48u8, 52u8, 44u8, 56u8, 51u8, 44u8, 49u8, 49u8, 57u8, 44u8, 49u8, 50u8,
            55u8, 44u8, 49u8, 57u8, 50u8, 44u8, 49u8, 52u8, 55u8, 44u8, 56u8, 57u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 51u8, 44u8, 50u8, 51u8, 50u8, 44u8, 48u8, 44u8, 51u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8,
            44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 48u8, 44u8, 53u8, 51u8, 44u8, 49u8, 49u8,
            54u8, 44u8, 52u8, 56u8, 44u8, 49u8, 49u8, 54u8, 44u8, 49u8, 52u8, 57u8, 44u8, 49u8,
            48u8, 56u8, 44u8, 49u8, 49u8, 51u8, 44u8, 56u8, 44u8, 48u8, 44u8, 50u8, 51u8, 50u8,
            44u8, 52u8, 57u8, 44u8, 49u8, 53u8, 50u8, 44u8, 49u8, 44u8, 50u8, 56u8, 44u8, 50u8,
            48u8, 51u8, 44u8, 50u8, 49u8, 50u8, 44u8, 50u8, 50u8, 49u8, 44u8, 50u8, 52u8, 49u8,
            44u8, 56u8, 53u8, 44u8, 49u8, 48u8, 57u8, 93u8,
        ];
        let b = Binary::from(x.clone());
        let y = b.as_slice().to_vec();
        assert_eq!(x, y);
        Ok(())
    }
}
