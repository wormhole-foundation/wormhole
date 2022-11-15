mod helpers;

use cosmwasm_schema::cw_serde;
use cosmwasm_std::{
    to_binary, Binary, Deps, DepsMut, Empty, Env, Event, MessageInfo, Response, StdResult,
};
use cw_multi_test::ContractWrapper;
use helpers::*;
use wormchain_accounting::msg::Upgrade;
use wormhole_bindings::WormholeQuery;

pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: Empty,
) -> StdResult<Response> {
    Ok(Response::default())
}

pub fn migrate(_deps: DepsMut<WormholeQuery>, _env: Env, _msg: Empty) -> StdResult<Response> {
    Ok(Response::default().add_event(Event::new("migrate-success")))
}

pub fn execute(_deps: DepsMut, _env: Env, _info: MessageInfo, _msg: Empty) -> StdResult<Response> {
    Ok(Response::default())
}

#[cw_serde]
struct NewContract;

pub fn query(_deps: Deps, _env: Env, _msg: Empty) -> StdResult<Binary> {
    to_binary(&NewContract)
}

#[test]
fn upgrade() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());

    let new_code_id = contract.app_mut().store_code(Box::new(
        ContractWrapper::new_with_empty(execute, instantiate, query).with_migrate_empty(migrate),
    ));

    let mut new_addr = [0u8; 32];
    new_addr[24..].copy_from_slice(&new_code_id.to_be_bytes());

    let upgrade = to_binary(&Upgrade { new_addr }).unwrap();
    let signatures = wh.sign(&upgrade);

    let resp = contract
        .upgrade_contract(upgrade, wh.guardian_set_index(), signatures)
        .unwrap();
    resp.assert_event(&Event::new("wasm-migrate-success"));

    contract
        .app()
        .wrap()
        .query_wasm_smart::<NewContract>(contract.addr(), &Empty {})
        .unwrap();
}
