mod helpers;

use accountant::state::transfer;
use cosmwasm_std::{to_json_binary, Binary, Uint256};
use cw_multi_test::AppResponse;
use helpers::*;
use ntt_global_accountant::msg::Observation;
use wormhole_bindings::fake;
use wormhole_sdk::{token::Message, Address, Amount};

fn _set_up(count: usize) -> (Vec<Message>, Vec<Observation>) {
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

// TODO: port batch test

// TODO: port duplicates test

fn transfer_tokens(
    wh: &fake::WormholeKeeper,
    contract: &mut Contract,
    key: transfer::Key,
    msg: Message,
    index: u32,
    num_signatures: usize,
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

    let obs = to_json_binary(&vec![o.clone()]).unwrap();
    let signatures = sign_observations(wh, &obs);

    let responses = signatures
        .into_iter()
        .take(num_signatures)
        .map(|s| contract.submit_observations(obs.clone(), index, s))
        .collect::<anyhow::Result<Vec<_>>>()?;

    Ok((o, responses))
}

// TODO: port round_trip test

#[test]
fn missing_guardian_set() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let num_guardians = wh.num_guardians();

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

    let err = transfer_tokens(&wh, &mut contract, key, msg, index + 1, num_guardians)
        .expect_err("successfully submitted observations with invalid guardian set");
    assert_eq!(
        "generic error: querier contract error: invalid guardian set",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn expired_guardian_set() {
    let (wh, mut contract) = proper_instantiate();
    let index = wh.guardian_set_index();
    let mut block = contract.app().block_info();

    let num_guardians = wh.num_guardians();

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

    let err = transfer_tokens(&wh, &mut contract, key, msg, index, num_guardians)
        .expect_err("successfully submitted observations with expired guardian set");
    assert_eq!(
        "generic error: querier contract error: guardian set expired",
        err.root_cause().to_string().to_lowercase()
    );
}

// TODO: port no_quorum test

// TODO: port missing_wrapped_account test

// TODO: port missing_native_account test

// TODO: port repeated test

// TODO: port wrapped_to_wrapped test

// TODO: port unknown_emitter test

// TODO: port different_observations test

// TODO: port emit_event_with_quorum test

// TODO: port duplicate_vaa test

// TODO: port digest_mismatch test
