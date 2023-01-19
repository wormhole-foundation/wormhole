mod helpers;

use accounting::state::{account, transfer, Kind, Modification, TokenAddress};
use cosmwasm_std::{to_binary, Binary, Event, Uint256};
use cw_multi_test::AppResponse;
use helpers::*;
use wormchain_accounting::msg::Observation;
use wormhole::{
    token::Message,
    vaa::{Body, Header},
    Address, Amount,
};
use wormhole_bindings::fake;

fn set_up(count: usize) -> (Vec<Message>, Vec<Observation>) {
    let mut txs = Vec::with_capacity(count);
    let mut observations = Vec::with_capacity(count);
    for i in 0..count {
        let tx = Message::Transfer {
            amount: Amount(Uint256::from(500u128).to_be_bytes()),
            token_address: Address([(i + 1) as u8; 32]),
            token_chain: (i as u16).into(),
            recipient: Address([(i + 2) as u8; 32]),
            recipient_chain: ((i + 3) as u16).into(),
            fee: Amount([0u8; 32]),
        };
        let payload = serde_wormhole::to_vec(&tx).map(Binary::from).unwrap();
        txs.push(tx);
        observations.push(Observation {
            tx_hash: vec![(i + 4) as u8; 20].into(),
            timestamp: i as u32,
            nonce: i as u32,
            emitter_chain: i as u16,
            emitter_address: [i as u8; 32],
            sequence: i as u64,
            consistency_level: 0,
            payload,
        });
    }

    (txs, observations)
}

#[test]
fn batch() {
    const COUNT: usize = 5;

    let (txs, observations) = set_up(COUNT);
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, COUNT);

    let index = wh.guardian_set_index();

    let obs = to_binary(&observations).unwrap();
    let signatures = wh.sign(&obs);
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    for (i, s) in signatures.into_iter().enumerate() {
        if i < quorum {
            contract.submit_observations(obs.clone(), index, s).unwrap();

            // Once there is a quorum the pending transfers are removed.
            if i < quorum - 1 {
                for o in &observations {
                    let key =
                        transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
                    let data = contract.query_pending_transfer(key.clone()).unwrap();
                    assert_eq!(o, data[0].observation());

                    // Make sure the transfer hasn't yet been committed.
                    contract
                        .query_transfer(key)
                        .expect_err("transfer committed without quorum");
                }
            } else {
                for o in &observations {
                    let key =
                        transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
                    contract
                        .query_pending_transfer(key)
                        .expect_err("found pending transfer for observation with quorum");
                }
            }
        } else {
            contract
                .submit_observations(obs.clone(), index, s)
                .expect_err("successfully submitted observation for committed transfer");
        }
    }

    for (tx, o) in txs.into_iter().zip(observations) {
        let expected = if let Message::Transfer {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        } = tx
        {
            transfer::Data {
                amount: Uint256::new(amount.0),
                token_chain: token_chain.into(),
                token_address: TokenAddress::new(token_address.0),
                recipient_chain: recipient_chain.into(),
            }
        } else {
            panic!("unexpected tokenbridge payload");
        };

        let key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
        let actual = contract.query_transfer(key).unwrap();
        assert_eq!(expected, actual.data);
        assert_eq!(o.digest().unwrap(), actual.digest);

        let src = contract
            .query_balance(account::Key::new(
                o.emitter_chain,
                expected.token_chain,
                expected.token_address,
            ))
            .unwrap();

        assert_eq!(expected.amount, *src);

        let dst = contract
            .query_balance(account::Key::new(
                expected.recipient_chain,
                expected.token_chain,
                expected.token_address,
            ))
            .unwrap();

        assert_eq!(expected.amount, *dst);
    }
}

#[test]
fn duplicates() {
    const COUNT: usize = 5;

    let (txs, observations) = set_up(COUNT);
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, COUNT);
    let index = wh.guardian_set_index();

    let obs = to_binary(&observations).unwrap();
    let signatures = wh.sign(&obs);
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    for (i, s) in signatures.iter().take(quorum).cloned().enumerate() {
        contract.submit_observations(obs.clone(), index, s).unwrap();
        let err = contract
            .submit_observations(obs.clone(), index, s)
            .expect_err("successfully submitted duplicate observations");
        if i < quorum - 1 {
            // Sadly we can't match on the exact error type in an integration test because the
            // test frameworks converts it into a string before it reaches this point.
            assert!(format!("{err:#}").contains("duplicate signatures"));
        }
    }

    for (tx, o) in txs.into_iter().zip(observations) {
        let expected = if let Message::Transfer {
            amount,
            token_address,
            token_chain,
            recipient_chain,
            ..
        } = tx
        {
            transfer::Data {
                amount: Uint256::new(amount.0),
                token_chain: token_chain.into(),
                token_address: TokenAddress::new(token_address.0),
                recipient_chain: recipient_chain.into(),
            }
        } else {
            panic!("unexpected tokenbridge payload");
        };

        let key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
        let actual = contract.query_transfer(key).unwrap();
        assert_eq!(expected, actual.data);
        assert_eq!(o.digest().unwrap(), actual.digest);

        let src = contract
            .query_balance(account::Key::new(
                o.emitter_chain,
                expected.token_chain,
                expected.token_address,
            ))
            .unwrap();

        assert_eq!(expected.amount, *src);

        let dst = contract
            .query_balance(account::Key::new(
                expected.recipient_chain,
                expected.token_chain,
                expected.token_address,
            ))
            .unwrap();

        assert_eq!(expected.amount, *dst);
    }

    for s in signatures {
        contract
            .submit_observations(obs.clone(), index, s)
            .expect_err("successfully submitted observation for committed transfer");
    }
}

fn transfer_tokens(
    wh: &fake::WormholeKeeper,
    contract: &mut Contract,
    key: transfer::Key,
    msg: Message,
    index: u32,
    quorum: usize,
) -> anyhow::Result<(Observation, Vec<AppResponse>)> {
    let payload = serde_wormhole::to_vec(&msg).map(Binary::from).unwrap();
    let o = Observation {
        tx_hash: vec![0xd8u8; 20].into(),
        timestamp: 0xec8d03d6,
        nonce: 0x4343b191,
        emitter_chain: key.emitter_chain(),
        emitter_address: **key.emitter_address(),
        sequence: key.sequence(),
        consistency_level: 0,
        payload,
    };

    let obs = to_binary(&vec![o.clone()]).unwrap();
    let signatures = wh.sign(&obs);

    let responses = signatures
        .into_iter()
        .take(quorum)
        .map(|s| contract.submit_observations(obs.clone(), index, s))
        .collect::<anyhow::Result<Vec<_>>>()?;

    Ok((o, responses))
}

#[test]
fn round_trip() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 15);
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2u16.into();
    let recipient_chain = 14u16.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    let (o, _) = transfer_tokens(&wh, &mut contract, key.clone(), msg, index, quorum).unwrap();

    let expected = transfer::Data {
        amount: Uint256::new(amount.0),
        token_chain: token_chain.into(),
        token_address: TokenAddress::new(token_address.0),
        recipient_chain: recipient_chain.into(),
    };
    let actual = contract.query_transfer(key).unwrap();
    assert_eq!(expected, actual.data);
    assert_eq!(o.digest().unwrap(), actual.digest);

    // Now send the tokens back.
    let key = transfer::Key::new(
        recipient_chain.into(),
        [u16::from(recipient_chain) as u8; 32].into(),
        91156748,
    );
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xe4u8; 32]),
        recipient_chain: emitter_chain.into(),
        fee: Amount([0u8; 32]),
    };
    let (o, _) = transfer_tokens(&wh, &mut contract, key.clone(), msg, index, quorum).unwrap();

    let expected = transfer::Data {
        amount: Uint256::new(amount.0),
        token_chain: token_chain.into(),
        token_address: TokenAddress::new(token_address.0),
        recipient_chain: emitter_chain,
    };
    let actual = contract.query_transfer(key).unwrap();
    assert_eq!(expected, actual.data);
    assert_eq!(o.digest().unwrap(), actual.digest);

    // Now both balances should be zero.
    let src = contract
        .query_balance(account::Key::new(
            emitter_chain,
            token_chain.into(),
            expected.token_address,
        ))
        .unwrap();

    assert_eq!(Uint256::zero(), *src);

    let dst = contract
        .query_balance(account::Key::new(
            recipient_chain.into(),
            token_chain.into(),
            expected.token_address,
        ))
        .unwrap();

    assert_eq!(Uint256::zero(), *dst);
}

#[test]
fn missing_guardian_set() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    transfer_tokens(&wh, &mut contract, key, msg, index + 1, quorum)
        .expect_err("successfully submitted observations with invalid guardian set");
}

#[test]
fn expired_guardian_set() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let mut block = contract.app().block_info();

    let quorum = wh.calculate_quorum(index, block.height).unwrap() as usize;

    let emitter_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    // Mark the guardian set expired.
    wh.set_expiration(block.height);
    block.height += 1;
    contract.app_mut().set_block(block);

    transfer_tokens(&wh, &mut contract, key, msg, index, quorum)
        .expect_err("successfully submitted observations with expired guardian set");
}

#[test]
fn no_quorum() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let emitter_address = [emitter_chain as u8; 32];
    let sequence = 37;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, emitter_address.into(), sequence);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    transfer_tokens(
        &wh,
        &mut contract,
        key.clone(),
        msg.clone(),
        index,
        quorum - 1,
    )
    .unwrap();

    let data = contract.query_pending_transfer(key.clone()).unwrap();
    assert_eq!(emitter_chain, data[0].observation().emitter_chain);
    assert_eq!(emitter_address, data[0].observation().emitter_address);
    assert_eq!(sequence, data[0].observation().sequence);

    let actual = serde_wormhole::from_slice(&data[0].observation().payload).unwrap();
    assert_eq!(msg, actual);

    // Make sure the transfer hasn't yet been committed.
    contract
        .query_transfer(key)
        .expect_err("transfer committed without quorum");
}

#[test]
fn missing_wrapped_account() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 14;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 2.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    transfer_tokens(&wh, &mut contract, key, msg, index, quorum)
        .expect_err("successfully burned wrapped tokens without a wrapped amount");
}

#[test]
fn missing_native_account() {
    let emitter_chain = 14;
    let recipient_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = [0xccu8; 32];
    let token_chain = 2;

    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    // We need to set up a fake wrapped account so that the initial check succeeds.
    let m = to_binary(&Modification {
        sequence: 0,
        chain_id: emitter_chain,
        token_chain,
        token_address: token_address.into(),
        kind: Kind::Add,
        amount: Uint256::new(amount.0),
        reason: "fake wrapped balance for testing".into(),
    })
    .unwrap();
    let signatures = wh.sign(&m);
    contract.modify_balance(m, index, signatures).unwrap();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address: Address(token_address),
        token_chain: token_chain.into(),
        recipient: Address([0xb9u8; 32]),
        recipient_chain: recipient_chain.into(),
        fee: Amount([0u8; 32]),
    };

    transfer_tokens(&wh, &mut contract, key, msg, index, quorum)
        .expect_err("successfully unlocked native tokens without a native account");
}

#[test]
fn repeated() {
    const ITERATIONS: usize = 10;

    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let recipient_chain = 14;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = [0xccu8; 32];
    let token_chain = 2;

    let msg = Message::Transfer {
        amount,
        token_address: Address(token_address),
        token_chain: token_chain.into(),
        recipient: Address([0xb9u8; 32]),
        recipient_chain: recipient_chain.into(),
        fee: Amount([0u8; 32]),
    };

    for i in 0..ITERATIONS {
        let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), i as u64);
        transfer_tokens(&wh, &mut contract, key.clone(), msg.clone(), index, quorum).unwrap();
    }

    let expected = Uint256::new(amount.0) * Uint256::from(ITERATIONS as u128);
    let src = contract
        .query_balance(account::Key::new(
            emitter_chain,
            token_chain,
            token_address.into(),
        ))
        .unwrap();

    assert_eq!(expected, *src);

    let dst = contract
        .query_balance(account::Key::new(
            recipient_chain,
            token_chain,
            token_address.into(),
        ))
        .unwrap();

    assert_eq!(expected, *dst);
}

#[test]
fn wrapped_to_wrapped() {
    let emitter_chain = 14;
    let recipient_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = [0xccu8; 32];
    let token_chain = 5;

    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 15);
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    // We need an initial fake wrapped account.
    let m = to_binary(&Modification {
        sequence: 0,
        chain_id: emitter_chain,
        token_chain,
        token_address: token_address.into(),
        kind: Kind::Add,
        amount: Uint256::new(amount.0),
        reason: "fake wrapped balance for testing".into(),
    })
    .unwrap();
    let signatures = wh.sign(&m);
    contract.modify_balance(m, index, signatures).unwrap();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address: Address(token_address),
        token_chain: token_chain.into(),
        recipient: Address([0xb9u8; 32]),
        recipient_chain: recipient_chain.into(),
        fee: Amount([0u8; 32]),
    };

    let (o, _) = transfer_tokens(&wh, &mut contract, key.clone(), msg, index, quorum).unwrap();

    let expected = transfer::Data {
        amount: Uint256::new(amount.0),
        token_chain,
        token_address: TokenAddress::new(token_address),
        recipient_chain,
    };
    let actual = contract.query_transfer(key).unwrap();
    assert_eq!(expected, actual.data);
    assert_eq!(o.digest().unwrap(), actual.digest);

    let src = contract
        .query_balance(account::Key::new(
            emitter_chain,
            token_chain,
            token_address.into(),
        ))
        .unwrap();

    assert_eq!(Uint256::zero(), *src);

    let dst = contract
        .query_balance(account::Key::new(
            recipient_chain,
            token_chain,
            token_address.into(),
        ))
        .unwrap();

    assert_eq!(Uint256::new(amount.0), *dst);
}

#[test]
fn unknown_emitter() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 14;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 2.into();

    let key = transfer::Key::new(emitter_chain, [0xde; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    transfer_tokens(&wh, &mut contract, key, msg, index, quorum)
        .expect_err("successfully transfered tokens with an invalid emitter address");
}

#[test]
fn different_observations() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);
    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    // First submit some observations without enough signatures for quorum.
    let emitter_chain = 2;
    let fake_amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let fake_recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let fake = Message::Transfer {
        amount: fake_amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain: fake_recipient_chain,
        fee: Amount([0u8; 32]),
    };

    transfer_tokens(&wh, &mut contract, key.clone(), fake, index, quorum - 1).unwrap();

    // Make sure there is no committed transfer yet.
    contract
        .query_transfer(key.clone())
        .expect_err("committed transfer without quorum");

    // Now change the details of the transfer and resubmit with the same key.
    let real_amount = Amount(Uint256::from(200u128).to_be_bytes());
    let real_recipient_chain = 9.into();
    let real = Message::Transfer {
        amount: real_amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain: real_recipient_chain,
        fee: Amount([0u8; 32]),
    };

    let (o, _) = transfer_tokens(&wh, &mut contract, key.clone(), real, index, quorum).unwrap();

    contract
        .query_pending_transfer(key.clone())
        .expect_err("found pending transfer for observation with quorum");

    let expected = transfer::Data {
        amount: Uint256::new(real_amount.0),
        token_chain: token_chain.into(),
        token_address: TokenAddress::new(token_address.0),
        recipient_chain: real_recipient_chain.into(),
    };
    let actual = contract.query_transfer(key).unwrap();
    assert_eq!(expected, actual.data);
    assert_eq!(o.digest().unwrap(), actual.digest);

    let src = contract
        .query_balance(account::Key::new(
            emitter_chain,
            token_chain.into(),
            expected.token_address,
        ))
        .unwrap();

    assert_eq!(Uint256::new(real_amount.0), *src);

    let dst = contract
        .query_balance(account::Key::new(
            real_recipient_chain.into(),
            token_chain.into(),
            expected.token_address,
        ))
        .unwrap();

    assert_eq!(Uint256::new(real_amount.0), *dst);
}

#[test]
fn emit_event_with_quorum() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);

    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    let (o, responses) = transfer_tokens(&wh, &mut contract, key, msg, index, quorum).unwrap();

    let expected = Event::new("wasm-Observation")
        .add_attribute("tx_hash", serde_json_wasm::to_string(&o.tx_hash).unwrap())
        .add_attribute(
            "timestamp",
            serde_json_wasm::to_string(&o.timestamp).unwrap(),
        )
        .add_attribute("nonce", serde_json_wasm::to_string(&o.nonce).unwrap())
        .add_attribute(
            "emitter_chain",
            serde_json_wasm::to_string(&o.emitter_chain).unwrap(),
        )
        .add_attribute(
            "emitter_address",
            serde_json_wasm::to_string(&hex::encode(o.emitter_address)).unwrap(),
        )
        .add_attribute("sequence", serde_json_wasm::to_string(&o.sequence).unwrap())
        .add_attribute(
            "consistency_level",
            serde_json_wasm::to_string(&o.consistency_level).unwrap(),
        )
        .add_attribute("payload", serde_json_wasm::to_string(&o.payload).unwrap());

    assert_eq!(responses.len(), quorum);
    for (i, r) in responses.into_iter().enumerate() {
        if i < quorum - 1 {
            assert!(!r.has_event(&expected));
        } else {
            r.assert_event(&expected);
        }
    }
}

#[test]
fn duplicate_vaa() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);

    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    let (o, _) = transfer_tokens(&wh, &mut contract, key, msg, index, quorum).unwrap();

    // Now try to submit a VAA for this transfer.  This should fail since the transfer is already
    // processed.
    let body = Body {
        timestamp: o.timestamp,
        nonce: o.nonce,
        emitter_chain: o.emitter_chain.into(),
        emitter_address: Address(o.emitter_address),
        sequence: o.sequence,
        consistency_level: o.consistency_level,
        payload: (),
    };

    let mut body_data = serde_wormhole::to_vec(&body).unwrap();
    body_data.extend_from_slice(&o.payload);

    let mut data = serde_wormhole::to_vec(&Header {
        version: 1,
        guardian_set_index: index,
        signatures: wh.sign(&body_data),
    })
    .unwrap();
    data.extend_from_slice(&body_data);

    let err = contract
        .submit_vaas(vec![data.into()])
        .expect_err("successfully submitted duplicate VAA for committed transfer");
    assert!(format!("{err:#}").contains("message already processed"));
}

#[test]
fn digest_mismatch() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);

    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let emitter_chain = 2;
    let amount = Amount(Uint256::from(500u128).to_be_bytes());
    let token_address = Address([0xccu8; 32]);
    let token_chain = 2.into();
    let recipient_chain = 14.into();

    let key = transfer::Key::new(emitter_chain, [emitter_chain as u8; 32].into(), 37);
    let msg = Message::Transfer {
        amount,
        token_address,
        token_chain,
        recipient: Address([0xb9u8; 32]),
        recipient_chain,
        fee: Amount([0u8; 32]),
    };

    let (o, _) = transfer_tokens(&wh, &mut contract, key, msg, index, quorum).unwrap();

    // Now try submitting a VAA with the same (chain, address, sequence) tuple but with
    // different details.
    let body = Body {
        timestamp: o.timestamp,
        nonce: o.nonce ^ u32::MAX,
        emitter_chain: o.emitter_chain.into(),
        emitter_address: Address(o.emitter_address),
        sequence: o.sequence,
        consistency_level: o.consistency_level,
        payload: (),
    };

    let mut body_data = serde_wormhole::to_vec(&body).unwrap();
    body_data.extend_from_slice(&o.payload);

    let mut data = serde_wormhole::to_vec(&Header {
        version: 1,
        guardian_set_index: index,
        signatures: wh.sign(&body_data),
    })
    .unwrap();
    data.extend_from_slice(&body_data);

    let err = contract
        .submit_vaas(vec![data.into()])
        .expect_err("successfully submitted duplicate VAA for committed transfer");
    assert!(format!("{err:#}").contains("digest mismatch"));
}
