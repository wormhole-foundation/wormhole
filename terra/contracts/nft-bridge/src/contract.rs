use crate::{
    msg::WrappedRegistryResponse,
    state::{
        wrapped_asset,
        BoundedVec,
    },
    token_id::{
        from_external_token_id,
        to_external_token_id,
    },
    CHAIN_ID,
};
use cosmwasm_std::{
    entry_point,
    to_binary,
    Binary,
    CanonicalAddr,
    CosmosMsg,
    Deps,
    DepsMut,
    Empty,
    Env,
    MessageInfo,
    Order,
    QueryRequest,
    Response,
    StdError,
    StdResult,
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
        config,
        config_read,
        wrapped_asset_address,
        wrapped_asset_address_read,
        wrapped_asset_read,
        Action,
        ConfigInfo,
        RegisterChain,
        TokenBridgeMessage,
        TransferInfo,
        UpgradeContract,
    },
};
use wormhole::{
    byte_utils::{
        extend_address_to_32,
        extend_address_to_32_array,
        get_string_from_32,
        string_to_array,
        ByteUtils,
    },
    error::ContractError,
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

use sha3::{
    Digest,
    Keccak256,
};

type HumanAddr = String;

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
        .add_attribute("migrate", "upgrade cw721 wrappers")
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
        ExecuteMsg::InitiateTransfer {
            contract_addr,
            token_id,
            recipient_chain,
            recipient,
            nonce,
        } => handle_initiate_transfer(
            deps,
            env,
            info,
            contract_addr,
            token_id,
            recipient_chain,
            recipient.to_array()?,
            nonce,
        ),
        // TODO(csongor): make this nicer (pull out parsing and look up guards)
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),
        ExecuteMsg::RegisterAssetHook { asset_id } => {
            handle_register_asset(deps, env, info, &asset_id.as_slice())
        }
    }
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
            TransferInfo::deserialize(&message.payload)?,
        ),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
}

fn handle_governance_payload(deps: DepsMut, env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let gov_packet = GovernancePacket::deserialize(&data)?;
    let module = get_string_from_32(&gov_packet.module);

    if module != "NFTBridge" {
        return Err(StdError::generic_err("this is not a valid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != CHAIN_ID {
        return Err(StdError::generic_err(
            "the governance VAA is for another chain",
        ));
    }

    match gov_packet.action {
        1u8 => handle_register_chain(deps, env, RegisterChain::deserialize(&gov_packet.payload)?),
        2u8 => handle_upgrade_contract(
            deps,
            env,
            UpgradeContract::deserialize(&gov_packet.payload)?,
        ),
        _ => ContractError::InvalidVAAAction.std_err(),
    }
}

fn handle_upgrade_contract(
    _deps: DepsMut,
    env: Env,
    upgrade_contract: UpgradeContract,
) -> StdResult<Response> {
    Ok(Response::new()
        .add_message(CosmosMsg::Wasm(WasmMsg::Migrate {
            contract_addr: env.contract.address.to_string(),
            new_code_id: upgrade_contract.new_contract,
            msg: to_binary(&MigrateMsg {})?,
        }))
        .add_attribute("action", "contract_upgrade"))
}

fn handle_register_chain(
    deps: DepsMut,
    _env: Env,
    register_chain: RegisterChain,
) -> StdResult<Response> {
    let RegisterChain {
        chain_id,
        chain_address,
    } = register_chain;

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
    _info: MessageInfo,
    emitter_chain: u16,
    emitter_address: Vec<u8>,
    transfer_info: TransferInfo,
) -> StdResult<Response> {
    let cfg = config_read(deps.storage).load()?;

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

    let token_chain = transfer_info.nft_chain;
    let target_address = &(&transfer_info.recipient[..]).get_address(0);

    let mut messages = vec![];

    let recipient = deps
        .api
        .addr_humanize(&target_address)
        .or_else(|_| ContractError::WrongTargetAddressFormat.std_err())?;

    let contract_addr;

    let token_id = from_external_token_id(
        deps.storage,
        token_chain,
        &transfer_info.nft_address,
        &transfer_info.token_id,
    )?;

    if token_chain != CHAIN_ID {
        // NFT is not native to this chain, so we need a wrapper
        let asset_address = transfer_info.nft_address;
        let asset_id = build_asset_id(token_chain, &asset_address);

        let owner = deps
            .api
            .addr_humanize(&(&transfer_info.recipient[..]).get_address(0))?
            .to_string();

        let token_uri = String::from_utf8(transfer_info.uri.to_vec())
            .or_else(|_| Err(StdError::generic_err("could not parse uri string")))?;

        let mint_msg = cw721_base::msg::MintMsg {
            token_id,
            owner,
            token_uri: Some(token_uri),
            extension: None,
        };

        // Check if this asset is already deployed
        if let Some(wrapped_addr) = wrapped_asset_read(deps.storage).load(&asset_id).ok() {
            contract_addr = wrapped_addr;
            // Asset already deployed, just mint

            messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: contract_addr.clone(),
                msg: to_binary(&cw721_base::msg::ExecuteMsg::Mint(mint_msg))?,
                funds: vec![],
            }));
        } else {
            contract_addr = env.contract.address.clone().into_string();
            wrapped_asset(deps.storage)
                .save(&asset_id, &HumanAddr::from(WRAPPED_ASSET_UPDATING))?;
            messages.push(CosmosMsg::Wasm(WasmMsg::Instantiate {
                admin: Some(contract_addr.clone()),
                code_id: cfg.wrapped_asset_code_id,
                msg: to_binary(&cw721_wrapped::msg::InstantiateMsg {
                    name: get_string_from_32(&transfer_info.name.to_vec()),
                    symbol: get_string_from_32(&transfer_info.symbol.to_vec()),
                    asset_chain: token_chain,
                    asset_address: (&transfer_info.nft_address[..]).into(),
                    minter: env.contract.address.clone().into_string(),
                    mint: Some(mint_msg),
                    init_hook: Some(cw721_wrapped::msg::InitHook {
                        msg: cw721_wrapped::to_binary(&ExecuteMsg::RegisterAssetHook {
                            asset_id: asset_id.to_vec().into(),
                        })
                        .map_err(|_| StdError::generic_err("couldn't convert to binary"))?,
                        contract_addr: contract_addr.clone(),
                    }),
                })?,
                funds: vec![],
                label: String::new(),
            }));
        }
    } else {
        // Native NFT, transfer from custody
        let mut messages = vec![];
        let token_address = (&transfer_info.nft_address[..]).get_address(0);

        contract_addr = deps.api.addr_humanize(&token_address)?.to_string();

        messages.push(CosmosMsg::<Empty>::Wasm(WasmMsg::Execute {
            contract_addr: token_address.to_string(),
            msg: to_binary(&cw721_base::msg::ExecuteMsg::<Option<Empty>>::TransferNft {
                recipient: env.contract.address.to_string(),
                token_id,
            })?,
            funds: vec![],
        }));
    }
    Ok(Response::new()
        .add_messages(messages)
        .add_attribute("action", "complete_transfer")
        .add_attribute("recipient", recipient)
        .add_attribute("contract", contract_addr))
}

fn handle_initiate_transfer(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    asset: HumanAddr,
    token_id: String,
    recipient_chain: u16,
    recipient: [u8; 32],
    nonce: u32,
) -> StdResult<Response> {
    if recipient_chain == CHAIN_ID {
        return ContractError::SameSourceAndTarget.std_err();
    }

    let asset_chain: u16;
    let asset_address: [u8; 32];

    let cfg: ConfigInfo = config_read(deps.storage).load()?;
    let asset_canonical: CanonicalAddr = deps.api.addr_canonicalize(&asset)?;

    let mut messages: Vec<CosmosMsg> = vec![];

    if let Ok(_) = wrapped_asset_address_read(deps.storage).load(asset_canonical.as_slice()) {
        // This is a deployed wrapped asset, burn it
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: asset.clone(),
            msg: to_binary(&cw721_wrapped::msg::ExecuteMsg::Burn::<Option<Empty>> {
                token_id: token_id.clone(),
            })?,
            funds: vec![],
        }));

        let wrapped_token_info: cw721_wrapped::msg::WrappedAssetInfoResponse = deps
            .querier
            .custom_query(&QueryRequest::<Empty>::Wasm(WasmQuery::Smart {
                contract_addr: asset.clone(),
                msg: to_binary(&cw721_wrapped::msg::QueryMsg::WrappedAssetInfo {})?,
            }))?;

        asset_address = wrapped_token_info.asset_address.to_array()?;
        asset_chain = wrapped_token_info.asset_chain;
    } else {
        // Native NFT, lock it up
        messages.push(CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: asset.clone(),
            msg: to_binary(&cw721_base::msg::ExecuteMsg::<Option<Empty>>::TransferNft {
                recipient: env.contract.address.to_string(),
                token_id: token_id.clone(),
            })?,
            funds: vec![],
        }));

        asset_chain = CHAIN_ID;
        asset_address = extend_address_to_32_array(&asset_canonical);
    };

    let cw721::ContractInfoResponse { symbol, name, .. } =
        deps.querier
            .custom_query(&QueryRequest::<Empty>::Wasm(WasmQuery::Smart {
                contract_addr: asset.clone(),
                msg: to_binary(&cw721_base::msg::QueryMsg::ContractInfo {})?,
            }))?;

    let cw721::NftInfoResponse::<Option<Empty>> { token_uri, .. } =
        deps.querier
            .custom_query(&QueryRequest::<Empty>::Wasm(WasmQuery::Smart {
                contract_addr: asset.clone(),
                msg: to_binary(&cw721_base::msg::QueryMsg::NftInfo {
                    token_id: token_id.clone(),
                })?,
            }))?;

    let transfer_info = TransferInfo {
        nft_address: asset_address,
        nft_chain: asset_chain,
        symbol: string_to_array(&symbol),
        name: string_to_array(&name),
        token_id: to_external_token_id(deps.storage, asset_chain, &asset_address, token_id)?,
        uri: BoundedVec::new(token_uri.unwrap_or("".to_string()).into())?,
        recipient,
        recipient_chain,
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
        funds: vec![],
    }));

    Ok(Response::new()
        .add_messages(messages)
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

/// Handle wrapped asset registration messages
fn handle_register_asset(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    asset_id: &[u8],
) -> StdResult<Response> {
    let mut bucket = wrapped_asset(deps.storage);
    let result = bucket
        .load(asset_id)
        .map_err(|_| ContractError::RegistrationForbidden.std())?;
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

#[cfg(test)]
mod tests {
    use cosmwasm_std::{
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
