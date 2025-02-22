mod helpers;

use accountant::state::{transfer, TokenAddress};
use cosmwasm_std::{from_json, to_json_binary, Binary, Event, Uint256};
use global_accountant::msg::{Observation, ObservationStatus, SubmitObservationResponse};
use helpers::*;
use serde_wormhole::RawMessage;
use wormhole_bindings::fake::WormholeKeeper;
use wormhole_sdk::{
    token::Message,
    vaa::{Body, Header, Vaa},
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

fn transfer_data_from_token_message<P>(msg: Message<P>) -> transfer::Data {
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

    let (wh, mut contract) = proper_instantiate();

    register_emitters(&wh, &mut contract, COUNT);

    let (vaas, payloads) = create_transfer_vaas(&wh, COUNT);

    let resp = contract.submit_vaas(payloads).unwrap();

    for v in vaas {
        let key = transfer::Key::new(
            v.emitter_chain.into(),
            TokenAddress::new(v.emitter_address.0),
            v.sequence,
        );
        let (_, body) = v.into();
        let digest = body.digest().unwrap().secp256k_hash;
        let data = transfer_data_from_token_message(body.payload);
        let tx = contract.query_transfer(key.clone()).unwrap();
        assert_eq!(data, tx.data);
        assert_eq!(&digest[..], &*tx.digest);
        resp.assert_event(
            &Event::new("wasm-Transfer")
                .add_attribute("key", serde_json_wasm::to_string(&key).unwrap())
                .add_attribute("data", serde_json_wasm::to_string(&data).unwrap()),
        );
    }
}

#[test]
fn invalid_emitter() {
    const COUNT: usize = 1;

    let (wh, mut contract) = proper_instantiate();

    let (_vaas, payloads) = create_transfer_vaas(&wh, COUNT);

    let err = contract
        .submit_vaas(payloads)
        .expect_err("successfully submitted VAA from invalid emitter");
    assert_eq!(
        "no registered emitter for chain any",
        err.root_cause().to_string().to_lowercase()
    );
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
    assert_eq!(
        "cannot burn wrapped tokens without an existing wrapped account",
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
    assert_eq!(
        "unknown tokenbridge payload",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn transfer_with_payload() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);
    let payload = [0x88; 17];
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
            payload: RawMessage::new(&payload[..]),
        },
    };

    let data = serde_wormhole::to_vec(&body).unwrap();

    let signatures = wh.sign(&data);
    let header = Header {
        version: 1,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let v = Vaa::from((header, body));

    let data = serde_wormhole::to_vec(&v).unwrap();
    let resp = contract.submit_vaas(vec![data.into()]).unwrap();

    let key = transfer::Key::new(
        v.emitter_chain.into(),
        TokenAddress::new(v.emitter_address.0),
        v.sequence,
    );
    let (_, body) = v.into();
    let digest = body.digest().unwrap().secp256k_hash;
    let data = transfer_data_from_token_message(body.payload);
    let tx = contract.query_transfer(key.clone()).unwrap();
    assert_eq!(data, tx.data);
    assert_eq!(&digest[..], &*tx.digest);
    resp.assert_event(
        &Event::new("wasm-Transfer")
            .add_attribute("key", serde_json_wasm::to_string(&key).unwrap())
            .add_attribute("data", serde_json_wasm::to_string(&data).unwrap()),
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

#[test]
fn reobservation() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 7);

    let (v, data) = sign_vaa_body(&wh, create_vaa_body(6));
    contract
        .submit_vaas(vec![data])
        .expect("failed to submit VAA");

    // Now try submitting the same transfer as an observation.  This can happen when a guardian
    // re-observes a tx.
    let o = Observation {
        tx_hash: vec![0x55u8; 20].into(),
        timestamp: v.timestamp,
        nonce: v.nonce,
        emitter_chain: v.emitter_chain.into(),
        emitter_address: v.emitter_address.0,
        sequence: v.sequence,
        consistency_level: v.consistency_level,
        payload: serde_wormhole::to_vec(&v.payload).unwrap().into(),
    };
    let key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);

    let obs = to_json_binary(&vec![o]).unwrap();
    let index = wh.guardian_set_index();
    let signatures = sign_observations(&wh, &obs);
    for s in signatures {
        let resp = contract.submit_observations(obs.clone(), index, s).unwrap();
        let mut responses: Vec<SubmitObservationResponse> = from_json(resp.data.unwrap()).unwrap();

        assert_eq!(1, responses.len());
        let d = responses.remove(0);
        assert_eq!(key, d.key);
        assert!(matches!(d.status, ObservationStatus::Committed));
    }
}

#[test]
fn digest_mismatch() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 7);

    let (v, data) = sign_vaa_body(&wh, create_vaa_body(6));
    contract
        .submit_vaas(vec![data])
        .expect("failed to submit VAA");

    // Now try submitting an observation with the same (chain, address, sequence) tuple but with
    // different details.
    let o = Observation {
        tx_hash: vec![0x55u8; 20].into(),
        timestamp: v.timestamp,
        nonce: v.nonce ^ u32::MAX,
        emitter_chain: v.emitter_chain.into(),
        emitter_address: v.emitter_address.0,
        sequence: v.sequence,
        consistency_level: v.consistency_level,
        payload: serde_wormhole::to_vec(&v.payload).unwrap().into(),
    };

    let key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
    let obs = to_json_binary(&vec![o]).unwrap();
    let index = wh.guardian_set_index();
    let signatures = sign_observations(&wh, &obs);
    for s in signatures {
        let resp = contract.submit_observations(obs.clone(), index, s).unwrap();
        let responses = from_json::<Vec<SubmitObservationResponse>>(&resp.data.unwrap()).unwrap();
        assert_eq!(key, responses[0].key);
        if let ObservationStatus::Error(ref err) = responses[0].status {
            assert!(err.contains("digest mismatch"));
        } else {
            panic!(
                "unexpected status for observation with mismatched digest: {:?}",
                responses[0].status
            );
        }
    }
}
