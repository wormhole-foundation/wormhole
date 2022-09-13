
use shutdown_wormhole::{
    byte_utils::{
        get_string_from_32,
    },
    error::ContractError,
    msg::{
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
    to_binary,
    Binary,
    CosmosMsg,
    Deps,
    DepsMut,
    Env,
    MessageInfo,
    QueryRequest,
    Reply,
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
    },
    state::{
        config,
        config_read,
        ConfigInfo,
        UpgradeContract,
    },
};

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
pub fn reply(_deps: DepsMut, _env: Env, _msg: Reply) -> StdResult<Response> {
    Ok(Response::default())
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
        ExecuteMsg::SubmitVaa { data } => submit_vaa(deps, env, info, &data),
    }
}


fn parse_and_archive_vaa(
    deps: DepsMut,
    env: Env,
    data: &Binary,
) -> StdResult<(ParsedVAA, GovernancePacket)> {
    let state = config_read(deps.storage).load()?;

    let vaa = parse_vaa(deps.as_ref(), env.block.time.seconds(), data)?;

    if vaa_archive_check(deps.storage, vaa.hash.as_slice()) {
        return ContractError::VaaAlreadyExecuted.std_err();
    }
    vaa_archive_add(deps.storage, vaa.hash.as_slice())?;

    // check if vaa is from governance
    if is_governance_emitter(&state, vaa.emitter_chain, &vaa.emitter_address) {
        let gov_packet = GovernancePacket::deserialize(&vaa.payload)?;
        return Ok((vaa, gov_packet));
    }

    return Err(StdError::generic_err("Unsupported VAA"));
}

fn submit_vaa(
    mut deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    data: &Binary,
) -> StdResult<Response> {
    let (_vaa, payload) = parse_and_archive_vaa(deps.branch(), env.clone(), data)?;
    handle_governance_payload(deps, env, &payload)
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


fn is_governance_emitter(cfg: &ConfigInfo, emitter_chain: u16, emitter_address: &[u8]) -> bool {
    cfg.gov_chain == emitter_chain && cfg.gov_address == emitter_address
}
