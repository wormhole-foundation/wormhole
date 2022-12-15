mod helpers;

use accounting::state::{transfer, TokenAddress};
use cosmwasm_std::{to_binary, Binary, Event, Uint256};
use helpers::*;
use wormhole::{
    token::Message,
    vaa::{Body, Header, Vaa},
    Address, Amount, Chain,
};
use wormhole_bindings::fake::WormholeKeeper;

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

fn transfer_data_from_token_message(msg: Message) -> transfer::Data {
    match msg {
        Message::Transfer {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        }
        | Message::TransferWithPayload {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        } => transfer::Data {
            amount: Uint256::from_be_bytes(amount.0),
            token_address: TokenAddress::new(token_address.0),
            token_chain: token_chain.into(),
            recipient_chain: recipient_chain.into(),
        },
        _ => panic!("not a transfer payload"),
    }
}

#[test]
fn basic() {
    const COUNT: usize = 7;

    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());

    let (vaas, payloads) = create_transfer_vaas(&wh, COUNT);

    let resp = contract.submit_vaas(payloads).unwrap();

    for v in vaas {
        let key = transfer::Key::new(
            v.emitter_chain.into(),
            TokenAddress::new(v.emitter_address.0),
            v.sequence,
        );
        let data = transfer_data_from_token_message(v.payload);
        assert_eq!(data, contract.query_transfer(key.clone()).unwrap());
        resp.assert_event(
            &Event::new("wasm-CommitTransfer")
                .add_attribute("key", key.to_string())
                .add_attribute("amount", data.amount.to_string())
                .add_attribute("token_chain", data.token_chain.to_string())
                .add_attribute("token_address", data.token_address.to_string())
                .add_attribute("recipient_chain", data.recipient_chain.to_string()),
        );
    }
}

#[test]
fn invalid_transfer() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());

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
    contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA containing invalid transfer");
}

#[test]
fn no_quorum() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body(3));
    v.signatures.truncate(quorum - 1);

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA without a quorum of signatures");
}

#[test]
fn bad_serialization() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());

    let (v, _) = sign_vaa_body(&wh, create_vaa_body(3));

    // Rather than using the wormhole wire format use cosmwasm json.
    let data = to_binary(&v).unwrap();

    contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with bad serialization");
}

#[test]
fn bad_signature() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());
    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body(3));

    // Flip a bit in the first signature so it becomes invalid.
    v.signatures[0].signature[0] ^= 1;

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();
    contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with bad signature");
}

#[test]
fn non_transfer_message() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());
    let body = Body {
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
    contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with non-transfer message");
}

#[test]
fn transfer_with_payload() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());
    let body = Body {
        timestamp: 2,
        nonce: 2,
        emitter_chain: Chain::Ethereum,
        emitter_address: Address([2u8; 32]),
        sequence: 2,
        consistency_level: 32,
        payload: Message::TransferWithPayload {
            amount: Amount(Uint256::from(2u128).to_be_bytes()),
            token_address: Address([3u8; 32]),
            token_chain: Chain::Ethereum,
            recipient: Address([2u8; 32]),
            recipient_chain: Chain::Bsc,
            sender_address: Address([0u8; 32]),
        },
    };

    let payload = [0x88; 17];
    let mut data = serde_wormhole::to_vec(&body).unwrap();
    data.extend_from_slice(&payload);

    let signatures = wh.sign(&data);
    let header = Header {
        version: 1,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let v = Vaa::from((header, body));

    let mut data = serde_wormhole::to_vec(&v).unwrap();
    data.extend_from_slice(&payload);
    let resp = contract.submit_vaas(vec![data.into()]).unwrap();

    let key = transfer::Key::new(
        v.emitter_chain.into(),
        TokenAddress::new(v.emitter_address.0),
        v.sequence,
    );
    let data = transfer_data_from_token_message(v.payload);
    assert_eq!(data, contract.query_transfer(key.clone()).unwrap());
    resp.assert_event(
        &Event::new("wasm-CommitTransfer")
            .add_attribute("key", key.to_string())
            .add_attribute("amount", data.amount.to_string())
            .add_attribute("token_chain", data.token_chain.to_string())
            .add_attribute("token_address", data.token_address.to_string())
            .add_attribute("recipient_chain", data.recipient_chain.to_string()),
    );
}

#[test]
fn unsupported_version() {
    let (wh, mut contract) = proper_instantiate(Vec::new(), Vec::new(), Vec::new());

    let (mut v, _) = sign_vaa_body(&wh, create_vaa_body(6));
    v.version = 0;

    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    contract
        .submit_vaas(vec![data])
        .expect_err("successfully submitted VAA with unsupported version");
}
