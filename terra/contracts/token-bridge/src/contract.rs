use crate::msg::WrappedRegistryResponse;
use cosmwasm_std::{
    log,
    to_binary,
    Api,
    Binary,
    CanonicalAddr,
    Coin,
    CosmosMsg,
    Env,
    Extern,
    HandleResponse,
    HumanAddr,
    InitResponse,
    Querier,
    QueryRequest,
    StdError,
    StdResult,
    Storage,
    Uint128,
    WasmMsg,
    WasmQuery,
};

use crate::{
    msg::{
        HandleMsg,
        InitMsg,
        QueryMsg,
    },
    state::{
        bridge_contracts,
        bridge_contracts_read,
        config,
        config_read,
        receive_native,
        send_native,
        wrapped_asset,
        wrapped_asset_address,
        wrapped_asset_address_read,
        wrapped_asset_read,
        Action,
        AssetMeta,
        ConfigInfo,
        RegisterChain,
        TokenBridgeMessage,
        TransferInfo,
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
    HandleMsg as TokenMsg,
    QueryMsg as TokenQuery,
};

use wormhole::msg::{
    HandleMsg as WormholeHandleMsg,
    QueryMsg as WormholeQueryMsg,
};

use wormhole::state::{
    vaa_archive_add,
    vaa_archive_check,
    GovernancePacket,
    ParsedVAA,
};

use cw20::TokenInfoResponse;

use cw20_wrapped::msg::{
    HandleMsg as WrappedMsg,
    InitHook,
    InitMsg as WrappedInit,
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
use std::cmp::{
    max,
    min,
};

// Chain ID of Terra
const CHAIN_ID: u16 = 3;

const WRAPPED_ASSET_UPDATING: &str = "updating";

pub fn init<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    _env: Env,
    msg: InitMsg,
) -> StdResult<InitResponse> {
    // Save general wormhole info
    let state = ConfigInfo {
        gov_chain: msg.gov_chain,
        gov_address: msg.gov_address.as_slice().to_vec(),
        wormhole_contract: msg.wormhole_contract,
        wrapped_asset_code_id: msg.wrapped_asset_code_id,
    };
    config(&mut deps.storage).save(&state)?;

    Ok(InitResponse::default())
}

pub fn coins_after_tax<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    coins: Vec<Coin>,
) -> StdResult<Vec<Coin>> {
    let mut res = vec![];
    for coin in coins {
        let asset = Asset {
            amount: coin.amount.clone(),
            info: AssetInfo::NativeToken {
                denom: coin.denom.clone(),
            },
        };
        res.push(asset.deduct_tax(&deps)?);
    }
    Ok(res)
}

pub fn parse_vaa<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    block_time: u64,
    data: &Binary,
) -> StdResult<ParsedVAA> {
    let cfg = config_read(&deps.storage).load()?;
    let vaa: ParsedVAA = deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: cfg.wormhole_contract.clone(),
        msg: to_binary(&WormholeQueryMsg::VerifyVAA {
            vaa: data.clone(),
            block_time,
        })?,
    }))?;
    Ok(vaa)
}

pub fn handle<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    msg: HandleMsg,
) -> StdResult<HandleResponse> {
    match msg {
        HandleMsg::RegisterAssetHook { asset_id } => {
            handle_register_asset(deps, env, &asset_id.as_slice())
        }
        HandleMsg::InitiateTransfer {
            asset,
            amount,
            recipient_chain,
            recipient,
            fee,
            nonce,
        } => handle_initiate_transfer(
            deps,
            env,
            asset,
            amount,
            recipient_chain,
            recipient.as_slice().to_vec(),
            fee,
            nonce,
        ),
        HandleMsg::SubmitVaa { data } => submit_vaa(deps, env, &data),
        HandleMsg::CreateAssetMeta {
            asset_address,
            nonce,
        } => handle_create_asset_meta(deps, env, &asset_address, nonce),
    }
}

/// Handle wrapped asset registration messages
fn handle_register_asset<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    asset_id: &[u8],
) -> StdResult<HandleResponse> {
    let mut bucket = wrapped_asset(&mut deps.storage);
    let result = bucket.load(asset_id);
    let result = result.map_err(|_| ContractError::RegistrationForbidden.std())?;
    if result != HumanAddr::from(WRAPPED_ASSET_UPDATING) {
        return ContractError::AssetAlreadyRegistered.std_err();
    }

    bucket.save(asset_id, &env.message.sender)?;

    let contract_address: CanonicalAddr = deps.api.canonical_address(&env.message.sender)?;
    wrapped_asset_address(&mut deps.storage)
        .save(contract_address.as_slice(), &asset_id.to_vec())?;

    Ok(HandleResponse {
        messages: vec![],
        log: vec![
            log("action", "register_asset"),
            log("asset_id", format!("{:?}", asset_id)),
            log("contract_addr", env.message.sender),
        ],
        data: None,
    })
}

fn handle_attest_meta<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let meta = AssetMeta::deserialize(data)?;

    let expected_contract =
        bridge_contracts_read(&deps.storage).load(&emitter_chain.to_be_bytes())?;

    // must be sent by a registered token bridge contract
    if expected_contract != emitter_address {
        return Err(StdError::unauthorized());
    }

    if CHAIN_ID == meta.token_chain {
        return Err(StdError::generic_err(
            "this asset is native to this chain and should not be attested",
        ));
    }

    let cfg = config_read(&deps.storage).load()?;
    let asset_id = build_asset_id(meta.token_chain, &meta.token_address.as_slice());

    if wrapped_asset_read(&mut deps.storage)
        .load(&asset_id)
        .is_ok()
    {
        return Err(StdError::generic_err(
            "this asset has already been attested",
        ));
    }

    wrapped_asset(&mut deps.storage).save(&asset_id, &HumanAddr::from(WRAPPED_ASSET_UPDATING))?;

    Ok(HandleResponse {
        messages: vec![CosmosMsg::Wasm(WasmMsg::Instantiate {
            code_id: cfg.wrapped_asset_code_id,
            msg: to_binary(&WrappedInit {
                name: get_string_from_32(&meta.name)?,
                symbol: get_string_from_32(&meta.symbol)?,
                asset_chain: meta.token_chain,
                asset_address: meta.token_address.to_vec().into(),
                decimals: min(meta.decimals, 8u8),
                mint: None,
                init_hook: Some(InitHook {
                    contract_addr: env.contract.address,
                    msg: to_binary(&HandleMsg::RegisterAssetHook {
                        asset_id: asset_id.to_vec().into(),
                    })?,
                }),
            })?,
            send: vec![],
            label: None,
        })],
        log: vec![],
        data: None,
    })
}

fn handle_create_asset_meta<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    asset_address: &HumanAddr,
    nonce: u32,
) -> StdResult<HandleResponse> {
    let cfg = config_read(&deps.storage).load()?;

    let request = QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: asset_address.clone(),
        msg: to_binary(&TokenQuery::TokenInfo {})?,
    });

    let asset_canonical = deps.api.canonical_address(asset_address)?;
    let token_info: TokenInfoResponse = deps.querier.query(&request)?;

    let meta: AssetMeta = AssetMeta {
        token_chain: CHAIN_ID,
        token_address: extend_address_to_32(&asset_canonical),
        decimals: token_info.decimals,
        symbol: extend_string_to_32(&token_info.symbol)?,
        name: extend_string_to_32(&token_info.name)?,
    };

    let token_bridge_message = TokenBridgeMessage {
        action: Action::ATTEST_META,
        payload: meta.serialize().to_vec(),
    };

    Ok(HandleResponse {
        messages: vec![CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: cfg.wormhole_contract,
            msg: to_binary(&WormholeHandleMsg::PostMessage {
                message: Binary::from(token_bridge_message.serialize()),
                nonce,
            })?,
            // forward coins sent to this message
            send: coins_after_tax(deps, env.message.sent_funds.clone())?,
        })],
        log: vec![
            log("meta.token_chain", CHAIN_ID),
            log("meta.token", asset_address),
            log("meta.nonce", nonce),
            log("meta.block_time", env.block.time),
        ],
        data: None,
    })
}

fn submit_vaa<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Binary,
) -> StdResult<HandleResponse> {
    let state = config_read(&deps.storage).load()?;

    let vaa = parse_vaa(deps, env.block.time, data)?;
    let data = vaa.payload;

    if vaa_archive_check(&deps.storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }
    vaa_archive_add(&mut deps.storage, vaa.hash.as_slice())?;

    // check if vaa is from governance
    if state.gov_chain == vaa.emitter_chain && state.gov_address == vaa.emitter_address {
        return handle_governance_payload(deps, env, &data);
    }

    let message = TokenBridgeMessage::deserialize(&data)?;

    let result = match message.action {
        Action::TRANSFER => handle_complete_transfer(
            deps,
            env,
            vaa.emitter_chain,
            vaa.emitter_address,
            &message.payload,
        ),
        Action::ATTEST_META => handle_attest_meta(
            deps,
            env,
            vaa.emitter_chain,
            vaa.emitter_address,
            &message.payload,
        ),
        _ => ContractError::InvalidVAAAction.std_err(),
    };
    return result;
}

fn handle_governance_payload<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let gov_packet = GovernancePacket::deserialize(&data)?;
    let module = get_string_from_32(&gov_packet.module)?;

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
        _ => ContractError::InvalidVAAAction.std_err(),
    }
}

fn handle_register_chain<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let RegisterChain {
        chain_id,
        chain_address,
    } = RegisterChain::deserialize(&data)?;

    let existing = bridge_contracts_read(&deps.storage).load(&chain_id.to_be_bytes());
    if existing.is_ok() {
        return Err(StdError::generic_err(
            "bridge contract already exists for this chain",
        ));
    }

    let mut bucket = bridge_contracts(&mut deps.storage);
    bucket.save(&chain_id.to_be_bytes(), &chain_address)?;

    Ok(HandleResponse {
        messages: vec![],
        log: vec![
            log("chain_id", chain_id),
            log("chain_address", hex::encode(chain_address)),
        ],
        data: None,
    })
}

fn handle_complete_transfer<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    data: &Vec<u8>,
) -> StdResult<HandleResponse> {
    let transfer_info = TransferInfo::deserialize(&data)?;

    let expected_contract =
        bridge_contracts_read(&deps.storage).load(&emitter_chain.to_be_bytes())?;

    // must be sent by a registered token bridge contract
    if expected_contract != emitter_address {
        return Err(StdError::unauthorized());
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
        let contract_addr = wrapped_asset_read(&deps.storage).load(&asset_id).ok();

        return if let Some(contract_addr) = contract_addr {
            // Asset already deployed, just mint

            let recipient = deps
                .api
                .human_address(&target_address)
                .or_else(|_| ContractError::WrongTargetAddressFormat.std_err())?;

            let mut messages = vec![CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: contract_addr.clone(),
                msg: to_binary(&WrappedMsg::Mint {
                    recipient: recipient.clone(),
                    amount: Uint128::from(amount),
                })?,
                send: vec![],
            })];
            if fee != 0 {
                messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                    contract_addr: contract_addr.clone(),
                    msg: to_binary(&WrappedMsg::Mint {
                        recipient: env.message.sender.clone(),
                        amount: Uint128::from(fee),
                    })?,
                    send: vec![],
                }))
            }

            Ok(HandleResponse {
                messages,
                log: vec![
                    log("action", "complete_transfer_wrapped"),
                    log("contract", contract_addr),
                    log("recipient", recipient),
                    log("amount", amount),
                ],
                data: None,
            })
        } else {
            Err(StdError::generic_err("Wrapped asset not deployed. To deploy, invoke CreateWrapped with the associated AssetMeta"))
        };
    } else {
        let token_address = transfer_info.token_address.as_slice().get_address(0);

        let recipient = deps.api.human_address(&target_address)?;
        let contract_addr = deps.api.human_address(&token_address)?;

        // note -- here the amount is the amount the recipient will receive;
        // amount + fee is the total sent
        receive_native(&mut deps.storage, &token_address, Uint128(amount + fee))?;

        // undo normalization to 8 decimals
        let token_info: TokenInfoResponse =
            deps.querier.query(&QueryRequest::Wasm(WasmQuery::Smart {
                contract_addr: contract_addr.clone(),
                msg: to_binary(&TokenQuery::TokenInfo {})?,
            }))?;

        let decimals = token_info.decimals;
        let multiplier = 10u128.pow((max(decimals, 8u8) - 8u8) as u32);
        amount = amount.checked_mul(multiplier).unwrap();
        fee = fee.checked_mul(multiplier).unwrap();

        let mut messages = vec![CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: contract_addr.clone(),
            msg: to_binary(&TokenMsg::Transfer {
                recipient: recipient.clone(),
                amount: Uint128::from(amount),
            })?,
            send: vec![],
        })];

        if fee != 0 {
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: contract_addr.clone(),
                msg: to_binary(&TokenMsg::Transfer {
                    recipient: env.message.sender.clone(),
                    amount: Uint128::from(fee),
                })?,
                send: vec![],
            }))
        }

        Ok(HandleResponse {
            messages,
            log: vec![
                log("action", "complete_transfer_native"),
                log("recipient", recipient),
                log("contract", contract_addr),
                log("amount", amount),
            ],
            data: None,
        })
    }
}

fn handle_initiate_transfer<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    asset: HumanAddr,
    mut amount: Uint128,
    recipient_chain: u16,
    recipient: Vec<u8>,
    mut fee: Uint128,
    nonce: u32,
) -> StdResult<HandleResponse> {
    if recipient_chain == CHAIN_ID {
        return ContractError::SameSourceAndTarget.std_err();
    }

    if amount.is_zero() {
        return ContractError::AmountTooLow.std_err();
    }

    if fee > amount {
        return Err(StdError::generic_err("fee greater than sent amount"));
    }

    let asset_chain: u16;
    let asset_address: Vec<u8>;

    let cfg: ConfigInfo = config_read(&deps.storage).load()?;
    let asset_canonical: CanonicalAddr = deps.api.canonical_address(&asset)?;

    let mut messages: Vec<CosmosMsg> = vec![];

    match wrapped_asset_address_read(&deps.storage).load(asset_canonical.as_slice()) {
        Ok(_) => {
            // This is a deployed wrapped asset, burn it
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: asset.clone(),
                msg: to_binary(&WrappedMsg::Burn {
                    account: env.message.sender.clone(),
                    amount,
                })?,
                send: vec![],
            }));
            let request = QueryRequest::<()>::Wasm(WasmQuery::Smart {
                contract_addr: asset,
                msg: to_binary(&WrappedQuery::WrappedAssetInfo {})?,
            });
            let wrapped_token_info: WrappedAssetInfoResponse =
                deps.querier.custom_query(&request)?;
            asset_chain = wrapped_token_info.asset_chain;
            asset_address = wrapped_token_info.asset_address.as_slice().to_vec();
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
            amount = Uint128(
                amount
                    .u128()
                    .checked_sub(amount.u128().checked_rem(multiplier).unwrap())
                    .unwrap(),
            );
            fee = Uint128(
                fee.u128()
                    .checked_sub(fee.u128().checked_rem(multiplier).unwrap())
                    .unwrap(),
            );

            // This is a regular asset, transfer its balance
            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: asset,
                msg: to_binary(&TokenMsg::TransferFrom {
                    owner: env.message.sender.clone(),
                    recipient: env.contract.address.clone(),
                    amount,
                })?,
                send: vec![],
            }));
            asset_address = extend_address_to_32(&asset_canonical);
            asset_chain = CHAIN_ID;

            // convert to normalized amounts before recording & posting vaa
            amount = Uint128(amount.u128().checked_div(multiplier).unwrap());
            fee = Uint128(fee.u128().checked_div(multiplier).unwrap());

            send_native(&mut deps.storage, &asset_canonical, amount)?;
        }
    };

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
        msg: to_binary(&WormholeHandleMsg::PostMessage {
            message: Binary::from(token_bridge_message.serialize()),
            nonce,
        })?,
        // forward coins sent to this message
        send: coins_after_tax(deps, env.message.sent_funds.clone())?,
    }));

    Ok(HandleResponse {
        messages,
        log: vec![
            log("transfer.token_chain", asset_chain),
            log("transfer.token", hex::encode(asset_address)),
            log(
                "transfer.sender",
                hex::encode(extend_address_to_32(
                    &deps.api.canonical_address(&env.message.sender)?,
                )),
            ),
            log("transfer.recipient_chain", recipient_chain),
            log("transfer.recipient", hex::encode(recipient)),
            log("transfer.amount", amount),
            log("transfer.nonce", nonce),
            log("transfer.block_time", env.block.time),
        ],
        data: None,
    })
}

pub fn query<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {
        QueryMsg::WrappedRegistry { chain, address } => {
            to_binary(&query_wrapped_registry(deps, chain, address.as_slice())?)
        }
    }
}

pub fn query_wrapped_registry<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    chain: u16,
    address: &[u8],
) -> StdResult<WrappedRegistryResponse> {
    let asset_id = build_asset_id(chain, address);
    // Check if this asset is already deployed
    match wrapped_asset_read(&deps.storage).load(&asset_id) {
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
