use cosmwasm_std::{
    coin,
    testing::{mock_dependencies, mock_env, mock_info},
    to_json_binary, Binary, ContractResult, CosmosMsg, Empty, Event, Reply, ReplyOn, Response,
    SubMsgResponse, SystemError, SystemResult, Uint128, WasmMsg, WasmQuery,
};
use cw_token_bridge::msg::TransferInfoResponse;
use ibc_translator::{
    contract::{execute, instantiate, migrate, query, reply},
    msg::{ChannelResponse, ExecuteMsg, InstantiateMsg, QueryMsg, COMPLETE_TRANSFER_REPLY_ID},
    state::{CHAIN_TO_CHANNEL_MAP, CURRENT_TRANSFER, CW_DENOMS, TOKEN_BRIDGE_CONTRACT},
};
use wormhole_bindings::tokenfactory::{TokenFactoryMsg, TokenMsg};

mod test_setup;
use test_setup::{
    execute_custom_mock_deps, mock_env_custom_contract, WORMHOLE_CONTRACT_ADDR, WORMHOLE_USER_ADDR,
};

// TESTS
// 1. instantiate
//    1. happy path
// 2. migrate
//    1. happy path
// 3. execute
//    1. CompleteTransferAndConvert
//    2. GatewayConvertAndTransfer
//    3. GatewayConvertAndTransferWithPaylod
//    4. SubmitUpdateChainToChannelMap
// 4. reply
//    1. happy path
//    2. no id match
// 5. query
//    1. happy path

// TESTS: instantiate
// 1. happy path
#[test]
fn instantiate_happy_path() {
    let tokenbridge_addr = "faketokenbridge".to_string();

    let mut deps = mock_dependencies();
    let env = mock_env();
    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let msg = InstantiateMsg {
        token_bridge_contract: tokenbridge_addr.clone(),
    };

    let response = instantiate(deps.as_mut(), env, info, msg).unwrap();

    // response should have 2 attributes
    assert_eq!(response.attributes.len(), 2);
    assert_eq!(response.attributes[0].key, "action");
    assert_eq!(response.attributes[0].value, "instantiate");
    assert_eq!(response.attributes[1].key, "owner");
    assert_eq!(response.attributes[1].value, WORMHOLE_USER_ADDR);

    // contract addrs should have been set in storage
    let saved_tb = TOKEN_BRIDGE_CONTRACT.load(deps.as_mut().storage).unwrap();
    assert_eq!(saved_tb, tokenbridge_addr);
}

// TESTS: migrate
// 1. happy path
#[test]
fn migrate_happy_path() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Empty {};

    let expected_response = Response::<Empty>::default();

    let response = migrate(deps.as_mut(), env, msg).unwrap();

    assert_eq!(response, expected_response);
}

// TESTS: execute
// 1. CompleteTransferAndConvert
#[test]
fn execute_complete_transfer_and_convert() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env_custom_contract(WORMHOLE_CONTRACT_ADDR);

    let transfer_info_response = TransferInfoResponse {
        amount: 1000000u32.into(),
        token_address: hex::decode("0000000000000000000000009c3c9283d3e44854697cd22d3faa240cfb032889").unwrap().try_into().unwrap(),
        token_chain: 5,
        recipient: hex::decode("23aae62840414d69ebc26023d1132f59eef316c82222da4644daaa832ea56349").unwrap().try_into().unwrap(),
        recipient_chain: 32,
        fee: 0u32.into(),
        payload: hex::decode("7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d575636637a56745a4731334f486436646d4e7a4f585a344f586b335a4774306357646c4d336c36626a52334d477735626a5130227d7d").unwrap(),
    };
    let transfer_info_response_copy = transfer_info_response.clone();

    deps.querier.update_wasm(move |q| match q {
        WasmQuery::Smart {
            contract_addr: _,
            msg: _,
        } => SystemResult::Ok(ContractResult::Ok(
            to_json_binary(&transfer_info_response_copy).unwrap(),
        )),
        _ => SystemResult::Err(SystemError::UnsupportedRequest {
            kind: "wasm".to_string(),
        }),
    });

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();

    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let vaa = Binary::from_base64("AAAAAA").unwrap();
    let msg = ExecuteMsg::CompleteTransferAndConvert { vaa };

    let response = execute(deps.as_mut(), env, info, msg).unwrap();

    // response should have 1 message
    assert_eq!(response.messages.len(), 1);

    // 1. WasmMsg::Execute (token bridge complete transfer)
    assert_eq!(response.messages[0].id, COMPLETE_TRANSFER_REPLY_ID);
    assert_eq!(response.messages[0].reply_on, ReplyOn::Success);
    assert_eq!(
        response.messages[0].msg,
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: token_bridge_addr,
            msg: Binary::from_base64("eyJjb21wbGV0ZV90cmFuc2Zlcl93aXRoX3BheWxvYWQiOnsiZGF0YSI6IkFBQUFBQT09IiwicmVsYXllciI6Indvcm1ob2xlMXZoa20ycXY3ODRydWx4OHlscnUwenB2eXZ3M20zY3k5OWU2d3kwIn19").unwrap(),
            funds: vec![]
        })
    );

    // response should have 2 attributes
    assert_eq!(response.attributes.len(), 2);
    assert_eq!(response.attributes[0].key, "action");
    assert_eq!(
        response.attributes[0].value,
        "complete_transfer_with_payload"
    );
    assert_eq!(response.attributes[1].key, "transfer_payload");
    assert_eq!(
        response.attributes[1].value,
        Binary::from(transfer_info_response.clone().payload).to_base64()
    );

    // finally, validate that the state was saved into storage
    let saved_transfer = CURRENT_TRANSFER.load(deps.as_mut().storage).unwrap();
    assert_eq!(saved_transfer, transfer_info_response);
}

// 2. GatewayConvertAndTransfer
#[test]
fn execute_gateway_convert_and_transfer() {
    let mut deps = execute_custom_mock_deps();

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();
    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(
            deps.as_mut().storage,
            WORMHOLE_CONTRACT_ADDR.to_string(),
            &tokenfactory_denom,
        )
        .unwrap();
    let coin = coin(1, tokenfactory_denom.clone());

    let info = mock_info(WORMHOLE_USER_ADDR, &[coin.clone()]);
    let env = mock_env();
    let recipient_chain = 2;
    let recipient = Binary::from_base64("AAAAAAAAAAAAAAAAjyagAl3Mxs/Aen04dWKAoQ4pWtc=").unwrap();
    let fee = Uint128::zero();
    let nonce = 0u32;

    let msg = ExecuteMsg::GatewayConvertAndTransfer {
        recipient,
        chain: recipient_chain,
        fee,
        nonce,
    };

    let response = execute(deps.as_mut(), env, info, msg).unwrap();

    // response should have 3 messages
    assert_eq!(response.messages.len(), 3);

    let mut expected_response: Response<TokenFactoryMsg> = Response::new();
    expected_response = expected_response.add_message(TokenMsg::BurnTokens {
        denom: tokenfactory_denom,
        amount: coin.amount.u128(),
        burn_from_address: "".to_string(),
    });
    expected_response = expected_response.add_message(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: WORMHOLE_CONTRACT_ADDR.to_string(),
            msg: Binary::from_base64("eyJpbmNyZWFzZV9hbGxvd2FuY2UiOnsic3BlbmRlciI6ImZha2V0b2tlbmJyaWRnZSIsImFtb3VudCI6IjEiLCJleHBpcmVzIjpudWxsfX0=").unwrap(),
            funds: vec![]
        })
    );
    expected_response = expected_response.add_message(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: token_bridge_addr,
            msg: Binary::from_base64("eyJpbml0aWF0ZV90cmFuc2ZlciI6eyJhc3NldCI6eyJpbmZvIjp7InRva2VuIjp7ImNvbnRyYWN0X2FkZHIiOiJ3b3JtaG9sZTF5dzR3djJ6cWc5eGtuNjd6dnEzYXp5ZTB0OGgweDlrZ3lnM2Q1M2p5bTI0Z3h0NDl2ZHlzNnM4aDdhIn19LCJhbW91bnQiOiIxIn0sInJlY2lwaWVudF9jaGFpbiI6MiwicmVjaXBpZW50IjoiQUFBQUFBQUFBQUFBQUFBQWp5YWdBbDNNeHMvQWVuMDRkV0tBb1E0cFd0Yz0iLCJmZWUiOiIwIiwibm9uY2UiOjB9fQ==").unwrap(),
            funds: vec![]
        })
    );

    // 1. TokenMsg::BurnTokens
    assert_eq!(response.messages[0].msg, expected_response.messages[0].msg,);

    // 2. WasmMsg::Execute (increase allowance)
    assert_eq!(response.messages[1].msg, expected_response.messages[1].msg,);

    // 3. WasmMsg::Execute (initiate transfer)
    assert_eq!(response.messages[2].msg, expected_response.messages[2].msg,);
}

// 3. GatewayConvertAndTransferWithPaylod
#[test]
fn execute_gateway_convert_and_transfer_with_payload() {
    let mut deps = execute_custom_mock_deps();

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();
    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(
            deps.as_mut().storage,
            WORMHOLE_CONTRACT_ADDR.to_string(),
            &tokenfactory_denom,
        )
        .unwrap();
    let coin = coin(1, tokenfactory_denom.clone());

    let info = mock_info(WORMHOLE_USER_ADDR, &[coin.clone()]);
    let env = mock_env();
    let recipient_chain = 2;
    let recipient = Binary::from_base64("AAAAAAAAAAAAAAAAjyagAl3Mxs/Aen04dWKAoQ4pWtc=").unwrap();
    let nonce = 0u32;

    let msg = ExecuteMsg::GatewayConvertAndTransferWithPayload {
        contract: recipient,
        chain: recipient_chain,
        payload: Binary::default(),
        nonce,
    };

    let response = execute(deps.as_mut(), env, info, msg).unwrap();

    // response should have 3 messages
    assert_eq!(response.messages.len(), 3);

    let mut expected_response: Response<TokenFactoryMsg> = Response::new();
    expected_response = expected_response.add_message(TokenMsg::BurnTokens {
        denom: tokenfactory_denom,
        amount: coin.amount.u128(),
        burn_from_address: "".to_string(),
    });
    expected_response = expected_response.add_message(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: WORMHOLE_CONTRACT_ADDR.to_string(),
            msg: Binary::from_base64("eyJpbmNyZWFzZV9hbGxvd2FuY2UiOnsic3BlbmRlciI6ImZha2V0b2tlbmJyaWRnZSIsImFtb3VudCI6IjEiLCJleHBpcmVzIjpudWxsfX0=").unwrap(),
            funds: vec![]
        })
    );
    expected_response = expected_response.add_message(
        CosmosMsg::Wasm(WasmMsg::Execute {
            contract_addr: token_bridge_addr,
            msg: Binary::from_base64("eyJpbml0aWF0ZV90cmFuc2Zlcl93aXRoX3BheWxvYWQiOnsiYXNzZXQiOnsiaW5mbyI6eyJ0b2tlbiI6eyJjb250cmFjdF9hZGRyIjoid29ybWhvbGUxeXc0d3YyenFnOXhrbjY3enZxM2F6eWUwdDhoMHg5a2d5ZzNkNTNqeW0yNGd4dDQ5dmR5czZzOGg3YSJ9fSwiYW1vdW50IjoiMSJ9LCJyZWNpcGllbnRfY2hhaW4iOjIsInJlY2lwaWVudCI6IkFBQUFBQUFBQUFBQUFBQUFqeWFnQWwzTXhzL0FlbjA0ZFdLQW9RNHBXdGM9IiwiZmVlIjoiMCIsInBheWxvYWQiOiIiLCJub25jZSI6MH19").unwrap(),
            funds: vec![]
        })
    );

    // 1. TokenMsg::BurnTokens
    assert_eq!(response.messages[0].msg, expected_response.messages[0].msg,);

    // 2. WasmMsg::Execute (increase allowance)
    assert_eq!(response.messages[1].msg, expected_response.messages[1].msg,);

    // 3. WasmMsg::Execute (initiate transfer)
    assert_eq!(response.messages[2].msg, expected_response.messages[2].msg,);
}

// 4. SubmitUpdateChainToChannelMap
#[test]
fn execute_submit_update_chain_to_channel_map() {
    let mut deps = execute_custom_mock_deps();
    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let env = mock_env();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAs=").unwrap();

    let msg = ExecuteMsg::SubmitUpdateChainToChannelMap { vaa };
    let response = execute(deps.as_mut(), env, info, msg).unwrap();

    // response should have 0 message
    assert_eq!(response.messages.len(), 0);
    assert_eq!(
        response,
        Response::new().add_event(
            Event::new("UpdateChainToChannelMap")
                .add_attribute("chain_id", "Karura".to_string())
                .add_attribute("channel_id", "channel-1".to_string()),
        )
    );
}

// TESTS: reply
// 1. Happy path: REPLY ID matches
#[test]
fn reply_happy_path() {
    let mut deps = mock_dependencies();
    let env = mock_env();

    // for this test we don't build a proper reply.
    // we're just testing that the handle_complete_transfer_reply method is called when the reply_id is 1
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Err("random error".to_string()),
    };

    let err = reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "msg result is not okay, we should never get here"
    );
}

// 2. ID does not match reply -- no op
#[test]
fn reply_no_id_match() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 0,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: None,
        }),
    };

    let err = reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(err.to_string(), "unmatched reply id 0");
}

// TEST: query
// 1. happy path
#[test]
fn query_query_ibc_channel_happy_path() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let chain_id: u16 = 0;
    let msg = QueryMsg::IbcChannel { chain_id };

    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, 0, &channel)
        .unwrap();

    let expected_response = to_json_binary(&ChannelResponse { channel }).unwrap();

    let response = query(deps.as_ref(), env, msg).unwrap();
    assert_eq!(expected_response, response);
}
