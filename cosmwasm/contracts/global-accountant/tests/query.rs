mod helpers;

use std::collections::BTreeMap;

use accountant::state::{
    account::{self, Balance},
    transfer, Kind, Modification, Transfer,
};
use cosmwasm_std::Uint256;
use global_accountant::msg::TransferStatus;
use helpers::*;
use wormhole_bindings::fake;
use wormhole_sdk::{token::Message, vaa::Body, Address, Amount};

fn create_accounts(wh: &fake::WormholeKeeper, contract: &mut Contract, count: usize) {
    let mut s = 0;
    for i in 0..count {
        for j in 0..count {
            s += 1;
            let m = Modification {
                sequence: s,
                chain_id: i as u16,
                token_chain: j as u16,
                token_address: [i as u8; 32].into(),
                kind: Kind::Add,
                amount: Uint256::from(j as u128),
                reason: "create_accounts".into(),
            };

            contract.modify_balance(m, wh).unwrap();
        }
    }
}

fn create_transfers(
    wh: &fake::WormholeKeeper,
    contract: &mut Contract,
    count: usize,
) -> Vec<Transfer> {
    let mut out = Vec::with_capacity(count);

    let mut vaas = Vec::with_capacity(count);
    for i in 0..count {
        let emitter_chain = i as u16;
        let emitter_address = [i as u8; 32];
        let sequence = i as u64;
        let token_chain = emitter_chain;
        let token_address = [i as u8; 32];
        let recipient_chain = emitter_chain + 1;
        let amount = Uint256::from(i as u128);

        let body: Body<Message> = Body {
            timestamp: i as u32,
            nonce: i as u32,
            emitter_chain: emitter_chain.into(),
            emitter_address: Address(emitter_address),
            sequence,
            consistency_level: 0,
            payload: Message::Transfer {
                amount: Amount(amount.to_be_bytes()),
                token_address: Address(token_address),
                token_chain: token_chain.into(),
                recipient: Address([i as u8; 32]),
                recipient_chain: recipient_chain.into(),
                fee: Amount(Uint256::zero().to_be_bytes()),
            },
        };

        let (_, data) = sign_vaa_body(wh, body);
        vaas.push(data);
        out.push(Transfer {
            key: transfer::Key::new(emitter_chain, emitter_address.into(), sequence),
            data: transfer::Data {
                amount,
                token_chain,
                token_address: token_address.into(),
                recipient_chain,
            },
        });
    }

    contract.submit_vaas(vaas).unwrap();
    out
}

pub fn create_modifications(
    wh: &fake::WormholeKeeper,
    contract: &mut Contract,
    count: usize,
) -> Vec<Modification> {
    let mut out = Vec::with_capacity(count);
    for i in 0..count {
        let m = Modification {
            sequence: i as u64,
            chain_id: i as u16,
            token_chain: i as u16,
            token_address: [i as u8; 32].into(),
            kind: Kind::Add,
            amount: Uint256::from(i as u128),
            reason: format!("{i}").as_str().into(),
        };

        contract.modify_balance(m.clone(), wh).unwrap();

        out.push(m);
    }

    out
}

#[test]
fn account_balance() {
    let count = 2;
    let (wh, mut contract) = proper_instantiate();
    create_accounts(&wh, &mut contract, count);

    for i in 0..count {
        for j in 0..count {
            let key = account::Key::new(i as u16, j as u16, [i as u8; 32].into());
            let balance = contract.query_balance(key).unwrap();
            assert_eq!(balance, Balance::new(Uint256::from(j as u128)))
        }
    }
}

#[test]
fn missing_account() {
    let count = 2;
    let (wh, mut contract) = proper_instantiate();
    create_accounts(&wh, &mut contract, count);

    let missing = account::Key::new(
        (count + 1) as u16,
        (count + 2) as u16,
        [(count + 3) as u8; 32].into(),
    );

    let err = contract
        .query_balance(missing)
        .expect_err("successfully queried missing account key");
    assert!(err.to_string().to_lowercase().contains("balance not found"));
}

#[test]
fn all_balances() {
    let count = 3;
    let (wh, mut contract) = proper_instantiate();
    create_accounts(&wh, &mut contract, count);

    let resp = contract.query_all_accounts(None, None).unwrap();
    let found = resp
        .accounts
        .into_iter()
        .map(|acc| (acc.key, acc.balance))
        .collect::<BTreeMap<_, _>>();
    assert_eq!(found.len(), count * count);

    for i in 0..count {
        for j in 0..count {
            let key = account::Key::new(i as u16, j as u16, [i as u8; 32].into());
            assert!(found.contains_key(&key));
        }
    }
}

#[test]
fn all_balances_sub_range() {
    let count = 3;
    let (wh, mut contract) = proper_instantiate();
    create_accounts(&wh, &mut contract, count);

    for i in 0..count {
        for j in 0..count {
            let max_limit = (count - i - 1) * count + (count - j - 1);
            for l in 1..=max_limit {
                let start_after = Some(account::Key::new(i as u16, j as u16, [i as u8; 32].into()));
                let limit = Some(l as u32);
                let resp = contract.query_all_accounts(start_after, limit).unwrap();
                let found = resp
                    .accounts
                    .into_iter()
                    .map(|acc| (acc.key, acc.balance))
                    .collect::<BTreeMap<_, _>>();
                assert_eq!(found.len(), l);

                let mut checked = 0;
                for y in j + 1..count {
                    if checked >= l {
                        break;
                    }

                    let key = account::Key::new(i as u16, y as u16, [i as u8; 32].into());
                    assert!(found.contains_key(&key));
                    checked += 1;
                }

                'outer: for x in i + 1..count {
                    for y in 0..count {
                        if checked >= l {
                            break 'outer;
                        }
                        let key = account::Key::new(x as u16, y as u16, [x as u8; 32].into());
                        assert!(found.contains_key(&key));
                        checked += 1;
                    }
                }
            }
        }
    }
}

#[test]
fn transfer_data() {
    let count = 2;
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, count);
    create_transfers(&wh, &mut contract, count);

    for i in 0..count {
        let expected = transfer::Data {
            amount: Uint256::from(i as u128),
            token_chain: i as u16,
            token_address: [i as u8; 32].into(),
            recipient_chain: (i + 1) as u16,
        };

        let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
        let actual = contract.query_transfer(key).unwrap();

        assert_eq!(expected, actual.data);
    }
}

#[test]
fn missing_transfer() {
    let count = 2;
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, count);
    create_transfers(&wh, &mut contract, count);

    let missing = transfer::Key::new(
        (count + 1) as u16,
        [(count + 2) as u8; 32].into(),
        (count + 3) as u64,
    );

    let err = contract
        .query_transfer(missing)
        .expect_err("successfully queried missing transfer key");
    assert!(err.to_string().to_lowercase().contains("not found"));
}

#[test]
fn all_transfer_data() {
    let count = 3;
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, count);
    let transfers = create_transfers(&wh, &mut contract, count);

    let resp = contract.query_all_transfers(None, None).unwrap();
    let found = resp
        .transfers
        .into_iter()
        .map(|(acc, _)| (acc.key, acc.data))
        .collect::<BTreeMap<_, _>>();
    assert_eq!(found.len(), count);

    for t in transfers {
        assert_eq!(found[&t.key], t.data);
    }
}

#[test]
fn batch_transfer_status() {
    let count = 3;
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, count);
    let transfers = create_transfers(&wh, &mut contract, count);

    let keys = transfers.iter().map(|t| &t.key).cloned().collect();
    let resp = contract.query_batch_transfer_status(keys).unwrap();

    for (tx, details) in transfers.into_iter().zip(resp.details) {
        assert_eq!(tx.key, details.key);
        match details.status {
            Some(TransferStatus::Committed { data, .. }) => assert_eq!(tx.data, data),
            s => panic!("unexpected transfer status: {s:?}"),
        }
    }
}

#[test]
fn all_transfer_data_sub_range() {
    let count = 5;
    let (wh, mut contract) = proper_instantiate();
    register_emitters(&wh, &mut contract, count);
    create_transfers(&wh, &mut contract, count);

    for i in 0..count {
        for l in 1..count - i {
            let start_after = Some(transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64));
            let limit = Some(l as u32);
            let resp = contract.query_all_transfers(start_after, limit).unwrap();
            let found = resp
                .transfers
                .into_iter()
                .map(|(acc, _)| (acc.key, acc.data))
                .collect::<BTreeMap<_, _>>();
            assert_eq!(found.len(), l);

            for x in i + 1..=i + l {
                let key = transfer::Key::new(x as u16, [x as u8; 32].into(), x as u64);
                assert!(found.contains_key(&key));
            }
        }
    }
}

#[test]
fn modification_data() {
    let count = 2;
    let (wh, mut contract) = proper_instantiate();
    let modifications = create_modifications(&wh, &mut contract, count);

    for m in modifications {
        let actual = contract.query_modification(m.sequence).unwrap();

        assert_eq!(m, actual);
    }
}

#[test]
fn missing_modification() {
    let count = 2;
    let (wh, mut contract) = proper_instantiate();
    create_modifications(&wh, &mut contract, count);

    let missing = (count + 1) as u64;

    let err = contract
        .query_modification(missing)
        .expect_err("successfully queried missing modification key");
    assert!(err.to_string().to_lowercase().contains("not found"));
}

#[test]
fn all_modification_data() {
    let count = 3;
    let (wh, mut contract) = proper_instantiate();
    let modifications = create_modifications(&wh, &mut contract, count);

    let resp = contract.query_all_modifications(None, None).unwrap();
    let found = resp
        .modifications
        .into_iter()
        .map(|m| (m.sequence, m))
        .collect::<BTreeMap<_, _>>();
    assert_eq!(found.len(), count);

    for m in modifications {
        assert_eq!(found[&m.sequence], m);
    }
}

#[test]
fn all_modification_data_sub_range() {
    let count = 5;
    let (wh, mut contract) = proper_instantiate();
    create_modifications(&wh, &mut contract, count);

    for i in 0..count {
        for l in 1..count - i {
            let start_after = Some(i as u64);
            let limit = Some(l as u32);
            let resp = contract
                .query_all_modifications(start_after, limit)
                .unwrap();
            let found = resp
                .modifications
                .into_iter()
                .map(|m| (m.sequence, m))
                .collect::<BTreeMap<_, _>>();
            assert_eq!(found.len(), l);

            for x in i + 1..=i + l {
                let key = x as u64;
                assert!(found.contains_key(&key));
            }
        }
    }
}
