mod helpers;

use cosmwasm_std::{Addr, Binary};
use cw_multi_test::{Executor};

use ibc_gateway::{
    msg::{ExecuteMsg},
};

use helpers::{instantiate_contracts, OWNER, setup_the_token_bridge};

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
