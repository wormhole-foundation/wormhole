use crate::testing::utils::{IntoGuardianAddress, WormholeApp};
use cosmwasm_std::StdResult;
use cosmwasm_std::Uint256;
use cw_multi_test::Executor;
use cw_wormhole::{
    msg::QueryMsg,
    msg::{ExecuteMsg, GuardianSetInfoResponse},
    state::ParsedVAA,
};
use k256::ecdsa::SigningKey;
use wormhole_bindings::fake::{create_gov_vaa_body, SignVaa};
use wormhole_sdk::{
    core::{Action, GovernancePacket},
    Address, Amount, Chain, GuardianSetInfo,
};

#[test]
fn post_message_blocked_in_shutdown() {
    let WormholeApp {
        mut app,
        wormhole_contract,
        user,
        ..
    } = WormholeApp::new_with_faker_guardians();

    // Attempt to post a message - should fail due to shutdown
    let post_message_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::PostMessage {
            message: b"test".into(),
            nonce: 2,
        },
        &[],
    );

    assert!(
        post_message_response.is_err(),
        "Post Message should fail in shutdown mode with \"ContractShutdown\""
    );
}

#[test]
fn fee_change_blocked_in_shutdown() -> StdResult<()> {
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

    // Attempt to submit fee change VAA - should fail due to shutdown
    let vaa_response = app.execute_contract(
        user.clone(),
        wormhole_contract.clone(),
        &ExecuteMsg::SubmitVAA {
            vaa: signed_vaa.clone(),
        },
        &[],
    );
    println!("vaa resp {vaa_response:?}");

    assert!(
        vaa_response.is_err(),
        "SetFee VAA submission should fail in shutdown mode with \"ContractShutdown\""
    );

    Ok(())
}

#[test]
fn transfer_fee_blocked_in_shutdown() -> StdResult<()> {
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
            action: Action::TransferFee {
                amount: Amount(Uint256::from(100u128).to_be_bytes()),
                recipient: Address([1u8; 32]),
            },
        },
    );

    let (_, signed_vaa) = vaa_body.clone().sign_vaa(&wormhole_keeper);

    // Attempt to submit transfer fee VAA - should fail due to shutdown
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
        "TransferFee VAA submission should fail in shutdown mode with \"ContractShutdown\""
    );

    Ok(())
}

#[test]
pub fn guardian_set_update_allowed_in_shutdown() -> StdResult<()> {
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

    // Query the current guardian set
    let guardian_set_response: GuardianSetInfoResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GuardianSetInfo {})?;

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

    let (_, signed_guardian_set_update_vaa) = update_guardian_set_vaa_body
        .clone()
        .sign_vaa(&wormhole_keeper);

    // Submit guardian set update VAA - should succeed despite shutdown
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
        "Guardian set update should succeed even in shutdown mode"
    );

    // Verify the guardian set was actually updated
    let new_guardian_set_response: GuardianSetInfoResponse = app
        .wrap()
        .query_wasm_smart(wormhole_contract.clone(), &QueryMsg::GuardianSetInfo {})?;

    assert_eq!(
        new_guardian_set_response.guardian_set_index,
        guardian_set_response.guardian_set_index + 1,
        "Guardian set index should be incremented"
    );
    assert_eq!(
        new_guardian_set_response.addresses.len(),
        2,
        "New guardian set should have 2 guardians"
    );

    Ok(())
}

#[test]
pub fn verify_vaa_allowed_in_shutdown() -> StdResult<()> {
    let WormholeApp {
        app,
        wormhole_contract,
        wormhole_keeper,
        ..
    } = WormholeApp::new_with_faker_guardians();

    let (_, signed_vaa) = create_gov_vaa_body(1, "test").sign_vaa(&wormhole_keeper);

    // Query verify VAA - should work despite shutdown
    let parsed_vaa: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: app.block_info().height,
        },
    )?;

    assert_eq!(parsed_vaa.version, 1, "version should match");
    assert_eq!(
        parsed_vaa.guardian_set_index, 0,
        "guardian set index should match"
    );

    Ok(())
}
