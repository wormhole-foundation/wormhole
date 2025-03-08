use crate::msg::{ExecuteMsg, GetStateResponse, GuardianSetInfoResponse};
use crate::testing::utils::{
    create_transfer_vaa_body, instantiate_with_guardians, sign_vaa_body_version_2,
    IntoGuardianAddress, WormholeApp,
};
use crate::{
    contract::instantiate,
    msg::QueryMsg,
    state::{ConfigInfo, GuardianAddress, ParsedVAA, CONFIG_KEY},
};
use cosmwasm_std::{
    from_json,
    testing::{mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage},
    Coin, OwnedDeps, Response, StdResult, Storage,
};
use cosmwasm_std::{Deps, DepsMut, Empty, QuerierWrapper, StdError, Uint128, Uint256};
use cosmwasm_storage::to_length_prefixed;
use cw_multi_test::{ContractWrapper, Executor};
use k256::ecdsa::SigningKey;
use serde_wormhole::RawMessage;
use std::ops::Deref;
use wormhole_bindings::fake::{create_gov_vaa_body, SignVaa, WormholeKeeper};
use wormhole_sdk::core::{Action, GovernancePacket};
use wormhole_sdk::token::Message;
use wormhole_sdk::{relayer, Address, Amount, Chain, GuardianSetInfo, GOVERNANCE_EMITTER};

static INITIALIZER: &str = "initializer";

fn get_config_info<S: Storage>(storage: &S) -> ConfigInfo {
    let key = to_length_prefixed(CONFIG_KEY);
    let data = storage.get(&key).expect("data should exist");
    from_json(&data).expect("invalid data")
}

fn do_init(guardians: &[GuardianAddress]) -> OwnedDeps<MockStorage, MockApi, MockQuerier> {
    let mut deps = mock_dependencies();
    let init_msg = instantiate_with_guardians(guardians);
    let env = mock_env();
    let info = mock_info(INITIALIZER, &[]);
    let res: Response = instantiate(deps.as_mut(), env, info, init_msg).unwrap();
    assert_eq!(0, res.messages.len());

    // query the store directly
    assert_eq!(
        get_config_info(&deps.storage),
        ConfigInfo {
            guardian_set_index: 0,
            guardian_set_expirity: 50,
            gov_chain: Chain::Solana.into(),
            gov_address: GOVERNANCE_EMITTER.0.to_vec(),
            fee: Coin::new(0, "uluna"),
            chain_id: Chain::Terra2.into(),
            fee_denom: "uluna".to_string(),
        }
    );
    deps
}

#[test]
fn init_works() {
    let guardians = [GuardianAddress {
        bytes: hex::decode("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")
            .expect("Decoding failed")
            .into(),
    }];
    let _deps = do_init(&guardians);
}

#[test]
fn queries_test() -> StdResult<()> {
    let WormholeApp {
        app,
        wormhole_contract,
        wormhole_keeper,
        ..
    } = WormholeApp::new_with_faker_guardians();

    let (_, signed_vaa) = create_gov_vaa_body(1, "test").sign_vaa(&wormhole_keeper);

    // Query verify VAA
    let parsed_vaa: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    )?;

    let test_payload = serde_wormhole::from_slice::<String>(parsed_vaa.payload.as_slice());
    assert!(test_payload.is_ok(), "failed to parse test payload");
    assert_eq!(test_payload.unwrap(), "test", "test payload does not match");

    assert_eq!(parsed_vaa.version, 1, "version does not match");
    assert_eq!(
        parsed_vaa.guardian_set_index, 0,
        "guardian set index does not match"
    );

    // Query guardian set info
    let guardian_set_response: GuardianSetInfoResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GuardianSetInfo {})?;

    assert_eq!(
        guardian_set_response.guardian_set_index, 0u32,
        "guardian set index does not match"
    );
    assert_eq!(
        guardian_set_response.addresses.len(),
        7,
        "guardian set length does not match"
    );

    // Query get state
    let get_state_resp: GetStateResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GetState {})?;
    assert_eq!(
        get_state_resp.fee.denom, "uluna",
        "fee denom does not match"
    );
    assert_eq!(
        get_state_resp.fee.amount,
        Uint128::from(0u128),
        "fee amount does not match"
    );

    // TODO: set the appropriate MockedApi in the AppBuilder so that QueryAddressHex can be integration tested
    // This should be simple once we're on cosmwasm 1.5+ https://docs.rs/cw-multi-test/1.2.0/cw_multi_test/struct.SimpleAddressGenerator.html

    Ok(())
}

#[test]
fn verify_vaas_query() -> StdResult<()> {
    let WormholeApp {
        app,
        wormhole_contract,
        wormhole_keeper,
        ..
    } = WormholeApp::new_with_faker_guardians();
    let (_, signed_vaa) =
        create_transfer_vaa_body(1, GOVERNANCE_EMITTER).sign_vaa(&wormhole_keeper.clone());

    let vaa_response: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    )?;

    assert_eq!(vaa_response.version, 1, "version does not match");
    assert_eq!(
        vaa_response.guardian_set_index, 0,
        "guardian set index does not match"
    );
    assert_eq!(vaa_response.timestamp, 1, "timestamp does not match");
    assert_eq!(vaa_response.nonce, 1, "nonce does not match");
    assert_eq!(vaa_response.len_signers, 7, "len signers does not match");
    assert_eq!(
        vaa_response.emitter_chain, 1,
        "emitter chain does not match"
    );
    assert_eq!(vaa_response.sequence, 1, "sequence does not match");
    assert_eq!(
        vaa_response.consistency_level, 32,
        "consistency level does not match"
    );
    assert_eq!(
        vaa_response.emitter_address.as_slice(),
        GOVERNANCE_EMITTER.0.as_slice(),
        "emitter address does not match"
    );

    let transfer_payload =
        serde_wormhole::from_slice::<Message<&RawMessage>>(vaa_response.payload.as_slice());

    assert!(transfer_payload.is_ok(), "failed to parse transfer payload");
    assert!(
        matches!(
            transfer_payload.unwrap(),
            Message::Transfer {
                token_chain: Chain::Solana,
                ..
            }
        ),
        "unexpected payload"
    );

    // Verify a governance VAA
    let (_, signed_vaa) = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Osmosis,
            action: Action::SetFee {
                amount: Amount(*b"00000000000000000000000000000012"),
            },
        },
    )
    .sign_vaa(&wormhole_keeper);

    let vaa_response: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    )?;

    assert_eq!(
        vaa_response.version, 1,
        "governance vaa version does not match"
    );
    assert_eq!(
        vaa_response.guardian_set_index, 0,
        "governance vaa guardian set index does not match"
    );
    let governance_payload =
        serde_wormhole::from_slice::<GovernancePacket>(vaa_response.payload.as_slice());

    assert!(
        governance_payload.is_ok(),
        "failed to parse governance payload"
    );
    assert!(
        matches!(
            governance_payload.unwrap(),
            GovernancePacket {
                action: Action::SetFee { .. },
                chain: Chain::Osmosis,
            }
        ),
        "unexpected payload"
    );

    Ok(())
}

#[test]
fn verify_vaa_failure_modes() -> StdResult<()> {
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    let vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::SetFee {
                amount: Amount(Uint256::from(1u128).to_be_bytes()),
            },
        },
    );

    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);
    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa.clone(),
            block_time: app.block_info().height,
        },
    );
    assert!(
        vaa_response.is_ok(),
        "VAA signed by the proper guardianset should verify successfully"
    );

    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: u64::MAX,
        },
    );
    assert!(
        vaa_response.is_err(),
        "VAA should fail if the guardian set is past it's expiry \"GuardianSetExpired\""
    );

    // VAA signed with a nonstandard version listed in the header
    let (_, signed_vaa) = sign_vaa_body_version_2(wormhole_keeper.clone(), vaa_body.clone());

    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa.clone(),
            block_time: app.block_info().height,
        },
    );

    assert!(
        vaa_response.is_err(),
        "VAA should fail \"InvalidVersion\" when signed with a nonstandard version"
    );

    // VAA signed with a non-matching guardianset
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(
        // signing with 7 guardians
        &WormholeKeeper::new(),
    );

    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    );
    assert!(
        vaa_response.is_err(),
        "VAA with more guardians than the established guardian set should fail \"TooManySignatures\""
    );

    // VAA signed with a non-matching guardianset
    let guardian_keys: Vec<SigningKey> = vec![];
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(
        // signing with 0 guardians
        &guardian_keys.into(),
    );

    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    );

    assert!(
        vaa_response.is_err(),
        "VAA with fewer guardians than the established guardian set should fail \"NoQuorum\""
    );

    // VAA signed with a different guardian
    let guardian_keys: Vec<SigningKey> = vec![SigningKey::from_bytes(&[
        121, 51, 199, 93, 237, 227, 62, 220, 128, 129, 195, 4, 190, 163, 254, 12, 212, 224, 188,
        76, 141, 242, 229, 121, 192, 5, 161, 176, 136, 99, 83, 53,
    ])
    .unwrap()];
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&guardian_keys.into());

    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    );
    assert!(
        vaa_response.is_err(),
        "VAA signed by a guardian not in the established guardian set should fail \"GuardianSignatureError\""
    );

    // Verifying a VAA that's already been executed should fail
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    // Submit the VAA first
    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(
        vaa_response.is_ok(),
        "VAA submission should succeed when the VAA has not been executed"
    );

    // Attempt to verify the VAA again after it's been executed
    let vaa_response: StdResult<ParsedVAA> = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    );
    assert!(
        vaa_response.is_err(),
        "VAA that has already been executed should fail \"VAAAlreadyExecuted\""
    );

    Ok(())
}

#[test]
#[ignore]
pub fn update_contract_gov_vaa() -> StdResult<()> {
    /// TODO: This test is disabled because it requires cw_multi_test 0.16+ to update the contract admin
    use wormchain_ibc_receiver::contract::{
        execute as receiver_execute, instantiate as receiver_instantiate, query as receiver_query,
    };
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        admin,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    // We have to give the wormhole contract admin rights over itself so that it can migrate itself
    let update_admin_response = app.execute(
        admin.clone(),
        cosmwasm_std::CosmosMsg::Wasm(cosmwasm_std::WasmMsg::UpdateAdmin {
            contract_addr: wormhole_contract.to_string(),
            admin: wormhole_contract.to_string(),
        }),
    );

    assert!(
        update_admin_response.is_ok(),
        "Update Contract Admin should succeed"
    );

    // store the wormchain_ibc_receiver contract so we can migrate to it
    let new_code_id = app.store_code(Box::new(ContractWrapper::new(
        |deps, env, info, msg| {
            receiver_execute(deps, env, info, msg)
                .map_err(|anyhow_err| StdError::generic_err(anyhow_err.to_string()))
        },
        |deps, env, info, msg| {
            receiver_instantiate(
                DepsMut {
                    storage: deps.storage,
                    api: deps.api,
                    querier: QuerierWrapper::new(deps.querier.deref()),
                },
                env,
                info,
                msg,
            )
            .map_err(|anyhow_err| StdError::generic_err(anyhow_err.to_string()))
        },
        |deps, env, msg| {
            receiver_query(
                Deps {
                    storage: deps.storage,
                    api: deps.api,
                    querier: QuerierWrapper::<Empty>::new(deps.querier.deref()),
                },
                env,
                msg,
            )
            .map_err(|anyhow_err| StdError::generic_err(anyhow_err.to_string()))
        },
    )));

    let vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::ContractUpgrade {
                new_contract: Address(Uint256::from(new_code_id).to_be_bytes()),
            },
        },
    );

    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    // Submit the VAA first
    let vaa_response = app.execute_contract(
        admin.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );

    assert!(
        vaa_response.is_ok(),
        "Update Contract VAA submission should succeed"
    );

    Ok(())
}

#[test]
#[ignore]
pub fn set_fee_gov_vaa() -> StdResult<()> {
    // TODO: set the appropriate MockedApi in the AppBuilder so that PostMessage can be integration tested
    // This should be simple once we're on cosmwasm 1.5+ https://docs.rs/cw-multi-test/1.2.0/cw_multi_test/struct.SimpleAddressGenerator.html
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    // At this point there is no fee and this should be a free action.
    let post_message_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::PostMessage {
            message: b"test".into(),
            nonce: 1,
        },
        &[],
    );

    assert!(
        post_message_response.is_ok(),
        "Post Message should succeed when there is no fee"
    );

    let vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::SetFee {
                amount: Amount(Uint256::from(18u128).to_be_bytes()),
            },
        },
    );

    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    // Submit the VAA first
    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(vaa_response.is_ok(), "SetFee VAA submission should succeed");

    // At this point there is a fee and this should fail since we aren't paying the fee.
    let post_message_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::PostMessage {
            message: b"test".into(),
            nonce: 1,
        },
        &[],
    );

    assert!(
        post_message_response.is_err(),
        "Post Message should fail \"FeeTooLow\""
    );

    Ok(())
}

#[test]
pub fn set_fee_gov_vaa_2() -> StdResult<()> {
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    let vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::SetFee {
                amount: Amount(Uint256::from(18u128).to_be_bytes()),
            },
        },
    );

    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    // Submit the VAA first
    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(vaa_response.is_ok(), "SetFee VAA submission should succeed");

    // now query the state and see if the fee has been updated
    let get_state_resp: GetStateResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GetState {})?;
    assert_eq!(
        get_state_resp.fee.denom, "uluna",
        "fee denom does not match"
    );
    assert_eq!(
        get_state_resp.fee.amount,
        Uint128::from(18u128),
        "fee amount does not match"
    );

    Ok(())
}

#[test]
pub fn submit_vaa_replay_protection() {
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    let vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::SetFee {
                amount: Amount(Uint256::from(18u128).to_be_bytes()),
            },
        },
    );

    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    // Submit the VAA first
    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(vaa_response.is_ok(), "SetFee VAA submission should succeed");

    // Submit the VAA again
    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(
        vaa_response.is_err(),
        "Submitting the same VAA twice should fail"
    );
}

#[test]
pub fn only_gov_vaas_allowed() {
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    let vaa_body = create_transfer_vaa_body(1, Address([100u8; 32]));
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );

    assert!(
        vaa_response.is_err(),
        "VAA submission should fail \"InvalidVAA\" when not a governance VAA"
    );
}

#[test]
pub fn only_core_module_vaas_allowed() {
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    let vaa_body = create_gov_vaa_body(
        1,
        relayer::GovernancePacket {
            chain: Chain::Terra2,
            action: relayer::Action::RegisterChain {
                chain: Chain::Solana,
                emitter_address: Address([0u8; 32]),
            },
        },
    );
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(
        vaa_response.is_err(),
        "VAA submission should fail \"this is not a valid module\" when not a core module VAA"
    );
}

#[test]
pub fn update_guardian_set() -> StdResult<()> {
    let WormholeApp {
        mut app,
        wormhole_contract,
        wormhole_keeper,
        user,
        ..
    } = WormholeApp::new_with_guardians(vec![SigningKey::from_bytes(&[
        93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238, 206,
        15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
    ])
    .unwrap()]);

    let vaa_body = create_gov_vaa_body(
        1,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::SetFee {
                amount: Amount([0u8; 32]),
            },
        },
    );
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    assert!(
        vaa_response.is_ok(),
        "VAA submission with initial guardian set should succeed"
    );

    // Add a second guardian
    let new_guardian_keys = vec![
        SigningKey::from_bytes(&[
            93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232, 199, 238,
            206, 15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
        ])
        .unwrap(),
        SigningKey::from_bytes(&[
            150, 48, 135, 223, 194, 186, 243, 139, 177, 8, 126, 32, 210, 57, 42, 28, 29, 102, 196,
            201, 106, 136, 40, 149, 218, 150, 240, 213, 192, 128, 161, 245,
        ])
        .unwrap(),
    ];

    // Query the current guardian set so we know what the next index should be
    let guardian_set_response: GuardianSetInfoResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GuardianSetInfo {})?;

    let invalid_guardian_set_vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::GuardianSetUpgrade {
                // This should fail because the index should only increase by one
                new_guardian_set_index: guardian_set_response.guardian_set_index + 2,
                new_guardian_set: GuardianSetInfo {
                    addresses: new_guardian_keys
                        .iter()
                        .map(|key| -> wormhole_sdk::GuardianAddress {
                            key.clone().into_guardian_address()
                        })
                        .collect(),
                },
            },
        },
    );

    let (_, signed_guardian_set_update_vaa) = invalid_guardian_set_vaa_body
        .clone()
        .sign_vaa(&wormhole_keeper);

    let guardian_set_update_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_guardian_set_update_vaa.clone(),
        },
        &[],
    );
    assert!(
        guardian_set_update_response.is_err(),
        "UpdateGuardianSet VAA submission should fail \"InvalidGuardianSetIndex\""
    );

    let update_guardian_set_vaa_body = create_gov_vaa_body(
        2,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::GuardianSetUpgrade {
                new_guardian_set_index: guardian_set_response.guardian_set_index + 1,
                new_guardian_set: GuardianSetInfo {
                    addresses: new_guardian_keys
                        .iter()
                        .map(|key| -> wormhole_sdk::GuardianAddress {
                            key.clone().into_guardian_address()
                        })
                        .collect(),
                },
            },
        },
    );

    // Sign with the current singular guardian
    let (_, signed_guardian_set_update_vaa) = update_guardian_set_vaa_body
        .clone()
        .sign_vaa(&wormhole_keeper);

    let guardian_set_update_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_guardian_set_update_vaa.clone(),
        },
        &[],
    );
    assert!(
        guardian_set_update_response.is_ok(),
        "UpdateGuardianSet VAA submission should succeed"
    );

    let wormhole_keeper: WormholeKeeper = new_guardian_keys.into();
    wormhole_keeper.set_index(guardian_set_response.guardian_set_index + 1);

    let vaa_body = create_gov_vaa_body(
        1,
        GovernancePacket {
            chain: Chain::Terra2,
            action: Action::SetFee {
                amount: Amount(Uint256::from(1u128).to_be_bytes()),
            },
        },
    );
    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );

    assert!(
        vaa_response.is_ok(),
        "VAA submission with updated guardian set should succeed"
    );

    let get_state_resp: GetStateResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GetState {})?;
    assert_eq!(
        get_state_resp.fee.amount,
        Uint128::from(1u128),
        "Fee should have been updated to 1uluna"
    );

    Ok(())
}
