use cosmwasm_std::{
    entry_point,
    to_binary,
    Binary,
    CosmosMsg,
    Deps,
    DepsMut,
    Env,
    MessageInfo,
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
        config,
        config_read,
        price_info,
        price_info_read,
        sequence,
        sequence_read,
        ConfigInfo,
        UpgradeContract,
    },
    types::PriceAttestation,
};
use wormhole::{
    byte_utils::get_string_from_32,
    error::ContractError,
    msg::QueryMsg as WormholeQueryMsg,
    state::{
        vaa_archive_add,
        vaa_archive_check,
        GovernancePacket,
        ParsedVAA,
    },
};

// Chain ID of Terra
const CHAIN_ID: u16 = 3;

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: MigrateMsg) -> StdResult<Response> {
    Ok(Response::new())
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
        pyth_emitter: msg.pyth_emitter.as_slice().to_vec(),
        pyth_emitter_chain: msg.pyth_emitter_chain,
    };
    config(deps.storage).save(&state)?;
    sequence(deps.storage).save(&0)?;

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
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),
    }
}

fn submit_vaa(
    mut deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let state = config_read(deps.storage).load()?;

    let vaa = parse_vaa(deps.branch(), env.block.time.seconds(), data)?;
    let data = vaa.payload;

    // check if vaa is from governance
    if state.gov_chain == vaa.emitter_chain && state.gov_address == vaa.emitter_address {
        if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
            return ContractError::VaaAlreadyExecuted.std_err();
        }
        vaa_archive_add(deps.storage, vaa.hash.as_slice())?;
        
        return handle_governance_payload(deps, env, &data);
    }

    // IMPORTANT: VAA replay-protection is not implemented in this code-path
    // Sequences are used to prevent replay or price rollbacks

    let message =
        PriceAttestation::deserialize(&data[..]).map_err(|_| ContractError::InvalidVAA.std())?;
    if vaa.emitter_address != state.pyth_emitter || vaa.emitter_chain != state.pyth_emitter_chain {
        return ContractError::InvalidVAA.std_err();
    }

    // Check sequence
    let last_sequence = sequence_read(deps.storage).load()?;
    if vaa.sequence <= last_sequence && last_sequence != 0 {
        return Err(StdError::generic_err(
            "price sequences need to be monotonically increasing",
        ));
    }
    sequence(deps.storage).save(&vaa.sequence)?;

    // Update price
    price_info(deps.storage).save(&message.price_id.to_bytes()[..], &data)?;

    Ok(Response::new()
        .add_attribute("action", "price_update")
        .add_attribute("price_feed", message.price_id.to_string()))
}

fn handle_governance_payload(deps: DepsMut, env: Env, data: &Vec<u8>) -> StdResult<Response> {
    let gov_packet = GovernancePacket::deserialize(&data)?;
    let module = get_string_from_32(&gov_packet.module);

    if module != "PythBridge" {
        return Err(StdError::generic_err("this is not a valid module"));
    }

    if gov_packet.chain != 0 && gov_packet.chain != CHAIN_ID {
        return Err(StdError::generic_err(
            "the governance VAA is for another chain",
        ));
    }

    match gov_packet.action {
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

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::PriceInfo { price_id } => {
            to_binary(&query_price_info(deps, price_id.as_slice())?)
        }
    }
}

pub fn query_price_info(deps: Deps, address: &[u8]) -> StdResult<PriceAttestation> {
    match price_info_read(deps.storage).load(address) {
        Ok(data) => PriceAttestation::deserialize(&data[..]).map_err(|_| {
            StdError::parse_err("PriceAttestation", "failed to decode price attestation")
        }),
        Err(_) => ContractError::AssetNotFound.std_err(),
    }
}
