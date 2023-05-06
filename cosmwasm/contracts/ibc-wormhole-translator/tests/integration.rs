use cw_multi_test::{App, Contract, ContractWrapper, Executor};
use cosmwasm_std::{Addr, Binary, Empty};

use cw_wormhole::{
    msg::{InstantiateMsg as WormholeInstantiateMsg},
    state::{GuardianAddress, GuardianSetInfo},
};

use cw_token_bridge::{
    msg::{InstantiateMsg as TokenBridgeInstantiateMsg},
};

use ibc_wormhole_translator::{
    msg::{AllChainChannelsResponse, ExecuteMsg, InstantiateMsg, QueryMsg},
};

static OWNER: &str = "OWNER";
static GOV_ADDR: &str = "0000000000000000000000000000000000000000000000000000000000000004";
static CHANNEL_0: &str = "Y2hhbm5lbC0w";

#[allow(dead_code)]
fn mock_app() -> App {
    App::default()
}

#[allow(dead_code)]
fn wormhole_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new_with_empty(
        cw_wormhole::contract::execute,
        cw_wormhole::contract::instantiate,
        cw_wormhole::contract::query,
    )
    .with_migrate(cw_wormhole::contract::migrate);
    Box::new(contract)
}

#[allow(dead_code)]
fn token_bridge_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new_with_empty(
        cw_token_bridge::contract::execute,
        cw_token_bridge::contract::instantiate,
        cw_token_bridge::contract::query,
    )
    .with_reply(cw_token_bridge::contract::reply);
    Box::new(contract)
}

#[allow(dead_code)]
fn ibc_wormhole_translator_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new_with_empty(
        ibc_wormhole_translator::contract::execute,
        ibc_wormhole_translator::contract::instantiate,
        ibc_wormhole_translator::contract::query,
    )
    .with_reply(ibc_wormhole_translator::contract::reply);
    Box::new(contract)
}

fn instantiate_contracts() -> (App, Addr) {
    let mut router = mock_app();

    let guardians = [GuardianAddress {
        bytes: hex::decode("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")
            .expect("Decoding failed")
            .into(),
    }];

    let gov_addr = hex::decode(GOV_ADDR).unwrap();
    let gov_addr = Binary::from(gov_addr.clone());
    
    let wormhole_codeid = router.store_code(wormhole_contract());
    let wormhole_contract_addr = router
        .instantiate_contract(
            wormhole_codeid,
            Addr::unchecked(OWNER),
            &WormholeInstantiateMsg {
                gov_chain: 1,
                gov_address: gov_addr.clone(),
                initial_guardian_set: GuardianSetInfo {
                    addresses: guardians.to_vec(),
                    expiration_time: 0,
                },
                guardian_set_expirity: 50,
                chain_id: 3104,
                fee_denom: "uworm".to_string(),
            },
            &[],                // funds
            "Wormhole core",    // label
            None,               // code admin (for migration)
        )
        .unwrap();

    let token_bridge_codeid = router.store_code(token_bridge_contract());
    let token_bridge_contract_addr = router
        .instantiate_contract(
            token_bridge_codeid,
            Addr::unchecked(OWNER),
            &TokenBridgeInstantiateMsg {
                gov_chain: 1,
                gov_address: gov_addr.clone(),
                wormhole_contract: wormhole_contract_addr.to_string(),
                wrapped_asset_code_id: 0,
                chain_id: 3104,
                native_denom: "uworm".to_string(),
                native_symbol: "WORM".to_string(),
                native_decimals: 6,
            },
            &[],                        // funds
            "Wormhole token bridge",    // label
            None,                       // code admin (for migration)
        )
        .unwrap();
    
    let ibc_wormhole_translator_codeid = router.store_code(ibc_wormhole_translator_contract());
    let ibc_wormhole_translator_contract_addr = router
        .instantiate_contract(
            ibc_wormhole_translator_codeid,
            Addr::unchecked(OWNER),
            &InstantiateMsg {
                gov_chain: 1,
                gov_address: gov_addr.clone(),
                wormhole_contract: wormhole_contract_addr.to_string(),
                token_bridge_contract: token_bridge_contract_addr.to_string(),
                wrapped_asset_code_id: 0,
                chain_id: 3104,
                native_denom: "uworm".to_string(),
                native_symbol: "WORM".to_string(),
                native_decimals: 6,
            },
            &[],                        // funds
            "Wormhole token bridge",    // label
            None,                       // code admin (for migration)
        )
        .unwrap();

    (router, ibc_wormhole_translator_contract_addr)
}

#[test]
fn basic_init() {
    let (_router, _ibc_wormhole_translator_contract_addr) = instantiate_contracts();
}

#[test]
fn query_chain_channels() {
    let (router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

    let query_msg = QueryMsg::AllChainChannels {};
    let query_response: AllChainChannelsResponse = router
        .wrap()
        .query_wasm_smart(
            ibc_wormhole_translator_contract_addr,
            &query_msg,
        )
        .unwrap();
    assert_eq!(query_response.chain_channels.len(), 1);
    assert_eq!(query_response.chain_channels[0].0.to_string(), CHANNEL_0.to_string());
    assert_eq!(query_response.chain_channels[0].1, 18);
}

// #[test]
// fn submit_invalid_vaa() {
//     let (mut router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

//     let vaa = "0000000000000000000000000000000000000000000000000000000075757364";
//     let vaa = hex::decode(vaa).unwrap();
//     let vaa = Binary::from(vaa.clone());

//     let execute_msg = ExecuteMsg::SubmitVaa {data: vaa};
//     router
//         .execute_contract(
//             Addr::unchecked(OWNER),
//             ibc_wormhole_translator_contract_addr.clone(),    // clone since we'll use it again
//             &execute_msg,
//             &[],                              // funds
//         )
//         .unwrap();
//         // .expect_err("Generic error: Querier contract error: Generic error: InvalidVAA");
// }

#[test]
fn submit_invalid_vaa2() {
    let (mut router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

    let vaa = "01000000000100540a7be3068e9aee755fcc29107577fada8c4b0fd41579a7f810de25742dcd24650c94777b60217fe2504847059f815f22fd8cda815a44622760067e95589efb00000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000070fb3b00000000000000000000000000000000000000000000546f6b656e42726964676501000000160000000000000000000000000000000000000000000000000000000000000001";
    let vaa = hex::decode(vaa).unwrap();
    let vaa = Binary::from(vaa.clone());

    let execute_msg = ExecuteMsg::SubmitVaa {data: vaa};
    router
        .execute_contract(
            Addr::unchecked(OWNER),
            ibc_wormhole_translator_contract_addr.clone(),    // clone since we'll use it again
            &execute_msg,
            &[],                              // funds
        )
        .unwrap();
        // .expect_err("Generic error: Querier contract error: Generic error: InvalidVAA");
}