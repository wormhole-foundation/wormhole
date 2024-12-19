mod helpers;

use cosmwasm_std::{to_json_binary, Event};
use global_accountant::msg::ChainRegistrationResponse;
use helpers::*;
use wormhole_sdk::{
    token::{Action, GovernancePacket},
    vaa::Body,
    Address, Chain,
};

fn create_vaa_body() -> Body<GovernancePacket> {
    Body {
        timestamp: 1,
        nonce: 1,
        emitter_chain: Chain::Solana,
        emitter_address: wormhole_sdk::GOVERNANCE_EMITTER,
        sequence: 15920283,
        consistency_level: 0,
        payload: GovernancePacket {
            chain: Chain::Any,
            action: Action::RegisterChain {
                chain: Chain::Solana,
                emitter_address: Address([
                    0xc6, 0x9a, 0x1b, 0x1a, 0x65, 0xdd, 0x33, 0x6b, 0xf1, 0xdf, 0x6a, 0x77, 0xaf,
                    0xb5, 0x01, 0xfc, 0x25, 0xdb, 0x7f, 0xc0, 0x93, 0x8c, 0xb0, 0x85, 0x95, 0xa9,
                    0xef, 0x47, 0x32, 0x65, 0xcb, 0x4f,
                ]),
            },
        },
    }
}

#[test]
fn any_target() {
    let (wh, mut contract) = proper_instantiate();

    let body = create_vaa_body();

    let (v, data) = sign_vaa_body(&wh, body);
    let resp = contract
        .submit_vaas(vec![data])
        .expect("failed to submit chain registration");

    let Action::RegisterChain {
        chain,
        emitter_address,
    } = v.payload.action
    else {
        panic!()
    };

    resp.assert_event(
        &Event::new("wasm-RegisterChain")
            .add_attribute("chain", chain.to_string())
            .add_attribute("emitter_address", emitter_address.to_string()),
    );

    let ChainRegistrationResponse { address } =
        contract.query_chain_registration(chain.into()).unwrap();
    assert_eq!(&*address, &emitter_address.0);
}

#[test]
fn wormchain_target() {
    let (wh, mut contract) = proper_instantiate();

    let mut body = create_vaa_body();
    body.payload.chain = Chain::Wormchain;

    let (v, data) = sign_vaa_body(&wh, body);
    let resp = contract
        .submit_vaas(vec![data])
        .expect("failed to submit chain registration");

    let Action::RegisterChain {
        chain,
        emitter_address,
    } = v.payload.action
    else {
        panic!()
    };

    resp.assert_event(
        &Event::new("wasm-RegisterChain")
            .add_attribute("chain", chain.to_string())
            .add_attribute("emitter_address", emitter_address.to_string()),
    );

    let ChainRegistrationResponse { address } =
        contract.query_chain_registration(chain.into()).unwrap();
    assert_eq!(&*address, &emitter_address.0);
}

#[test]
fn wrong_target() {
    let (wh, mut contract) = proper_instantiate();

    let mut body = create_vaa_body();
    body.payload.chain = Chain::Oasis;

    let (_, data) = sign_vaa_body(&wh, body);
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed chain registration VAA for different chain");
    assert_eq!(
        "this token governance vaa is for another chain",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn non_governance_chain() {
    let (wh, mut contract) = proper_instantiate();

    let mut body = create_vaa_body();
    body.emitter_chain = Chain::Fantom;

    let (_, data) = sign_vaa_body(&wh, body);
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed chain registration with non-governance chain");

    // A governance message with wrong chain or emitter will be parsed as a token bridge message
    assert!(err
        .source()
        .unwrap()
        .to_string()
        .contains("failed to parse tokenbridge message",));
}

#[test]
fn non_governance_emitter() {
    let (wh, mut contract) = proper_instantiate();

    let mut body = create_vaa_body();
    body.emitter_address = Address([0x88; 32]);

    let (_, data) = sign_vaa_body(&wh, body);
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed chain registration with non-governance emitter");

    // A governance message with wrong chain or emitter will be parsed as a token bridge message
    assert!(err
        .source()
        .unwrap()
        .to_string()
        .contains("failed to parse tokenbridge message",));
}

#[test]
fn duplicate() {
    let (wh, mut contract) = proper_instantiate();

    let (_, data) = sign_vaa_body(&wh, create_vaa_body());
    contract
        .submit_vaas(vec![data.clone()])
        .expect("failed to submit chain registration");

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted duplicate vaa");
    assert_eq!(
        "message already processed",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn no_quorum() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body());
    v.signatures.truncate(quorum - 1);

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed chain registration without a quorum of signatures");
    assert_eq!(
        "generic error: querier contract error: no quorum",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn bad_signature() {
    let (wh, mut contract) = proper_instantiate();
    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body());

    // Flip a bit in the first signature so it becomes invalid.
    v.signatures[0].signature[0] ^= 1;

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed chain registration with bad signature");
    assert_eq!(
        "generic error: querier contract error: failed to recover verifying key",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn bad_serialization() {
    let (wh, mut contract) = proper_instantiate();

    let (v, _) = sign_vaa_body(&wh, create_vaa_body());

    // Rather than using the wormhole wire format use cosmwasm json.
    let data = to_json_binary(&v).unwrap();

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed chain registration with bad serialization");
    assert_eq!(
        "unexpected end of input",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn non_chain_registration() {
    let (wh, mut contract) = proper_instantiate();

    let mut body = create_vaa_body();
    body.payload.action = Action::ContractUpgrade {
        new_contract: Address([0x2f; 32]),
    };
    let (_, data) = sign_vaa_body(&wh, body);

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully executed VAA with non-chain registration action");
    assert_eq!(
        "unsupported governance action",
        err.root_cause().to_string().to_lowercase()
    );
}
