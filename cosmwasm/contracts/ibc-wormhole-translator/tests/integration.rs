mod helpers;

use cosmwasm_std::{Addr, Binary};
use cw_multi_test::{Executor};

use cw_token_bridge::{
    msg::{QueryMsg as TokenBridgeQueryMsg, WrappedRegistryResponse},
};


// use ibc_wormhole_translator::{
//     msg::{ExecuteMsg},
// };

use helpers::{create_submit_vaa_msg, instantiate_contracts, OWNER};

use crate::helpers::create_transfer_vaa_msg;

#[test]
fn basic_init() {
    let (_router, _ibc_wormhole_translator_contract_addr, _, _) = instantiate_contracts();
}

#[test]
fn attest_weth_on_token_bridge() {
    // Connect to the token bridge.
    let (mut router, _ibc_wormhole_translator, _, tb) = instantiate_contracts();

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

    // Next we want to send a complete transfer to the contract. For now, this is only a payload1. We will need to make it a payload3.

    // Payload 1 transfer of 123 WETH from Ethereum to BSC:
    //010000000001004b07da959fc05de2686b76b0fa744ac2ccc8cd2f24c816ae0a2c634974ea68a62b835342d1c126023b19622f6f648bdb849db8dc3bc567dc5b935d4c84263a2601000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c100040000000000000000000000000000000000000000000000000000000000000000
    
    // Modified to go to 3104:
    // // To convert this to payload three, change this char to a 3 and add the payload three stuff on the end --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
    // let complete_xfer_vaa = hex::decode("01000000000100b663658ac3a7164973b80b9f172e28f2cf9c39dede860a8ec61e44a3736b903900de62e755b0093b8d0f546fa144be5968b3b0ee46376b4eaa82f83ca24c6c8001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c10c200000000000000000000000000000000000000000000000000000000000000000").unwrap();
    // let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: "".to_string()};
    let complete_xfer = create_submit_vaa_msg("01000000000100b663658ac3a7164973b80b9f172e28f2cf9c39dede860a8ec61e44a3736b903900de62e755b0093b8d0f546fa144be5968b3b0ee46376b4eaa82f83ca24c6c8001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c10c200000000000000000000000000000000000000000000000000000000000000000");
    // let complete_xfer = create_transfer_vaa_msg("01000000000100b663658ac3a7164973b80b9f172e28f2cf9c39dede860a8ec61e44a3736b903900de62e755b0093b8d0f546fa144be5968b3b0ee46376b4eaa82f83ca24c6c8001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c10c200000000000000000000000000000000000000000000000000000000000000000");

    // Note that this is currently sending to the token bridge.
    // Also note that payload 1 should probably be submitted by calling submit_vaa rather than complete_transfer_with_payload?
    // Also note that is is failing with "Generic error: Invalid input: canonical address length not correct". Not sure what that's about??
    router
    .execute_contract(
        Addr::unchecked(OWNER),
        tb.clone(),
        &complete_xfer,
        &[],
    )
    .unwrap();
}