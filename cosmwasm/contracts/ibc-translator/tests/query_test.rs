use ibc_translator::{msg::ChannelResponse, query::query_ibc_channel, state::CHAIN_TO_CHANNEL_MAP};

use cosmwasm_std::testing::mock_dependencies;

// Tests
// 1. query_ibc_channel
//    1. happy path
//    2. No chain id to channel mapping

// 1. happy path
#[test]
fn query_ibc_channel_happy_path() {
    let mut deps = mock_dependencies();

    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, 0, &channel)
        .unwrap();

    let expected_response = ChannelResponse { channel };

    let response = query_ibc_channel(deps.as_ref(), 0).unwrap();
    assert_eq!(expected_response, response);
}
// 2. No chain id to channel mapping
#[test]
fn query_ibc_channel_no_chain_id() {
    let deps = mock_dependencies();

    let err = query_ibc_channel(deps.as_ref(), 0).unwrap_err();
    assert_eq!(err.to_string(), "alloc::string::String not found");
}
