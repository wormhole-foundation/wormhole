use cosmwasm_std::{
    testing::{mock_dependencies, mock_env},
    to_json_binary, to_json_vec, Binary, ContractResult,
    CosmosMsg::Stargate,
    Reply, Response, SubMsgResponse, SystemError, SystemResult, Uint128, WasmQuery,
};
use cw20::TokenInfoResponse;
use cw_token_bridge::msg::{AssetInfo, CompleteTransferResponse, TransferInfoResponse};
use ibc_translator::{
    reply::{
        contract_addr_to_base58, convert_cw20_to_bank_and_send, handle_complete_transfer_reply,
    },
    state::{CHAIN_TO_CHANNEL_MAP, CURRENT_TRANSFER, CW_DENOMS},
};
use prost::Message;
use wormhole_bindings::tokenfactory::{DenomUnit, Metadata, TokenFactoryMsg, TokenMsg};
use wormhole_sdk::Chain;

mod test_setup;
use test_setup::{default_custom_mock_deps, WORMHOLE_CONTRACT_ADDR, WORMHOLE_USER_ADDR};

#[derive(Clone, PartialEq, Message)]
struct MsgExecuteContractResponse {
    #[prost(bytes, tag = "1")]
    pub data: ::prost::alloc::vec::Vec<u8>,
}

// Tests
// 1. handle_complete_transfer_reply
//    1. happy path, GatewayTransfer
//    2. happy path, GatewayTransferWithPayload
//    2. bad msg result
//    3. invalid response data
//    4. no repsonse data
//    5. invalid reponse data type
//    6. no reponse contract
//    7. no storage
//    8. wrong stored payload
//    9. invalid recipient
//    10. invalid contract
// 2. convert_cw20_to_bank_and_send
//    1. happy path
//    2. happy path create denom
//    3. happy path create denom with empty symbol
//    4. failure invalid contract
//    5. chain id no channel
//    6. bad payload
// 3. contract_addr_to_base58
//    1. happy path
//    2. bad contract address

// TESTS: handle_complete_transfer_reply
// 1. Happy path: GatewayTransfer
#[test]
fn handle_complete_transfer_reply_happy_path() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("Cv0BeyJjb250cmFjdCI6Indvcm1ob2xlMXl3NHd2MnpxZzl4a242N3p2cTNhenllMHQ4aDB4OWtneWczZDUzanltMjRneHQ0OXZkeXM2czhoN2EiLCJkZW5vbSI6bnVsbCwicmVjaXBpZW50Ijoic2VpMWRrZHdkdmtueDBxYXY1Y3A1a3c2OG1rbjNyOTltM3N2a3lqZnZrenR3aDk3ZHYybG0wa3NqNnhyYWsiLCJhbW91bnQiOiIxMDAwIiwicmVsYXllciI6InNlaTF2aGttMnF2Nzg0cnVseDh5bHJ1MHpwdnl2dzNtM2N5OXgzeHlmdiIsImZlZSI6IjAifQ==").unwrap())
        })
    };

    let contract_addr =
        "wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7a".to_string();
    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(deps.as_mut().storage, contract_addr, &tokenfactory_denom)
        .unwrap();

    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, 0, &channel)
        .unwrap();

    let transfer_payload = TransferInfoResponse {
        amount: 0u32.into(),
        token_address: [0; 32],
        token_chain: 0,
        recipient: [0; 32],
        recipient_chain: 0,
        fee: 0u32.into(),
        payload: hex::decode("7B22676174657761795F7472616E73666572223A7B22636861696E223A302C22726563697069656E74223A22633256704D575636637A56745A4731334F486436646D4E7A4F585A344F586B335A4774306357646C4D336C36626A52334D477735626A5130222C22666565223A2230222C226E6F6E6365223A307D7D").unwrap()
    };
    CURRENT_TRANSFER
        .save(deps.as_mut().storage, &transfer_payload)
        .unwrap();

    // just verifying that we called the convert_cw20_to_bank -- unwrap the result without an error
    // other tests verify the correctness of this method
    // wormhole core and token bridge tests verify the correctness of the VAA parameters
    handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap();
}

// 2. Happy path: GatewayTransferWithPayload
#[test]
fn handle_complete_transfer_reply_happy_path_with_payload() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("Cv0BeyJjb250cmFjdCI6Indvcm1ob2xlMXl3NHd2MnpxZzl4a242N3p2cTNhenllMHQ4aDB4OWtneWczZDUzanltMjRneHQ0OXZkeXM2czhoN2EiLCJkZW5vbSI6bnVsbCwicmVjaXBpZW50Ijoic2VpMWRrZHdkdmtueDBxYXY1Y3A1a3c2OG1rbjNyOTltM3N2a3lqZnZrenR3aDk3ZHYybG0wa3NqNnhyYWsiLCJhbW91bnQiOiIxMDAwIiwicmVsYXllciI6InNlaTF2aGttMnF2Nzg0cnVseDh5bHJ1MHpwdnl2dzNtM2N5OXgzeHlmdiIsImZlZSI6IjAifQ==").unwrap())
        })
    };

    let contract_addr =
        "wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7a".to_string();
    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(deps.as_mut().storage, contract_addr, &tokenfactory_denom)
        .unwrap();

    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, 0, &channel)
        .unwrap();

    let transfer_payload = TransferInfoResponse {
        amount: 0u32.into(),
        token_address: [0; 32],
        token_chain: 0,
        recipient: [0; 32],
        recipient_chain: 0,
        fee: 0u32.into(),
        payload: hex::decode("7B22676174657761795F7472616E736665725F776974685F7061796C6F6164223A7B22636861696E223A302C22636F6E7472616374223A22633256704D575636637A56745A4731334F486436646D4E7A4F585A344F586B335A4774306357646C4D336C36626A52334D477735626A5130222C227061796C6F6164223A225647567A64464268655778765957513D222C226E6F6E6365223A307D7D").unwrap()
    };
    CURRENT_TRANSFER
        .save(deps.as_mut().storage, &transfer_payload)
        .unwrap();

    // just verifying that we called the convert_cw20_to_bank -- unwrap the result without an error
    // other tests verify the correctness of this method
    // wormhole core and token bridge tests verify the correctness of the VAA parameters
    handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap();
}

// 3. Failure: msg result is not okay
#[test]
fn handle_complete_transfer_reply_bad_msg_result() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Err("random error".to_string()),
    };

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "msg result is not okay, we should never get here"
    );
}

// 4. Failure: could not parse reply response_data
#[test]
fn handle_complete_transfer_reply_invalid_response_data() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("eyJiYXNpY19yZWNpcGllbnQiOnsicmVjaXBpZW50IjoiYzJWcE1XVjZjelZ0WkcxM09IZDZkbU56T1haNE9YazNaR3QwY1dkbE0zbDZialIzTUd3NWJqUTAifX0=").unwrap())
        })
    };

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "failed to parse protobuf reply response_data"
    );
}

// 5. Failure: no data in the parsed response
#[test]
fn handle_complete_transfer_reply_no_response_data() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("").unwrap()),
        }),
    };

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "no data in the response, we should never get here"
    );
}

// 6. Failure: could not deserialize response data
#[test]
fn handle_complete_transfer_reply_invalid_response_data_type() {
    let mut deps = mock_dependencies();
    let env = mock_env();

    // encode a payload as protobuf
    // the payload is NOT a CompleteTransferResponse
    let execute_reply = MsgExecuteContractResponse {
        data: to_json_vec(&AssetInfo::NativeToken {
            denom: "denomA".to_string(),
        })
        .unwrap(),
    };
    let mut encoded_execute_reply = Vec::<u8>::with_capacity(execute_reply.encoded_len());
    execute_reply.encode(&mut encoded_execute_reply).unwrap();

    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(encoded_execute_reply.into()),
        }),
    };

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(err.to_string(), "failed to deserialize response data");
}

// 7. Failure: no contract in the response
#[test]
fn handle_complete_transfer_reply_no_response_contract() {
    let mut deps = mock_dependencies();
    let env = mock_env();

    // encode a payload as protobuf
    // the payload is a CompleteTransferResponse
    let execute_reply = MsgExecuteContractResponse {
        data: to_json_vec(&CompleteTransferResponse {
            contract: None,
            denom: None,
            recipient: "fake".to_string(),
            amount: 1u32.into(),
            relayer: "fake".to_string(),
            fee: 0u32.into(),
        })
        .unwrap(),
    };
    let mut encoded_execute_reply = Vec::<u8>::with_capacity(execute_reply.encoded_len());
    execute_reply.encode(&mut encoded_execute_reply).unwrap();

    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(encoded_execute_reply.into()),
        }),
    };

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "no contract in response, we should never get here"
    );
}

// 8. Failure: no current transfer in storage
#[test]
fn handle_complete_transfer_reply_no_storage() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("CvgBeyJjb250cmFjdCI6InNlaTE0MG02eGFnbXcwemVzZWp6aHN2azQ2enByZ3Njcjd0dTk0aDM2cndzdXRjc3hjczRmbWRzOXNldnltIiwiZGVub20iOm51bGwsInJlY2lwaWVudCI6InNlaTFka2R3ZHZrbngwcWF2NWNwNWt3Njhta24zcjk5bTNzdmt5amZ2a3p0d2g5N2R2MmxtMGtzajZ4cmFrIiwiYW1vdW50IjoiMTAwMCIsInJlbGF5ZXIiOiJzZWkxdmhrbTJxdjc4NHJ1bHg4eWxydTB6cHZ5dnczbTNjeTl4M3h5ZnYiLCJmZWUiOiIwIn0=").unwrap())
        })
    };

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "failed to load current transfer from storage"
    );
}

// 9. Failure: could not deserialize payload3 payload from stored transfer
#[test]
fn handle_complete_transfer_reply_wrong_stored_payload() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("CvgBeyJjb250cmFjdCI6InNlaTE0MG02eGFnbXcwemVzZWp6aHN2azQ2enByZ3Njcjd0dTk0aDM2cndzdXRjc3hjczRmbWRzOXNldnltIiwiZGVub20iOm51bGwsInJlY2lwaWVudCI6InNlaTFka2R3ZHZrbngwcWF2NWNwNWt3Njhta24zcjk5bTNzdmt5amZ2a3p0d2g5N2R2MmxtMGtzajZ4cmFrIiwiYW1vdW50IjoiMTAwMCIsInJlbGF5ZXIiOiJzZWkxdmhrbTJxdjc4NHJ1bHg4eWxydTB6cHZ5dnczbTNjeTl4M3h5ZnYiLCJmZWUiOiIwIn0=").unwrap())
        })
    };

    let bad_transfer_payload = TransferInfoResponse {
        amount: 0u32.into(),
        token_address: [0; 32],
        token_chain: 0,
        recipient: [0; 32],
        recipient_chain: 0,
        fee: 0u32.into(),
        payload: hex::decode("7b22726563697069656e74223a7b22726563697069656e74223a22633256704d575636637a56745a4731334f486436646d4e7a4f585a344f586b335a4774306357646c4d336c36626a52334d477735626a5130227d7d").unwrap()
    };
    CURRENT_TRANSFER
        .save(deps.as_mut().storage, &bad_transfer_payload)
        .unwrap();

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(err.to_string(), "failed to deserialize transfer payload");
}

// 10. Failure: could not convert the recipient (GatewayTransfer) base64 encoded bytes to a utf8 string
#[test]
fn handle_complete_transfer_reply_invalid_recipient() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("CvgBeyJjb250cmFjdCI6InNlaTE0MG02eGFnbXcwemVzZWp6aHN2azQ2enByZ3Njcjd0dTk0aDM2cndzdXRjc3hjczRmbWRzOXNldnltIiwiZGVub20iOm51bGwsInJlY2lwaWVudCI6InNlaTFka2R3ZHZrbngwcWF2NWNwNWt3Njhta24zcjk5bTNzdmt5amZ2a3p0d2g5N2R2MmxtMGtzajZ4cmFrIiwiYW1vdW50IjoiMTAwMCIsInJlbGF5ZXIiOiJzZWkxdmhrbTJxdjc4NHJ1bHg4eWxydTB6cHZ5dnczbTNjeTl4M3h5ZnYiLCJmZWUiOiIwIn0=").unwrap())
        })
    };

    let bad_transfer_payload = TransferInfoResponse {
        amount: 0u32.into(),
        token_address: [0; 32],
        token_chain: 0,
        recipient: [0; 32],
        recipient_chain: 0,
        fee: 0u32.into(),
        payload: hex::decode("7B22676174657761795F7472616E73666572223A7B22636861696E223A302C22726563697069656E74223A223256704D575636637A56745A4731334F486436646D4E7A4F585A344F586B335A4774306357646C4D336C36626A52334D477735626A5130222C22666565223A2230222C226E6F6E6365223A307D7D").unwrap()
    };
    CURRENT_TRANSFER
        .save(deps.as_mut().storage, &bad_transfer_payload)
        .unwrap();

    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "failed to convert 2VpMWV6czVtZG13OHd6dmNzOXZ4OXk3ZGt0cWdlM3l6bjR3MGw5bjQ0= to utf8 string"
    );
}

// 11. Failure: could not convert the contract (GatewayTransferWithPayload) base64 encoded bytes to a utf8 string
#[test]
fn handle_complete_transfer_reply_invalid_contract() {
    let mut deps = mock_dependencies();
    let env = mock_env();
    let msg = Reply {
        id: 1,
        result: cosmwasm_std::SubMsgResult::Ok(SubMsgResponse {
            events: vec![],
            data: Some(Binary::from_base64("Cv0BeyJjb250cmFjdCI6Indvcm1ob2xlMXl3NHd2MnpxZzl4a242N3p2cTNhenllMHQ4aDB4OWtneWczZDUzanltMjRneHQ0OXZkeXM2czhoN2EiLCJkZW5vbSI6bnVsbCwicmVjaXBpZW50Ijoic2VpMWRrZHdkdmtueDBxYXY1Y3A1a3c2OG1rbjNyOTltM3N2a3lqZnZrenR3aDk3ZHYybG0wa3NqNnhyYWsiLCJhbW91bnQiOiIxMDAwIiwicmVsYXllciI6InNlaTF2aGttMnF2Nzg0cnVseDh5bHJ1MHpwdnl2dzNtM2N5OXgzeHlmdiIsImZlZSI6IjAifQ==").unwrap())
        })
    };

    let bad_transfer_payload = TransferInfoResponse {
        amount: 0u32.into(),
        token_address: [0; 32],
        token_chain: 0,
        recipient: [0; 32],
        recipient_chain: 0,
        fee: 0u32.into(),
        payload: hex::decode("7B22676174657761795F7472616E736665725F776974685F7061796C6F6164223A7B22636861696E223A302C22636F6E7472616374223A223256704D575636637A56745A4731334F486436646D4E7A4F585A344F586B335A4774306357646C4D336C36626A52334D477735626A5130222C227061796C6F6164223A225647567A64464268655778765957513D222C226E6F6E6365223A307D7D").unwrap()
    };
    CURRENT_TRANSFER
        .save(deps.as_mut().storage, &bad_transfer_payload)
        .unwrap();

    // just verifying that we called the convert_cw20_to_bank -- unwrap the result without an error
    // other tests verify the correctness of this method
    // wormhole core and token bridge tests verify the correctness of the VAA parameters
    let err = handle_complete_transfer_reply(deps.as_mut(), env, msg).unwrap_err();
    assert_eq!(
        err.to_string(),
        "failed to convert 2VpMWV6czVtZG13OHd6dmNzOXZ4OXk3ZGt0cWdlM3l6bjR3MGw5bjQ0= to utf8 string"
    );
}

// Test convert_cw20_to_bank_and_send
// TESTS: convert_cw20_to_bank
// 1. Happy path
#[test]
fn convert_cw20_to_bank_and_send_happy_path() {
    let mut deps = default_custom_mock_deps();
    let env = mock_env();
    let recipient = WORMHOLE_USER_ADDR.to_string();
    let amount = 1;
    let contract_addr = WORMHOLE_CONTRACT_ADDR.to_string();

    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(
            deps.as_mut().storage,
            contract_addr.clone(),
            &tokenfactory_denom,
        )
        .unwrap();

    let chain_id = Chain::Ethereum;
    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, chain_id.into(), &channel)
        .unwrap();

    let response = convert_cw20_to_bank_and_send(
        deps.as_mut(),
        env,
        recipient,
        amount,
        contract_addr,
        chain_id.into(),
        None,
    )
    .unwrap();

    // response should have 2 messages:
    assert_eq!(response.messages.len(), 2);

    let mut expected_response: Response<TokenFactoryMsg> = Response::new();
    expected_response = expected_response.add_message(TokenMsg::MintTokens {
        denom: tokenfactory_denom,
        amount,
        mint_to_address: "cosmos2contract".to_string(),
    });
    expected_response = expected_response.add_message(Stargate {
        type_url: "/ibc.applications.transfer.v1.MsgTransfer".to_string(),
        value: Binary::from_base64("Cgh0cmFuc2ZlchIJY2hhbm5lbC0wGkkKRGZhY3RvcnkvY29zbW9zMmNvbnRyYWN0LzNRRVF5aTdpeUpId1E0d2ZVTUxGUEI0a1J6Y3pNQVhDaXRXaDdoNlRFVERhEgExIg9jb3Ntb3MyY29udHJhY3QqL3dvcm1ob2xlMXZoa20ycXY3ODRydWx4OHlscnUwenB2eXZ3M20zY3k5OWU2d3kwOL2irKOC6IugFg==").unwrap(),
    });

    // 1. TokenMsg::MintTokens
    assert_eq!(response.messages[0].msg, expected_response.messages[0].msg,);

    // 2. Stargate ibc transfer
    assert_eq!(response.messages[1].msg, expected_response.messages[1].msg,);
}

// 2. Happy path + CreateDenom on TokenFactory
#[test]
fn convert_cw20_to_bank_happy_path_create_denom() {
    let mut deps = default_custom_mock_deps();
    let env = mock_env();
    let recipient = WORMHOLE_USER_ADDR.to_string();
    let amount = 1;
    let contract_addr = WORMHOLE_CONTRACT_ADDR.to_string();

    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    let subdenom = "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();

    let chain_id = Chain::Ethereum;
    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, chain_id.into(), &channel)
        .unwrap();

    let token_info_response = TokenInfoResponse {
        name: "TestCoin".to_string(),
        symbol: "TEST".to_string(),
        decimals: 6,
        total_supply: Uint128::new(10_000_000),
    };
    let token_info_response_copy = token_info_response.clone();
    deps.querier.update_wasm(move |q| match q {
        WasmQuery::Smart {
            contract_addr: _,
            msg: _,
        } => SystemResult::Ok(ContractResult::Ok(
            to_json_binary(&token_info_response_copy).unwrap(),
        )),
        _ => SystemResult::Err(SystemError::UnsupportedRequest {
            kind: "wasm".to_string(),
        }),
    });

    // Populate token factory token's metadata from cw20 token's metadata
    let tf_description = token_info_response.name.clone()
        + ", "
        + token_info_response.symbol.as_str()
        + ", "
        + tokenfactory_denom.as_str();
    let tf_denom_unit_base = DenomUnit {
        denom: tokenfactory_denom.clone(),
        exponent: 0,
        aliases: vec![],
    };
    let tf_scaled_denom = "wormhole/".to_string()
        + subdenom.as_str()
        + "/"
        + token_info_response.decimals.to_string().as_str();
    let tf_denom_unit_scaled = DenomUnit {
        denom: tf_scaled_denom.clone(),
        exponent: u32::from(token_info_response.decimals),
        aliases: vec![],
    };
    let tf_metadata = Metadata {
        description: Some(tf_description),
        base: Some(tokenfactory_denom.clone()),
        denom_units: vec![tf_denom_unit_base, tf_denom_unit_scaled],
        display: Some(tf_scaled_denom),
        name: Some(token_info_response.name),
        symbol: Some(token_info_response.symbol),
    };

    let response = convert_cw20_to_bank_and_send(
        deps.as_mut(),
        env,
        recipient,
        amount,
        contract_addr,
        chain_id.into(),
        None,
    )
    .unwrap();

    // response should have 3 messages:
    assert_eq!(response.messages.len(), 3);

    let mut expected_response: Response<TokenFactoryMsg> = Response::new();
    expected_response = expected_response.add_message(TokenMsg::CreateDenom {
        subdenom,
        metadata: Some(tf_metadata),
    });
    expected_response = expected_response.add_message(TokenMsg::MintTokens {
        denom: tokenfactory_denom,
        amount,
        mint_to_address: "cosmos2contract".to_string(),
    });
    expected_response = expected_response.add_message(Stargate {
        type_url: "/ibc.applications.transfer.v1.MsgTransfer".to_string(),
        value: Binary::from_base64("Cgh0cmFuc2ZlchIJY2hhbm5lbC0wGkkKRGZhY3RvcnkvY29zbW9zMmNvbnRyYWN0LzNRRVF5aTdpeUpId1E0d2ZVTUxGUEI0a1J6Y3pNQVhDaXRXaDdoNlRFVERhEgExIg9jb3Ntb3MyY29udHJhY3QqL3dvcm1ob2xlMXZoa20ycXY3ODRydWx4OHlscnUwenB2eXZ3M20zY3k5OWU2d3kwOL2irKOC6IugFg==").unwrap(),
    });

    // 1. TokenMsg::CreateDenom
    assert_eq!(response.messages[0].msg, expected_response.messages[0].msg,);

    // 2. TokenMsg::MintTokens
    assert_eq!(response.messages[1].msg, expected_response.messages[1].msg,);

    // 3. Stargate ibc transfer
    assert_eq!(response.messages[2].msg, expected_response.messages[2].msg,);
}

// 3. Happy path + CreateDenom on TokenFactory with empty symbol
#[test]
fn convert_cw20_to_bank_happy_path_create_denom_empty_symbol() {
    let mut deps = default_custom_mock_deps();
    let env = mock_env();
    let recipient = WORMHOLE_USER_ADDR.to_string();
    let amount = 1;
    let contract_addr = WORMHOLE_CONTRACT_ADDR.to_string();

    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    let subdenom = "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();

    let chain_id = Chain::Ethereum;
    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, chain_id.into(), &channel)
        .unwrap();

    let token_info_response = TokenInfoResponse {
        name: "TestCoin".to_string(),
        symbol: "".to_string(),
        decimals: 6,
        total_supply: Uint128::new(10_000_000),
    };
    let token_info_response_copy = token_info_response.clone();
    deps.querier.update_wasm(move |q| match q {
        WasmQuery::Smart {
            contract_addr: _,
            msg: _,
        } => SystemResult::Ok(ContractResult::Ok(
            to_json_binary(&token_info_response_copy).unwrap(),
        )),
        _ => SystemResult::Err(SystemError::UnsupportedRequest {
            kind: "wasm".to_string(),
        }),
    });

    // Populate token factory token's metadata from cw20 token's metadata
    let tf_denom_unit_base = DenomUnit {
        denom: tokenfactory_denom.clone(),
        exponent: 0,
        aliases: vec![],
    };
    let tf_scaled_denom = "wormhole/".to_string()
        + subdenom.as_str()
        + "/"
        + token_info_response.decimals.to_string().as_str();
    let tf_denom_unit_scaled = DenomUnit {
        denom: tf_scaled_denom.clone(),
        exponent: u32::from(token_info_response.decimals),
        aliases: vec![],
    };
    let tf_description = token_info_response.name.clone()
        + ", "
        + tf_scaled_denom.as_str() // CW20 symbol is empty, use tf_scaled_denom
        + ", "
        + tokenfactory_denom.as_str();
    let tf_metadata = Metadata {
        description: Some(tf_description),
        base: Some(tokenfactory_denom.clone()),
        denom_units: vec![tf_denom_unit_base, tf_denom_unit_scaled],
        display: Some(tf_scaled_denom.clone()),
        name: Some(token_info_response.name),
        symbol: Some(tf_scaled_denom), // CW20 symbol is empty, use tf_scaled_denom
    };

    let response = convert_cw20_to_bank_and_send(
        deps.as_mut(),
        env,
        recipient,
        amount,
        contract_addr,
        chain_id.into(),
        None,
    )
    .unwrap();

    // response should have 3 messages:
    assert_eq!(response.messages.len(), 3);

    let mut expected_response: Response<TokenFactoryMsg> = Response::new();
    expected_response = expected_response.add_message(TokenMsg::CreateDenom {
        subdenom,
        metadata: Some(tf_metadata),
    });
    expected_response = expected_response.add_message(TokenMsg::MintTokens {
        denom: tokenfactory_denom,
        amount,
        mint_to_address: "cosmos2contract".to_string(),
    });
    expected_response = expected_response.add_message(Stargate {
        type_url: "/ibc.applications.transfer.v1.MsgTransfer".to_string(),
        value: Binary::from_base64("Cgh0cmFuc2ZlchIJY2hhbm5lbC0wGkkKRGZhY3RvcnkvY29zbW9zMmNvbnRyYWN0LzNRRVF5aTdpeUpId1E0d2ZVTUxGUEI0a1J6Y3pNQVhDaXRXaDdoNlRFVERhEgExIg9jb3Ntb3MyY29udHJhY3QqL3dvcm1ob2xlMXZoa20ycXY3ODRydWx4OHlscnUwenB2eXZ3M20zY3k5OWU2d3kwOL2irKOC6IugFg==").unwrap(),
    });

    // 1. TokenMsg::CreateDenom
    assert_eq!(response.messages[0].msg, expected_response.messages[0].msg,);

    // 2. TokenMsg::MintTokens
    assert_eq!(response.messages[1].msg, expected_response.messages[1].msg,);

    // 3. Stargate ibc transfer
    assert_eq!(response.messages[2].msg, expected_response.messages[2].msg,);
}

// 4. Failure: couldn't validate contract address
#[test]
fn convert_cw20_to_bank_failure_invalid_contract() {
    let mut deps = default_custom_mock_deps();
    let env = mock_env();
    let recipient = WORMHOLE_USER_ADDR.to_string();
    let amount = 1;
    let contract_addr = "badContractAddr".to_string();
    let chain_id = Chain::Ethereum;

    let method_err = convert_cw20_to_bank_and_send(
        deps.as_mut(),
        env,
        recipient,
        amount,
        contract_addr,
        chain_id.into(),
        None,
    )
    .unwrap_err();
    assert_eq!(
        method_err.to_string(),
        "invalid contract address badContractAddr"
    );
}

// 5. Failure: Chain id doesn't have a channel
#[test]
fn convert_cw20_to_bank_and_send_chain_id_no_channel() {
    let mut deps = default_custom_mock_deps();
    let env = mock_env();
    let recipient = WORMHOLE_USER_ADDR.to_string();
    let amount = 1;
    let contract_addr = WORMHOLE_CONTRACT_ADDR.to_string();

    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(
            deps.as_mut().storage,
            contract_addr.clone(),
            &tokenfactory_denom,
        )
        .unwrap();

    let chain_id = Chain::Ethereum;

    let method_err = convert_cw20_to_bank_and_send(
        deps.as_mut(),
        env,
        recipient,
        amount,
        contract_addr,
        chain_id.into(),
        None,
    )
    .unwrap_err();

    assert_eq!(
        method_err.to_string(),
        "chain id does not have an allowed channel"
    );
}

// 6. Failure: bad payload
#[test]
fn convert_cw20_to_bank_and_send_bad_payload() {
    let mut deps = default_custom_mock_deps();
    let env = mock_env();
    let recipient = WORMHOLE_USER_ADDR.to_string();
    let amount = 1;
    let contract_addr = WORMHOLE_CONTRACT_ADDR.to_string();

    let tokenfactory_denom =
        "factory/cosmos2contract/3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa".to_string();
    CW_DENOMS
        .save(
            deps.as_mut().storage,
            contract_addr.clone(),
            &tokenfactory_denom,
        )
        .unwrap();

    let chain_id = Chain::Ethereum;
    let channel = "channel-0".to_string();
    CHAIN_TO_CHANNEL_MAP
        .save(deps.as_mut().storage, chain_id.into(), &channel)
        .unwrap();

    let method_err = convert_cw20_to_bank_and_send(
        deps.as_mut(),
        env,
        recipient,
        amount,
        contract_addr,
        chain_id.into(),
        Some(
            Binary::from_base64("2VpMWV6czVtZG13OHd6dmNzOXZ4OXk3ZGt0cWdlM3l6bjR3MGw5bjQ0").unwrap(),
        ),
    )
    .unwrap_err();

    assert_eq!(
        method_err.to_string(),
        "failed to convert 2VpMWV6czVtZG13OHd6dmNzOXZ4OXk3ZGt0cWdlM3l6bjR3MGw5bjQ0= to utf8 string"
    );
}

// 1. happy path
#[test]
fn contract_addr_to_base58_happy_path() {
    let deps = default_custom_mock_deps();
    let b58_str = contract_addr_to_base58(
        deps.as_ref(),
        "wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7a".to_string(),
    )
    .unwrap();
    assert_eq!(b58_str, "3QEQyi7iyJHwQ4wfUMLFPB4kRzczMAXCitWh7h6TETDa");
}

// 2. bad contract address, could not canonicalize contract address
#[test]
fn contract_addr_to_base58_bad_contract_address() {
    let deps = default_custom_mock_deps();
    let method_err = contract_addr_to_base58(
        deps.as_ref(),
        "wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7".to_string(),
    )
    .unwrap_err();
    assert_eq!(
        method_err.to_string(),
        "could not canonicalize contract address wormhole1yw4wv2zqg9xkn67zvq3azye0t8h0x9kgyg3d53jym24gxt49vdys6s8h7"
    )
}
