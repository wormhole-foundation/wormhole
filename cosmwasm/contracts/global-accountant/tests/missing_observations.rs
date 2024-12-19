mod helpers;

use accountant::state::transfer;
use cosmwasm_std::{to_json_binary, Uint256};
use global_accountant::msg::Observation;
use helpers::*;
use wormhole_sdk::{token::Message, Address, Amount, Chain};

fn create_observation() -> Observation {
    let msg: Message = Message::Transfer {
        amount: Amount(Uint256::from(500u128).to_be_bytes()),
        token_address: Address([0x02; 32]),
        token_chain: Chain::Ethereum,
        recipient: Address([0x1c; 32]),
        recipient_chain: Chain::Solana,
        fee: Amount([0u8; 32]),
    };
    Observation {
        tx_hash: vec![
            0x60, 0x5a, 0x56, 0x46, 0x3a, 0x0d, 0x71, 0x8b, 0x92, 0xf7, 0xe0, 0x00, 0x31, 0x1b,
            0x63, 0xde, 0xb1, 0x50, 0xd6, 0x36, 0x66, 0x47, 0xef, 0x0b, 0x38, 0xd7, 0x7d, 0x60,
            0xf5, 0xc6, 0xc4, 0x32,
        ]
        .into(),
        timestamp: 0xcf863e0c,
        nonce: 0x7058f400,
        emitter_chain: 2,
        emitter_address: [2; 32],
        sequence: 0xcc0b5753769752a3,
        consistency_level: 200,
        payload: serde_wormhole::to_vec(&msg).map(From::from).unwrap(),
    }
}

#[test]
fn missing_observations() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);

    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let o = create_observation();
    let digest = o.digest().unwrap();
    let data = to_json_binary(&[o.clone()]).unwrap();
    let signatures = sign_observations(&wh, &data);

    // Don't submit enough signatures for the transfer to reach quorum.
    for s in &signatures[..quorum - 1] {
        contract
            .submit_observations(data.clone(), index, *s)
            .unwrap();
    }

    // The transfer should still be pending.
    let key = transfer::Key::new(o.emitter_chain, o.emitter_address.into(), o.sequence);
    let pending = contract.query_pending_transfer(key).unwrap();
    assert_eq!(&digest, pending[0].digest());

    for (i, s) in signatures.iter().enumerate() {
        let resp = contract.query_missing_observations(index, s.index).unwrap();
        if i < quorum - 1 {
            assert!(resp.missing.is_empty());
        } else {
            assert_eq!(resp.missing.len(), 1);

            let missing = resp.missing.first().unwrap();
            assert_eq!(missing.chain_id, o.emitter_chain);
            assert_eq!(missing.tx_hash, o.tx_hash);
        }
    }

    // Now submit one more signature so the transfer is committed.
    contract
        .submit_observations(data, index, signatures[quorum])
        .unwrap();

    // There should be no more missing observations.
    for s in signatures {
        let resp = contract.query_missing_observations(index, s.index).unwrap();
        assert!(resp.missing.is_empty());
    }
}

#[test]
fn different_observations() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);

    let index = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(index, contract.app().block_info().height)
        .unwrap() as usize;

    let first = create_observation();
    let first_data = to_json_binary(&[first.clone()]).unwrap();
    let first_signatures = sign_observations(&wh, &first_data);

    // Don't submit enough signatures for the transfer to reach quorum.
    for s in &first_signatures[..quorum - 1] {
        contract
            .submit_observations(first_data.clone(), index, *s)
            .unwrap();
    }

    // Create a new observation with a different tx hash and payload.
    let msg: Message = Message::Transfer {
        amount: Amount(Uint256::from(900u128).to_be_bytes()),
        token_address: Address([0x02; 32]),
        token_chain: Chain::Ethereum,
        recipient: Address([0x1c; 32]),
        recipient_chain: Chain::Algorand,
        fee: Amount([0u8; 32]),
    };
    let mut second = create_observation();
    second.tx_hash = vec![
        0x8e, 0x29, 0xe6, 0xbc, 0xc0, 0x48, 0x88, 0x75, 0x57, 0xeb, 0x50, 0x9e, 0xb8, 0x5a, 0x4d,
        0x96, 0x53, 0xc0, 0xd7, 0x0b, 0x2c, 0xcb, 0xf1, 0x7b, 0x4d, 0x7c, 0x6e, 0x9a, 0xa6, 0x5e,
        0x2f, 0x1f,
    ]
    .into();
    second.payload = serde_wormhole::to_vec(&msg).map(From::from).unwrap();
    let second_data = to_json_binary(&[second.clone()]).unwrap();
    let second_signatures = sign_observations(&wh, &second_data);

    // Submit a different set of signatures for the second observation.
    for s in second_signatures.iter().rev().take(quorum - 1) {
        contract
            .submit_observations(second_data.clone(), index, *s)
            .unwrap();
    }

    let num_signatures = second_signatures.len();
    for (i, s) in second_signatures.into_iter().enumerate() {
        let resp = contract.query_missing_observations(index, s.index).unwrap();
        if i < num_signatures - quorum + 1 {
            // We should be missing the second observation.
            assert_eq!(resp.missing.len(), 1);

            let missing = resp.missing.first().unwrap();
            assert_eq!(second.emitter_chain, missing.chain_id);
            assert_eq!(second.tx_hash, missing.tx_hash);
        } else if i >= quorum - 1 {
            // We should be missing the first observation.
            assert_eq!(resp.missing.len(), 1);

            let missing = resp.missing.first().unwrap();
            assert_eq!(first.emitter_chain, missing.chain_id);
            assert_eq!(first.tx_hash, missing.tx_hash);
        } else {
            // We shouldn't be missing any observations.
            assert!(resp.missing.is_empty());
        }
    }
}

#[test]
fn guardian_set_change() {
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, 3);

    let first_set = wh.guardian_set_index();
    let quorum = wh
        .calculate_quorum(first_set, contract.app().block_info().height)
        .unwrap() as usize;

    let o = create_observation();
    let data = to_json_binary(&[o.clone()]).unwrap();
    let signatures = sign_observations(&wh, &data);

    // Don't submit enough signatures for the transfer to reach quorum.
    for s in &signatures[..quorum - 1] {
        contract
            .submit_observations(data.clone(), first_set, *s)
            .unwrap();
    }

    // Update the guardian set index and submit a different set of signatures.
    let second_set = first_set + 1;
    wh.set_index(second_set);
    for s in signatures.iter().rev().take(quorum - 1) {
        contract
            .submit_observations(data.clone(), second_set, *s)
            .unwrap();
    }

    let num_signatures = signatures.len();
    for (i, s) in signatures.into_iter().enumerate() {
        let first_missing = contract
            .query_missing_observations(first_set, s.index)
            .unwrap()
            .missing;
        let second_missing = contract
            .query_missing_observations(second_set, s.index)
            .unwrap()
            .missing;

        if i < num_signatures - quorum + 1 {
            // We should be missing signatures for the new guardian set.
            assert!(first_missing.is_empty());

            assert_eq!(second_missing.len(), 1);

            let missing = second_missing.first().unwrap();
            assert_eq!(o.emitter_chain, missing.chain_id);
            assert_eq!(o.tx_hash, missing.tx_hash);
        } else if i >= quorum - 1 {
            // We should be missing signatures for the old guardian set.
            assert!(second_missing.is_empty());

            assert_eq!(first_missing.len(), 1);

            let missing = first_missing.first().unwrap();
            assert_eq!(o.emitter_chain, missing.chain_id);
            assert_eq!(o.tx_hash, missing.tx_hash);
        } else {
            // We shouldn't be missing signatures for either set.
            assert!(first_missing.is_empty());

            assert!(second_missing.is_empty());
        }
    }
}
