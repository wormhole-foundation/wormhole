use std::str::FromStr;

use cosmwasm_std::{
    testing::{mock_dependencies, mock_env, mock_info},
    Binary, IbcChannel, IbcEndpoint, IbcOrder,
};
use wormhole::msg::{ExecuteMsg, InstantiateMsg};

use super::{execute, find_wormchain_channel_id, instantiate, WORMCHAIN_IBC_RECEIVER_PORT};

// instantiate
// 1. success - happy path
// 2. failure - mock wormhole core bridge function to fail

// post_message_ibc
// 1. failure - mock the querier to fail
// 3. failure - mock getting matching channel id to fail
// 4. failure - mock core contract execution to fail

// 5. success - validate IBC packet was sent? How to do this?
#[test]
fn execute_post_message_ibc_happy_path() {
    // instantiate
    let mut deps = mock_dependencies();
    let inst_info = mock_info("creator", &[]);
    let inst_res = instantiate(
        deps.as_mut(),
        mock_env(),
        inst_info,
        InstantiateMsg {
            gov_chain: 1,
            gov_address: Binary::from_base64("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=")
                .unwrap(),
            guardian_set_expirity: 86400,
            initial_guardian_set: wormhole::state::GuardianSetInfo {
                addresses: Vec::new(),
                expiration_time: 0,
            },
            chain_id: 18,
            fee_denom: String::from("uluna"),
        },
    );

    let execute_info = mock_info(
        "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
        &[],
    );
    let wh_message = Binary::from_base64("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=").unwrap();
    let res = execute(
        deps.as_mut(),
        mock_env(),
        execute_info,
        ExecuteMsg::PostMessage {
            message: wh_message,
            nonce: 1,
        },
    )
    .unwrap();
    assert_eq!(res.attributes.len(), 5);
}

// find_wormchain_channel_id
// 1. failure - no matching channel found
#[test]
fn find_wormchain_channel_id_failure() {
    let channel = IbcChannel::new(
        IbcEndpoint {
            port_id: String::from("transfer"),
            channel_id: String::from("channel-0"),
        },
        IbcEndpoint {
            port_id: String::from("transfer"),
            channel_id: String::from("channel-0"),
        },
        IbcOrder::Unordered,
        String::from("version"),
        String::from("connection-0"),
    );

    let result = find_wormchain_channel_id(vec![channel]);
    assert_eq!(result.is_err(), true)
}

// 2. success - matching channel found (happy path)
#[test]
fn find_wormchain_channel_id_happy_path() {
    let expected_channel_id = "channel-0";

    let channel = IbcChannel::new(
        IbcEndpoint {
            port_id: String::from("transfer"),
            channel_id: String::from("channel-0"),
        },
        IbcEndpoint {
            port_id: String::from(WORMCHAIN_IBC_RECEIVER_PORT),
            channel_id: String::from(expected_channel_id),
        },
        IbcOrder::Unordered,
        String::from("version"),
        String::from("connection-0"),
    );

    let matching_channel_id = find_wormchain_channel_id(vec![channel]).unwrap();
    assert_eq!(matching_channel_id, expected_channel_id);
}
