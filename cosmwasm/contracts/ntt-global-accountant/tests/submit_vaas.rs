mod helpers;

use cosmwasm_std::{to_json_binary, Binary, Uint256};
use helpers::*;

use wormhole_bindings::fake::WormholeKeeper;
use wormhole_sdk::{
    token::Message,
    vaa::{Body, Vaa},
    Address, Amount, Chain,
};

fn create_transfer_vaas(wh: &WormholeKeeper, count: usize) -> (Vec<Vaa<Message>>, Vec<Binary>) {
    let mut vaas = Vec::with_capacity(count);
    let mut payloads = Vec::with_capacity(count);

    for i in 0..count {
        let (v, data) = sign_vaa_body(wh, create_vaa_body(i));
        vaas.push(v);
        payloads.push(data);
    }

    (vaas, payloads)
}

fn create_vaa_body(i: usize) -> Body<Message> {
    Body {
        timestamp: i as u32,
        nonce: i as u32,
        emitter_chain: (i as u16).into(),
        emitter_address: Address([(i as u8); 32]),
        sequence: i as u64,
        consistency_level: 32,
        payload: Message::Transfer {
            amount: Amount(Uint256::from(i as u128).to_be_bytes()),
            token_address: Address([(i + 1) as u8; 32]),
            token_chain: (i as u16).into(),
            recipient: Address([i as u8; 32]),
            recipient_chain: ((i + 2) as u16).into(),
            fee: Amount([0u8; 32]),
        },
    }
}

// TODO: port basic test

#[test]
fn invalid_emitter() {
    const COUNT: usize = 1;

    let (wh, mut contract) = proper_instantiate();

    let (_vaas, payloads) = create_transfer_vaas(&wh, COUNT);

    let err = contract
        .submit_vaas(payloads)
        .expect_err("successfully submitted VAA from invalid emitter");
    // TODO: fix
    assert_eq!("unsupported NTT action", err.root_cause().to_string());
}

#[test]
fn invalid_transfer() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 4);

    let mut body = create_vaa_body(1);
    match body.payload {
        Message::Transfer {
            ref mut token_chain,
            recipient_chain,
            ..
        }
        | Message::TransferWithPayload {
            ref mut token_chain,
            recipient_chain,
            ..
        } => *token_chain = recipient_chain,
        _ => panic!("not a transfer payload"),
    }

    let (_, data) = sign_vaa_body(&wh, body);
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA containing invalid transfer");
    // TODO: fix
    assert_eq!(
        "failed to fill whole buffer",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn no_quorum() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 4);
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body(3));
    v.signatures.truncate(quorum - 1);

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA without a quorum of signatures");
    assert_eq!(
        "generic error: querier contract error: no quorum",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn bad_serialization() {
    let (wh, mut contract) = proper_instantiate();

    let (v, _) = sign_vaa_body(&wh, create_vaa_body(3));

    // Rather than using the wormhole wire format use cosmwasm json.
    let data = to_json_binary(&v).unwrap();

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with bad serialization");
    assert!(err
        .root_cause()
        .to_string()
        .to_lowercase()
        .contains("unexpected end of input"));
}

#[test]
fn bad_signature() {
    let (wh, mut contract) = proper_instantiate();
    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body(3));

    // Flip a bit in the first signature so it becomes invalid.
    v.signatures[0].signature[0] ^= 1;

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with bad signature");
    assert_eq!(
        "generic error: querier contract error: failed to verify signature",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn non_transfer_message() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 4);
    let body: Body<Message> = Body {
        timestamp: 2,
        nonce: 2,
        emitter_chain: Chain::Ethereum,
        emitter_address: Address([2u8; 32]),
        sequence: 2,
        consistency_level: 32,
        payload: Message::AssetMeta {
            token_address: Address([
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
                0xbe, 0xef, 0xfa, 0xce,
            ]),
            token_chain: Chain::Ethereum,
            decimals: 12,
            symbol: "BEEF".into(),
            name: "Beef face Token".into(),
        },
    };

    let (_, data) = sign_vaa_body(&wh, body);
    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with non-transfer message");
    // TODO: fix, this is currently registering a relayer and then passing a non delivery VAA
    assert_eq!(
        "payloadmismatch",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn unsupported_version() {
    let (wh, mut contract) = proper_instantiate();

    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body(6));
    v.version = 0;

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    let err = contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with unsupported version");
    assert_eq!(
        "unsupported vaa version",
        err.root_cause().to_string().to_lowercase()
    );
}

// TODO: port reobservation test

// TODO: port digest_mismatch test
