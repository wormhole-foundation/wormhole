#![allow(dead_code)]

// The framework assigns the following contract addresses:
//   WH:   contract0
//   TB:   contract1
//   IBC:  contract2
//   WETH: contract3 (first attested token)

use cw_multi_test::{App, Contract, ContractWrapper, Executor, AddressGenerator, WasmKeeper, Router};
use cosmwasm_std::{Addr, Binary, Empty, Storage, Api};

use cw_wormhole::{
    msg::{InstantiateMsg as WormholeInstantiateMsg},
    state::{GuardianAddress, GuardianSetInfo},
};

use cw_token_bridge::{
    msg::{InstantiateMsg as TokenBridgeInstantiateMsg, QueryMsg as TokenBridgeQueryMsg, WrappedRegistryResponse},
};

use ibc_gateway::{
    msg::{AllChainChannelsResponse, ExecuteMsg, InstantiateMsg, QueryMsg},
};

static GOV_ADDR: &str = "0000000000000000000000000000000000000000000000000000000000000004";

pub static OWNER: &str = "OWNER";
pub static CHANNEL_18: &str = "Y2hhbm5lbC0xOA==";
pub static CHANNEL_42: &str = "Y2hhbm5lbC00Mg==";

struct TestAddressGenerator {
    addr_to_return: Addr,
}

impl AddressGenerator for TestAddressGenerator {
    fn next_address(&self, _: &mut dyn Storage) -> Addr {
        self.addr_to_return.clone()
    }
}

fn no_init<BankT, CustomT, WasmT, StakingT, DistrT, IbcT, GovT>(
    _: &mut Router<BankT, CustomT, WasmT, StakingT, DistrT, IbcT, GovT>,
    _: &dyn Api,
    _: &mut dyn Storage,
) {
    println!("hello there!");
    let expected_addr = Addr::unchecked("new_test_addr");
    // let mut keeper =
    //     WasmKeeper::<Empty, Empty>::new_with_custom_address_generator(TestAddressGenerator {
    //         addr_to_return: expected_addr.clone(),
    //     });
}

#[allow(dead_code)]
fn mock_app() -> App {
    App::new(no_init)
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
fn cw20_wrapped_2_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new_with_empty(
        cw20_wrapped_2::contract::execute,
        cw20_wrapped_2::contract::instantiate,
        cw20_wrapped_2::contract::query,
    )
    .with_migrate(cw20_wrapped_2::contract::migrate);
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
fn ibc_gateway_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new_with_empty(
        ibc_gateway::contract::execute,
        ibc_gateway::contract::instantiate,
        ibc_gateway::contract::query,
    )
    .with_reply(ibc_gateway::contract::reply);
    Box::new(contract)
}

pub fn instantiate_contracts() -> (App, Addr, Addr, Addr) {
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
            &[],           // funds
            "Wormhole core",    // label
            None,               // code admin (for migration)
        )
        .unwrap();

    let cw20_wrapped_2_codeid = router.store_code(cw20_wrapped_2_contract());

    let token_bridge_codeid = router.store_code(token_bridge_contract());
    let token_bridge_contract_addr = router
        .instantiate_contract(
            token_bridge_codeid,
            Addr::unchecked(OWNER),
            &TokenBridgeInstantiateMsg {
                gov_chain: 1,
                gov_address: gov_addr.clone(),
                wormhole_contract: wormhole_contract_addr.to_string(),
                wrapped_asset_code_id: cw20_wrapped_2_codeid,
                chain_id: 3104,
                native_denom: "uworm".to_string(),
                native_symbol: "WORM".to_string(),
                native_decimals: 6,
            },
            &[],                   // funds
            "Wormhole token bridge",    // label
            None,                       // code admin (for migration)
        )
        .unwrap();
    
    let ibc_gateway_codeid = router.store_code(ibc_gateway_contract());
    let ibc_gateway_contract_addr = router
        .instantiate_contract(
            ibc_gateway_codeid,
            Addr::unchecked(OWNER),
            &InstantiateMsg {
                gov_chain: 1,
                gov_address: gov_addr.clone(),
                wormhole_contract: wormhole_contract_addr.to_string(),
                token_bridge_contract: token_bridge_contract_addr.to_string(),
                wrapped_asset_code_id: cw20_wrapped_2_codeid,
                chain_id: 3104,
                native_denom: "uworm".to_string(),
                native_symbol: "WORM".to_string(),
                native_decimals: 6,
            },
            &[],                   // funds
            "Wormhole token bridge",    // label
            None,                       // code admin (for migration)
        )
        .unwrap();

    (router, ibc_gateway_contract_addr, wormhole_contract_addr, token_bridge_contract_addr)
}


pub fn create_submit_vaa_msg(s: &str) -> ExecuteMsg {
    let vaa = s;
    let vaa = hex::decode(vaa).unwrap();
    let vaa = Binary::from(vaa.clone());
    ExecuteMsg::SubmitVaa {data: vaa}
}

pub fn create_transfer_vaa_msg(s: &str) -> ExecuteMsg {
    let vaa = s;
    let vaa = hex::decode(vaa).unwrap();
    let vaa = Binary::from(vaa.clone());
    ExecuteMsg::CompleteTransferWithPayload { data: vaa, relayer: "000000000000000123456789".to_string() }
}

pub fn query_chain_channels(router: &App, addr: Addr) -> Vec<(Binary, u16)> {
    let query_msg = QueryMsg::AllChainChannels {};
    let query_response: AllChainChannelsResponse = router
        .wrap()
        .query_wasm_smart(
            addr,
            &query_msg,
        )
        .unwrap();

    assert_eq!(query_response.chain_channels.len(), 1);
    query_response.chain_channels
}

pub fn setup_the_token_bridge(router: &mut App, tb: Addr) {
    // Register Ethereum Token Bridge.
    let reg_eth = create_submit_vaa_msg("01000000000100e2e1975d14734206e7a23d90db48a6b5b6696df72675443293c6057dcb936bf224b5df67d32967adeb220d4fe3cb28be515be5608c74aab6adb31099a478db5c01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e42726964676501000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16");

    router
    .execute_contract(
        Addr::unchecked(OWNER),
        tb.clone(),
        &reg_eth,
        &[],
    )
    .unwrap();

    // Attest WETH.
    let attest_weth = create_submit_vaa_msg("010000000001005d0a19315e0579ca9b3c4290a68555c59dc667db124df5a61a685619daea8dbe2e50d7933f5d8184d7e91ef98c7083ad2c2d26437f60f33d7b6194fd3881fe5900000002acc929010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c1600000000000000000102000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e00021257455448000000000000000000000000000000000000000000000000000000005772617070656420457468657200000000000000000000000000000000000000");

    router
        .execute_contract(
            Addr::unchecked(OWNER),
            tb.clone(),
            &attest_weth,
            &[],
        )
        .unwrap();

    // Verify it exists.
    let token_address = hex::decode("000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e").unwrap();
    let query_msg = TokenBridgeQueryMsg::WrappedRegistry {chain: 2, address: Binary::from(token_address)};
    let query_response: WrappedRegistryResponse = router
        .wrap()
        .query_wasm_smart(
            tb.clone(),
            &query_msg,
        )
        .unwrap();

    // In our test environment, the first attested token is assigned this address.
    assert_eq!("contract3", query_response.address);
}
