mod helpers;

use cosmwasm_std::{Addr, Binary, Uint128};
use cw_multi_test::{Executor};

use cw_token_bridge::{
    msg::{
        IsVaaRedeemedResponse as TokenBridgeIsVaaRedeemedResponse,
        QueryMsg as TokenBridgeQueryMsg,
        WrappedRegistryResponse as TokenBridgeWrappedRegistryResponse,
    },
};

use cw20::{BalanceResponse as TokenBalanceResponse};
use cw20_base::msg::{QueryMsg as TokenQueryMsg};

use ibc_gateway::{
    msg::{ExecuteMsg},
};

use helpers::{create_submit_vaa_msg, instantiate_contracts, OWNER, setup_the_token_bridge};

#[test]
fn basic_init() {
    let (_router, _ibc_gateway_contract_addr, _, _) = instantiate_contracts();
}

#[test]
fn complete_transfer_with_payload_type_1() {
    let (mut router, ibc_gateway, _, tb) = instantiate_contracts();
    setup_the_token_bridge(&mut router, tb.clone());

    // Payload 1 transfer of 123 WETH from Ethereum to BSC:
    let complete_xfer_vaa = hex::decode("010000000001004b07da959fc05de2686b76b0fa744ac2ccc8cd2f24c816ae0a2c634974ea68a62b835342d1c126023b19622f6f648bdb849db8dc3bc567dc5b935d4c84263a2601000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c100040000000000000000000000000000000000000000000000000000000000000000").unwrap();
    let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: "".to_string()};

    let err = router
    .execute_contract(
        Addr::unchecked(OWNER),
        ibc_gateway.clone(),
        &complete_xfer,
        &[],
    )
    .expect_err("complete transfer should have failed");

    assert_eq!(
        "generic error: unexpected payload type",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn complete_transfer_with_wrong_recipient_chain() {
    let (mut router, ibc_gateway, _, tb) = instantiate_contracts();
    setup_the_token_bridge(&mut router, tb.clone());

    // Payload 3 transfer of 123 WETH from Ethereum to BSC:
    let complete_xfer_vaa = hex::decode("010000000001009602268fabb875ca266882051beccbef22f638925d7ad3fdfb723597eae805d65ad43b97e5b7714dfeec7e3eef9da14303ce9abc1c2a362e1167814709809c1201000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010300000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c100040000000000000000000000000000000000000000000000000000000000000000").unwrap();
    let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: "".to_string()};

    let err = router
    .execute_contract(
        Addr::unchecked(OWNER),
        ibc_gateway.clone(),
        &complete_xfer,
        &[],
    )
    .expect_err("complete transfer should have failed");

    assert_eq!(
        "generic error: invalid recipient chain",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn complete_transfer_with_wrong_target_chain() {
    // Connect to the token bridge.
    let (mut router, ibc_gateway, _, tb) = instantiate_contracts();

    // Register and attest WETH on the token bridge.
    setup_the_token_bridge(&mut router, tb.clone());
    
    // Recipient chain: 3104
    // Recipient address: ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b20391 (wormhole1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsqyfker)
    // Transfer payload: '{"basic_transfer":{"chain_id":18,"recipient":"terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"}}' converted to hex
    let complete_xfer_vaa = hex::decode("0100000000010074949ab64f5b0bc7ddfb63bd1db3f64383510662617769254d4d1638ec723cbb36df6b5bae8851837bb5ae762cb93565e6d7e263e71f6671f445c6e23f36512001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010300000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e0002ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b203910c2000000000000000000000000000000000000000000000000000000000000000007b2262617369635f7472616e73666572223a7b22636861696e5f6964223a31382c22726563697069656e74223a2274657272613178343672716179346433637373713867787876717a387874366e776c7a34746432306b333876227d7d").unwrap();
    let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: "".to_string()};

    let err = router
    .execute_contract(
        Addr::unchecked(OWNER),
        ibc_gateway.clone(),
        &complete_xfer,
        &[],
    )
    .expect_err("complete transfer should have failed");

    assert_eq!(
        "generic error: unknown target chain",
        err.root_cause().to_string().to_lowercase()
    );
}


//
// The tests from here on down fail unless you hack ibc-gateway (around line 286) and token bridge (around line 836)
// to set the recipient address to "contract2".
//

#[test]
fn complete_transfer_with_disabled_target_chain() {
    // Connect to the token bridge.
    let (mut router, ibc_gateway, _, tb) = instantiate_contracts();

    // Register and attest WETH on the token bridge.
    setup_the_token_bridge(&mut router, tb.clone());

    // Register chain 18 but set the channel to the null string, meaning it is disabled.
    let execute_msg = create_submit_vaa_msg("010000000001008cc95280459d52fae6a20770cae61fa02269fa7a6d513c0b7390e7e03c5a24060d77f1cca1af29da800ce4eb4f6125f1d5d5afd3bbea8c0bcdff1d9cde38d9d70000000000a5567d7d00010000000000000000000000000000000000000000000000000000000000000004ee8c114665f9261f20000000000000000000000000000000000000004962635472616e736c61746f72010c20001200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000");

    let (mut router, ibc_gateway_contract_addr, _, _) = instantiate_contracts();
    router
        .execute_contract(
            Addr::unchecked(OWNER),
            ibc_gateway_contract_addr.clone(),
            &execute_msg,
            &[],
        )
        .unwrap();
    
    // Recipient chain: 3104
    // Recipient address: ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b20391 (wormhole1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsqyfker)
    // Transfer payload: '{"basic_transfer":{"chain_id":18,"recipient":"terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"}}' converted to hex
    let complete_xfer_vaa = hex::decode("0100000000010074949ab64f5b0bc7ddfb63bd1db3f64383510662617769254d4d1638ec723cbb36df6b5bae8851837bb5ae762cb93565e6d7e263e71f6671f445c6e23f36512001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010300000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e0002ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b203910c2000000000000000000000000000000000000000000000000000000000000000007b2262617369635f7472616e73666572223a7b22636861696e5f6964223a31382c22726563697069656e74223a2274657272613178343672716179346433637373713867787876717a387874366e776c7a34746432306b333876227d7d").unwrap();
    let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: "".to_string()};

    let err = router
    .execute_contract(
        Addr::unchecked(OWNER),
        ibc_gateway.clone(),
        &complete_xfer,
        &[],
    )
    .expect_err("complete transfer should have failed");

    assert_eq!(
        "generic error: transfers to target chain are disabled",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn successful_complete_transfer() {
    // Connect to the token bridge.
    let (mut router, ibc_gateway, _, tb) = instantiate_contracts();

    // Register and attest WETH on the token bridge.
    setup_the_token_bridge(&mut router, tb.clone());

    // Query the token bridge to get the address of WETH.
    let look_up_weth = TokenBridgeQueryMsg::WrappedRegistry{chain: 2, address: Binary::from(hex::decode("000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e").unwrap())};
    let look_up_weth_response: TokenBridgeWrappedRegistryResponse = router
        .wrap()
        .query_wasm_smart(
            tb.clone(),
            &look_up_weth,
        )
        .unwrap();

    let weth_addr = look_up_weth_response.address;
    assert_eq!("contract3", weth_addr);

    // Before we start, query the balance on WETH for our contract. It should be zero.
    let initial_balance = TokenQueryMsg::Balance{address: ibc_gateway.to_string()};
    let initial_balance_response: TokenBalanceResponse = router
        .wrap()
        .query_wasm_smart(
            weth_addr.clone(),
            &initial_balance,
        )
        .unwrap();

    assert_eq!(Uint128::new(0), initial_balance_response.balance);

    // Register the chain to channel mapping for chain 18.
    let execute_msg = create_submit_vaa_msg("010000000001003954625825b74af01b602e401026731b5eda40b0eec103c6c80a7d33102947ca111e67baaa4dca6e2313acc03292e19c60f9130656a6bc4e9ddffb84c17cc2a30000000000a5567d7d00010000000000000000000000000000000000000000000000000000000000000004ee8c114665f9261f20000000000000000000000000000000000000004962635472616e736c61746f72010c2000120000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d3138");

    router
        .execute_contract(
            Addr::unchecked(OWNER),
            ibc_gateway.clone(),
            &execute_msg,
            &[],
        )
        .unwrap();
    
    // Recipient chain: 3104
    // Recipient address: ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b20391 (wormhole1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsqyfker)
    // Transfer payload: '{"basic_transfer":{"chain_id":18,"recipient":"terra1x46rqay4d3cssq8gxxvqz8xt6nwlz4td20k38v"}}' converted to hex
    let complete_xfer_vaa = hex::decode("0100000000010074949ab64f5b0bc7ddfb63bd1db3f64383510662617769254d4d1638ec723cbb36df6b5bae8851837bb5ae762cb93565e6d7e263e71f6671f445c6e23f36512001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010300000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e0002ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b203910c2000000000000000000000000000000000000000000000000000000000000000007b2262617369635f7472616e73666572223a7b22636861696e5f6964223a31382c22726563697069656e74223a2274657272613178343672716179346433637373713867787876717a387874366e776c7a34746432306b333876227d7d").unwrap();
    let relayer = OWNER.to_string(); // If we set the relayer to the null string, the token bridge throws an error at line 906.
    let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: relayer};

    router
    .execute_contract(
        Addr::unchecked(OWNER),
        ibc_gateway.clone(),
        &complete_xfer,
        &[],
    )
    .unwrap();

    // Query the token bridge to see if the VAA was redeemed.
    let is_redeemed = TokenBridgeQueryMsg::IsVaaRedeemed{vaa: Binary::from(complete_xfer_vaa.clone())};
    let is_redeemed_response: TokenBridgeIsVaaRedeemedResponse = router
        .wrap()
        .query_wasm_smart(
            tb.clone(),
            &is_redeemed,
        )
        .unwrap();

    assert_eq!(true, is_redeemed_response.is_redeemed);

    // Make sure the balance on WETH for our contract was updated properly.
    let final_balance = TokenQueryMsg::Balance{address: ibc_gateway.to_string()};
    let final_balance_response: TokenBalanceResponse = router
        .wrap()
        .query_wasm_smart(
            weth_addr.clone(),
            &final_balance,
        )
        .unwrap();

    assert_eq!(Uint128::new(12300000000), final_balance_response.balance);
}

/*
#[test]
fn successful_complete_transfer() {
    // Connect to the token bridge.
    let (mut router, ibc_gateway, _, tb) = instantiate_contracts();

    // Register and attest WETH on the token bridge.
    setup_the_token_bridge(&mut router, tb.clone());

    // Next we want to send a complete transfer to the contract. For now, this is only a payload1. We will need to make it a payload3.

    // Payload 1 transfer of 123 WETH from Ethereum to BSC:
    //010000000001004b07da959fc05de2686b76b0fa744ac2ccc8cd2f24c816ae0a2c634974ea68a62b835342d1c126023b19622f6f648bdb849db8dc3bc567dc5b935d4c84263a2601000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c100040000000000000000000000000000000000000000000000000000000000000000
    
    // Modified to go to 3104, ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b20391 (wormhole1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsqyfker).
    let complete_xfer_vaa = hex::decode("01000000000100af63bf3303670af91c9603e18d6cbe4f6e00b4a664dce7d87d2954076ddca08733b86bcfcd50bef4085793a2511e43cca932202d5265add6aa0a9d8ff99efeeb00000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010300000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e0002ac756341ee5661a37c010946d8d3316bf129bab061e51efaa78c116828b203910c200000000000000000000000000000000000000000000000000000000000000000").unwrap();
    let complete_xfer = ExecuteMsg::CompleteTransferWithPayload {data: Binary::from(complete_xfer_vaa.clone()), relayer: "".to_string()};
    // let complete_xfer = create_submit_vaa_msg("01000000000100b663658ac3a7164973b80b9f172e28f2cf9c39dede860a8ec61e44a3736b903900de62e755b0093b8d0f546fa144be5968b3b0ee46376b4eaa82f83ca24c6c8001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c10c200000000000000000000000000000000000000000000000000000000000000000");
    // let complete_xfer = create_transfer_vaa_msg("01000000000100b663658ac3a7164973b80b9f172e28f2cf9c39dede860a8ec61e44a3736b903900de62e755b0093b8d0f546fa144be5968b3b0ee46376b4eaa82f83ca24c6c8001000001ed8c10010000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c160000000000000001010100000000000000000000000000000000000000000000000000000002dd231b00000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c10c200000000000000000000000000000000000000000000000000000000000000000");

    // Note that this is currently sending to the token bridge.
    // Also note that payload 1 should probably be submitted by calling submit_vaa rather than complete_transfer_with_payload?
    // Also note that is is failing with "Generic error: Invalid input: canonical address length not correct". Not sure what that's about??
    router
    .execute_contract(
        Addr::unchecked(OWNER),
        ibc_gateway.clone(),
        &complete_xfer,
        &[],
    )
    .unwrap();
}
*/
