mod helpers;

use cosmwasm_std::{Addr, Binary};
use cw_multi_test::{Executor};

use cw_token_bridge::{
    msg::{QueryMsg as TokenBridgeQueryMsg, WrappedRegistryResponse},
};

use helpers::{create_submit_vaa_msg, instantiate_contracts, OWNER};

#[test]
fn basic_init() {
    let (_router, _ibc_wormhole_translator_contract_addr, _, _) = instantiate_contracts();
}

#[test]
fn attest_weth_on_token_bridge() {
    // Connect to the token bridge.
    let (mut router, _, _, tb) = instantiate_contracts();

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
    let _query_response: WrappedRegistryResponse = router
        .wrap()
        .query_wasm_smart(
            tb,
            &query_msg,
        )
        .unwrap();
}