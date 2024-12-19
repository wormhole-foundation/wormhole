use cosmwasm_std::{
    coin,
    testing::{mock_env, mock_info, MOCK_CONTRACT_ADDR},
    to_json_binary, Binary, Coin, ContractResult, CosmosMsg, Event, ReplyOn, Response, SystemError,
    SystemResult, Uint128, WasmMsg, WasmQuery,
};
use cw_token_bridge::msg::TransferInfoResponse;
use ibc_translator::{
    execute::{
        complete_transfer_and_convert, contract_addr_from_base58, convert_and_transfer,
        parse_bank_token_factory_contract, submit_update_chain_to_channel_map, TransferType,
    },
    msg::COMPLETE_TRANSFER_REPLY_ID,
    state::{CURRENT_TRANSFER, CW_DENOMS, TOKEN_BRIDGE_CONTRACT},
};
use wormhole_bindings::tokenfactory::{TokenFactoryMsg, TokenMsg};

mod test_setup;
use test_setup::{
    execute_custom_mock_deps, mock_env_custom_contract, WORMHOLE_CONTRACT_ADDR, WORMHOLE_USER_ADDR,
};

// Tests
// 1. complete_transfer_and_convert
//    1. happy path
//    2. no token bridge state
//    3. failure transferinfo query
//    4. failure humanize recipient
//    5. no match recipient contract
// 2. convert_and_transfer
//    1. happy path
//    2. no token bridge state
//    3. no funds
//    4. too many funds
//    5. parse method failure
// 3. parse_bank_token_factory_contract
//    1. happy path
//    2. failure denom length
//    3. failure non factory token
//    4. failure non contract created
//    5. failure base58 decode failure
//    6. failure no storage
//    7. failure storage mismatch
// 4. contract_addr_from_base58
//    1. happy path
//    2. failure decode base58
// 5. submit_update_chain_to_channel_map
//    1. happy path
//    2. failed to parse vaa
//    3. unsupported VAA version
//    4. not a governance vaa
//    5. failed to parse governance packet
//    6. governance vaa is for another chain
//    7. governance vaa already executed
//    8. chain is for wormchain
//    9. failed to parse channel-id

// TESTS: complete_transfer_and_convert
// 1. Happy path
#[test]
fn complete_transfer_and_convert_happy_path() {
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

    let response = complete_transfer_and_convert(deps.as_mut(), env, info, vaa).unwrap();

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

// 2. Failure: no token bridge address in state
#[test]
fn complete_transfer_and_convert_no_token_bridge_state() {
    let mut deps = execute_custom_mock_deps();
    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let env = mock_env();
    let vaa = Binary::from_base64("AAAAAA").unwrap();

    let err = complete_transfer_and_convert(deps.as_mut(), env, info, vaa).unwrap_err();
    assert_eq!(
        err.to_string(),
        "could not load token bridge contract address"
    );
}

// 3. Failure: token bridge query TransferInfo failed
#[test]
fn complete_transfer_and_convert_failure_transferinfo_query() {
    let mut deps = execute_custom_mock_deps();
    deps.querier.update_wasm(|q| match q {
        WasmQuery::Smart {
            contract_addr: _,
            msg: _,
        } => SystemResult::Ok(ContractResult::Err("query failed".to_string())),
        _ => SystemResult::Err(SystemError::UnsupportedRequest {
            kind: "wasm".to_string(),
        }),
    });

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();

    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let env = mock_env();
    let vaa = Binary::from_base64("AAAAAA").unwrap();

    let err = complete_transfer_and_convert(deps.as_mut(), env, info, vaa).unwrap_err();
    assert_eq!(err.to_string(), "could not parse token bridge payload3 vaa");
}

// 4. Failure: could not humanize recipient address
#[test]
fn complete_transfer_and_convert_failure_humanize_recipient() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();

    let transfer_info_response = to_json_binary(&cw_token_bridge::msg::TransferInfoResponse {
        amount: 1000000u32.into(),
        token_address: hex::decode("0000000000000000000000009c3c9283d3e44854697cd22d3faa240cfb032889").unwrap().try_into().unwrap(),
        token_chain: 5,
        recipient: hex::decode("6d9ae6b2d333c1d65301a59da3eed388ca5dc60cb12496584b75cbe6b15fdbed").unwrap().try_into().unwrap(),
        recipient_chain: 32,
        fee: 0u32.into(),
        payload: hex::decode("7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d575636637a56745a4731334f486436646d4e7a4f585a344f586b335a4774306357646c4d336c36626a52334d477735626a5130227d7d").unwrap(),
    }).unwrap();

    deps.querier.update_wasm(move |q| match q {
        WasmQuery::Smart {
            contract_addr: _,
            msg: _,
        } => SystemResult::Ok(ContractResult::Ok(transfer_info_response.clone())),
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

    let err = complete_transfer_and_convert(deps.as_mut(), env, info, vaa).unwrap_err();
    assert_eq!(err.to_string(), "Generic error: case not found");
}

// 5. Failure: recipient address doesn't match contract address
#[test]
fn complete_transfer_and_convert_nomatch_recipient_contract() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();

    let transfer_info_response = to_json_binary(&cw_token_bridge::msg::TransferInfoResponse {
        amount: 1000000u32.into(),
        token_address: hex::decode("0000000000000000000000009c3c9283d3e44854697cd22d3faa240cfb032889").unwrap().try_into().unwrap(),
        token_chain: 5,
        recipient: hex::decode("23aae62840414d69ebc26023d1132f59eef316c82222da4644daaa832ea56349").unwrap().try_into().unwrap(),
        recipient_chain: 32,
        fee: 0u32.into(),
        payload: hex::decode("7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d575636637a56745a4731334f486436646d4e7a4f585a344f586b335a4774306357646c4d336c36626a52334d477735626a5130227d7d").unwrap(),
    }).unwrap();

    deps.querier.update_wasm(move |q| match q {
        WasmQuery::Smart {
            contract_addr: _,
            msg: _,
        } => SystemResult::Ok(ContractResult::Ok(transfer_info_response.clone())),
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

    let err = complete_transfer_and_convert(deps.as_mut(), env, info, vaa).unwrap_err();
    assert_eq!(err.to_string(), "vaa recipient must be this contract");
}

// TESTS: convert_and_transfer
// 1. Happy path
#[test]
fn convert_and_transfer_happy_path() {
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
    let transfer_type = TransferType::Simple { fee };
    let nonce = 0u32;

    let response = convert_and_transfer(
        deps.as_mut(),
        info,
        env,
        recipient,
        recipient_chain,
        transfer_type,
        nonce,
    )
    .unwrap();

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

// 2. Failure: no token bridge address in state
#[test]
fn convert_and_transfer_no_token_bridge_state() {
    let mut deps = execute_custom_mock_deps();
    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let env = mock_env();
    let recipient_chain = 2;
    let recipient = Binary::from_base64("AAAAAAAAAAAAAAAAjyagAl3Mxs/Aen04dWKAoQ4pWtc=").unwrap();
    let fee = Uint128::zero();
    let transfer_type = TransferType::Simple { fee };
    let nonce = 0u32;

    let err = convert_and_transfer(
        deps.as_mut(),
        info,
        env,
        recipient,
        recipient_chain,
        transfer_type,
        nonce,
    )
    .unwrap_err();
    assert_eq!(
        err.to_string(),
        "could not load token bridge contract address"
    );
}

// 3. Failure: no coin in funds
#[test]
fn convert_and_transfer_no_funds() {
    let mut deps = execute_custom_mock_deps();

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();

    let info = mock_info(WORMHOLE_USER_ADDR, &[]);
    let env = mock_env();
    let recipient_chain = 2;
    let recipient = Binary::from_base64("AAAAAAAAAAAAAAAAjyagAl3Mxs/Aen04dWKAoQ4pWtc=").unwrap();
    let fee = Uint128::zero();
    let transfer_type = TransferType::Simple { fee };
    let nonce = 0u32;

    let err = convert_and_transfer(
        deps.as_mut(),
        info,
        env,
        recipient,
        recipient_chain,
        transfer_type,
        nonce,
    )
    .unwrap_err();
    assert_eq!(err.to_string(), "info.funds should contain only 1 coin");
}

// 4. Failure: more coins than expected in funds
#[test]
fn convert_and_transfer_too_many_funds() {
    let mut deps = execute_custom_mock_deps();

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();

    let info = mock_info(WORMHOLE_USER_ADDR, &[coin(1, "denomA"), coin(1, "denomB")]);
    let env = mock_env();
    let recipient_chain = 2;
    let recipient = Binary::from_base64("AAAAAAAAAAAAAAAAjyagAl3Mxs/Aen04dWKAoQ4pWtc=").unwrap();
    let fee = Uint128::zero();
    let transfer_type = TransferType::Simple { fee };
    let nonce = 0u32;

    let err = convert_and_transfer(
        deps.as_mut(),
        info,
        env,
        recipient,
        recipient_chain,
        transfer_type,
        nonce,
    )
    .unwrap_err();
    assert_eq!(err.to_string(), "info.funds should contain only 1 coin");
}

// 5. Failure: parse_bank_token_factory_contract method failure
#[test]
fn convert_and_transfer_parse_method_failure() {
    let mut deps = execute_custom_mock_deps();

    let token_bridge_addr = "faketokenbridge".to_string();
    TOKEN_BRIDGE_CONTRACT
        .save(deps.as_mut().storage, &token_bridge_addr)
        .unwrap();

    let info = mock_info(WORMHOLE_USER_ADDR, &[coin(1, "denomA")]);
    let env = mock_env();
    let recipient_chain = 2;
    let recipient = Binary::from_base64("AAAAAAAAAAAAAAAAjyagAl3Mxs/Aen04dWKAoQ4pWtc=").unwrap();
    let fee = Uint128::zero();
    let transfer_type = TransferType::Simple { fee };
    let nonce = 0u32;

    let err = convert_and_transfer(
        deps.as_mut(),
        info,
        env,
        recipient,
        recipient_chain,
        transfer_type,
        nonce,
    )
    .unwrap_err();
    assert_eq!(err.to_string(), "coin is not from the token factory");
}

// TESTS: parse_bank_token_factory_contract
// 1. Happy path
#[test]
fn parse_bank_token_factory_contract_happy_path() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();

    let tokenfactory_denom = format!(
        "factory/{}/{}",
        MOCK_CONTRACT_ADDR, "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa"
    );
    let coin = Coin::new(100, tokenfactory_denom.clone());
    CW_DENOMS
        .save(
            deps.as_mut().storage,
            WORMHOLE_CONTRACT_ADDR.to_string(),
            &tokenfactory_denom,
        )
        .unwrap();

    let contract_addr = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap();
    assert_eq!(contract_addr, WORMHOLE_CONTRACT_ADDR);
}

// 2. Failure: parsed denom not of length 3
#[test]
fn parse_bank_token_factory_contract_failure_denom_length() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();
    let coin = Coin::new(100, "tokenfactory/denom");

    let method_err = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap_err();
    assert_eq!(method_err.to_string(), "coin is not from the token factory");
}

// 3. Failure: parsed denom[0] != "factory"
#[test]
fn parse_bank_token_factory_contract_failure_non_factory_token() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();
    let coin = Coin::new(100, "tokenfactory/contract/denom");

    let method_err = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap_err();
    assert_eq!(method_err.to_string(), "coin is not from the token factory");
}

// 4. Failure: parsed denom[1] != contract address
#[test]
fn parse_bank_token_factory_contract_failure_non_contract_created() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();
    let coin = Coin::new(100, "factory/contract/denom");

    let method_err = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap_err();
    assert_eq!(method_err.to_string(), "coin is not from the token factory");
}

// 5. Failure: contract_addr_from_base58 method failure
#[test]
fn parse_bank_token_factory_contract_failure_base58_decode_failure() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();
    let coin = Coin::new(100, format!("factory/{MOCK_CONTRACT_ADDR}/denom0"));

    let method_err = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap_err();
    assert_eq!(
        method_err.to_string(),
        "failed to decode base58 subdenom denom0"
    );
}

// 6. Failure: the parsed contract address is not in CW_DENOMS storage
#[test]
fn parse_bank_token_factory_contract_failure_no_storage() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();
    let coin = Coin::new(
        100,
        format!(
            "factory/{}/{}",
            MOCK_CONTRACT_ADDR, "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa"
        ),
    );

    let method_err = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap_err();
    assert_eq!(
        method_err.to_string(),
        "a corresponding denom for the extracted contract addr is not contained in storage"
    );
}

// 7. Failure: the stored denom doesn't equal the coin's denom
#[test]
fn parse_bank_token_factory_contract_failure_storage_mismatch() {
    let mut deps = execute_custom_mock_deps();
    let env = mock_env();
    let coin = Coin::new(
        100,
        format!(
            "factory/{}/{}",
            MOCK_CONTRACT_ADDR, "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa"
        ),
    );

    CW_DENOMS
        .save(
            deps.as_mut().storage,
            WORMHOLE_CONTRACT_ADDR.to_string(),
            &"factory/fake/fake".to_string(),
        )
        .unwrap();

    let method_err = parse_bank_token_factory_contract(deps.as_mut(), env, coin).unwrap_err();
    assert_eq!(
        method_err.to_string(),
        "the stored denom for the contract does not match the actual coin denom"
    );
}

// TESTS: contract_addr_from_base58
// 1. Happy path: convert to contract address
#[test]
fn contract_addr_from_base58_happy_path() {
    let deps = execute_custom_mock_deps();
    let contract_addr = contract_addr_from_base58(
        deps.as_ref(),
        "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa",
    )
    .unwrap();
    assert_eq!(
        contract_addr,
        "wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7a"
    );
}

// 2. Failure: could not decode base58
#[test]
fn contract_addr_from_base58_failure_decode_base58() {
    let deps = execute_custom_mock_deps();
    let method_err = contract_addr_from_base58(
        deps.as_ref(),
        "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETD0",
    )
    .unwrap_err();
    assert_eq!(
        method_err.to_string(),
        "failed to decode base58 subdenom 3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETD0"
    )
}

// 1. happy path
#[test]
fn submit_update_chain_to_channel_map_happy_path() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAs=").unwrap();

    let response = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap();

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

// 2. failed to parse vaa
#[test]
fn submit_update_chain_to_channel_map_failure_parse_vaa() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AAAABQCPOJcHa/xtVL9+sHyaS7ZU7HxXsQXEo1hjh17e9YeNyCTZQ0YfhTF+HiKTHcgZWPNixGTxmQ0/1XrUrqRYEeM2AQEy0QKlDBn6ga2gQDHSNdm0ZN2ldmrsJbPo7gTrlHvpFzoWK1CEIpba+CRgVUPlVtzLe8pDOZzlKv/sVkQXcxLjAAJuOPAeDOu8byFDsOAWmFM+aucKJ9IVy0fMNI6HDt/W13cZLEfmSk3r3xB4JZh9LYtJRXbjBs56rtwCfJ/Vm8rXAQNyQgtCXeTx3rvHALgJ2E7AXXg54NZx8zKlLFPa/9gQrBqLhI66oGInMeUrcLgao8iE+3nXtv98Iyn6MTizDHkFAATUdU1kIWPCxCQ4KnrV7M6bCoYfcXiad7Ez0jaXysSmvCxy/HuDHKBxMG2d7gTLKjQ1Va4tb0h8RZKzmiyGCMKrAQAAAAAAAAABAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAOIAAAAAAAAAAAAAAAAAAAAAAAAABJYmNUcmFuc2xhdG9yAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAGNoYW5uZWwtMQAL").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "failed to parse VAA header")
}

// 3. unsupported VAA version
#[test]
fn submit_update_chain_to_channel_map_failure_unsupported_vaa_version() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AAAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAs=").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "unsupported VAA version")
}

// 4. not a governance vaa
#[test]
fn submit_update_chain_to_channel_map_failure_not_gov_vaa() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAs=").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "not a governance VAA")
}

// 5. failed to parse governance packet
#[test]
fn submit_update_chain_to_channel_map_failed_parsing_gov_packet() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAsR").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "failed to parse governance packet")
}

// 6. governance vaa is for another chain
#[test]
fn submit_update_chain_to_channel_map_failure_gov_vaa_for_another_chain() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAs=").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "this governance VAA is for another chain")
}

// 7. governance vaa already executed
#[test]
fn submit_update_chain_to_channel_map_failure_gov_vaa_already_executed() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xAAs=").unwrap();

    submit_update_chain_to_channel_map(deps.as_mut(), vaa.clone()).unwrap();
    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "governance vaa already executed")
}

// 8. chain is for wormchain
#[test]
fn submit_update_chain_to_channel_map_failure_chain_id_is_wormchain() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lbC0xDCA=").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(
        err.to_string(),
        "the ibc-translator contract should not maintain channel mappings to wormchain"
    )
}
// 9. failed to parse channel-id
#[test]
fn submit_update_chain_to_channel_map_invalid_channel_id() {
    let mut deps = execute_custom_mock_deps();
    let vaa = Binary::from_base64("AQAAAAAFAI84lwdr/G1Uv36wfJpLtlTsfFexBcSjWGOHXt71h43IJNlDRh+FMX4eIpMdyBlY82LEZPGZDT/VetSupFgR4zYBATLRAqUMGfqBraBAMdI12bRk3aV2auwls+juBOuUe+kXOhYrUIQiltr4JGBVQ+VW3Mt7ykM5nOUq/+xWRBdzEuMAAm448B4M67xvIUOw4BaYUz5q5won0hXLR8w0jocO39bXdxksR+ZKTevfEHglmH0ti0lFduMGznqu3AJ8n9WbytcBA3JCC0Jd5PHeu8cAuAnYTsBdeDng1nHzMqUsU9r/2BCsGouEjrqgYicx5StwuBqjyIT7ede2/3wjKfoxOLMMeQUABNR1TWQhY8LEJDgqetXszpsKhh9xeJp3sTPSNpfKxKa8LHL8e4McoHEwbZ3uBMsqNDVVri1vSHxFkrOaLIYIwqsBAAAAAAAAAAEAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAA4gAAAAAAAAAAAAAAAAAAAAAAAAAEliY1RyYW5zbGF0b3IBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY2hhbm5lAC3/AAs=").unwrap();

    let err = submit_update_chain_to_channel_map(deps.as_mut(), vaa).unwrap_err();

    assert_eq!(err.to_string(), "failed to parse channel-id as utf-8")
}
