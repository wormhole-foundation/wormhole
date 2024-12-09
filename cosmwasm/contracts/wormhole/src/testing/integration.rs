use crate::msg::{GetAddressHexResponse, GetStateResponse, GuardianSetInfoResponse};
use crate::testing::utils::{
    create_gov_vaa_body, create_transfer_vaa_body, instantiate_with_faker_guardians, sign_vaa_body,
    WormholeApp,
};
use crate::{
    contract::instantiate,
    msg::QueryMsg,
    state::{ConfigInfo, GuardianAddress, ParsedVAA, CONFIG_KEY},
    testing::utils::instantiate_with_guardians,
};
use cosmwasm_std::Uint128;
use cosmwasm_std::{
    from_slice,
    testing::{mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage},
    to_binary, Coin, OwnedDeps, Response, StdResult, Storage,
};
use cosmwasm_storage::to_length_prefixed;
use cw_multi_test::Executor;
use serde::{de::IntoDeserializer, Deserialize};
use serde_wormhole::RawMessage;
use std::convert::TryInto;
use wormhole_sdk::core::{Action, GovernancePacket};
use wormhole_sdk::token::Message;
use wormhole_sdk::{Amount, Chain, GOVERNANCE_EMITTER};

static INITIALIZER: &str = "initializer";
static GOV_ADDR: &[u8] = b"GOVERNANCE_ADDRESS";

fn get_config_info<S: Storage>(storage: &S) -> ConfigInfo {
    let key = to_length_prefixed(CONFIG_KEY);
    let data = storage.get(&key).expect("data should exist");
    from_slice(&data).expect("invalid data")
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
            gov_chain: 0,
            gov_address: GOV_ADDR.to_vec(),
            fee: Coin::new(0, "uluna"),
            chain_id: 18,
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
        admin,
        user,
        wormhole_contract,
        wormhole_keeper,
    } = WormholeApp::new(instantiate_with_faker_guardians());

    let (_, signed_vaa) = sign_vaa_body(wormhole_keeper, create_gov_vaa_body(1, "test"));

    // Query verify VAA
    let parsed_vaa: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: 0,
        },
    )?;

    let test_payload = serde_wormhole::from_slice::<String>(&parsed_vaa.payload.as_slice());
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
        admin,
        user,
        wormhole_contract,
        wormhole_keeper,
    } = WormholeApp::new(instantiate_with_faker_guardians());
    let (_, signed_vaa) = sign_vaa_body(
        wormhole_keeper.clone(),
        create_transfer_vaa_body(1, GOVERNANCE_EMITTER),
    );

    let vaa_response: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: 0,
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
        serde_wormhole::from_slice::<Message<&RawMessage>>(&vaa_response.payload.as_slice());

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
    let (_, signed_vaa) = sign_vaa_body(
        wormhole_keeper,
        create_gov_vaa_body(
            2,
            GovernancePacket {
                chain: Chain::Wormchain,
                action: Action::SetFee {
                    amount: Amount(*b"00000000000000000000000000000012"),
                },
            },
        ),
    );

    let vaa_response: ParsedVAA = app.wrap().query_wasm_smart(
        wormhole_contract.clone(),
        &QueryMsg::VerifyVAA {
            vaa: signed_vaa,
            block_time: 0,
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

    Ok(())
}
