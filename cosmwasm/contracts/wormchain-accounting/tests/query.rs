mod helpers;

use std::collections::BTreeMap;

use accounting::state::{
    account::{self, Balance},
    transfer, Kind, Modification,
};
use cosmwasm_std::Uint256;
use helpers::*;
use wormhole_bindings::fake;

fn set_up(count: usize) -> (fake::WormholeKeeper, Contract) {
    let accounts = create_accounts(count);
    let transfers = create_transfers(count);
    let modifications = create_modifications(count);

    proper_instantiate(accounts, transfers, modifications)
}

#[test]
fn account_balance() {
    let count = 2;
    let (_, contract) = set_up(count);

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
    let (_, contract) = set_up(count);

    let missing = account::Key::new(
        (count + 1) as u16,
        (count + 2) as u16,
        [(count + 3) as u8; 32].into(),
    );

    contract
        .query_balance(missing)
        .expect_err("successfully queried missing account key");
}

#[test]
fn all_balances() {
    let count = 3;
    let (_, contract) = set_up(count);

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
    let (_, contract) = set_up(count);

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
    let (_, contract) = set_up(count);

    for i in 0..count {
        let expected = transfer::Data {
            amount: Uint256::from(i as u128),
            token_chain: i as u16,
            token_address: [i as u8; 32].into(),
            recipient_chain: i as u16,
        };

        let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
        let actual = contract.query_transfer(key).unwrap();

        assert_eq!(expected, actual);
    }
}

#[test]
fn missing_transfer() {
    let count = 2;
    let (_, contract) = set_up(count);

    let missing = transfer::Key::new(
        (count + 1) as u16,
        [(count + 2) as u8; 32].into(),
        (count + 3) as u64,
    );

    contract
        .query_transfer(missing)
        .expect_err("successfully queried missing transfer key");
}

#[test]
fn all_transfer_data() {
    let count = 3;
    let (_, contract) = set_up(count);

    let resp = contract.query_all_transfers(None, None).unwrap();
    let found = resp
        .transfers
        .into_iter()
        .map(|acc| (acc.key, acc.data))
        .collect::<BTreeMap<_, _>>();
    assert_eq!(found.len(), count);

    for i in 0..count {
        let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
        assert!(found.contains_key(&key));
    }
}

#[test]
fn all_transfer_data_sub_range() {
    let count = 5;
    let (_, contract) = set_up(count);

    for i in 0..count {
        for l in 1..count - i {
            let start_after = Some(transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64));
            let limit = Some(l as u32);
            let resp = contract.query_all_transfers(start_after, limit).unwrap();
            let found = resp
                .transfers
                .into_iter()
                .map(|acc| (acc.key, acc.data))
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
    let (_, contract) = set_up(count);

    for i in 0..count {
        let expected = Modification {
            sequence: i as u64,
            chain_id: i as u16,
            token_chain: i as u16,
            token_address: [i as u8; 32].into(),
            kind: if i % 2 == 0 { Kind::Add } else { Kind::Sub },
            amount: Uint256::from(i as u128),
            reason: format!("{i}"),
        };

        let key = i as u64;
        let actual = contract.query_modification(key).unwrap();

        assert_eq!(expected, actual);
    }
}

#[test]
fn missing_modification() {
    let count = 2;
    let (_, contract) = set_up(count);

    let missing = (count + 1) as u64;

    contract
        .query_modification(missing)
        .expect_err("successfully queried missing modification key");
}

#[test]
fn all_modification_data() {
    let count = 3;
    let (_, contract) = set_up(count);

    let resp = contract.query_all_modifications(None, None).unwrap();
    let found = resp
        .modifications
        .into_iter()
        .map(|m| (m.sequence, m))
        .collect::<BTreeMap<_, _>>();
    assert_eq!(found.len(), count);

    for i in 0..count {
        let key = i as u64;
        assert!(found.contains_key(&key));
    }
}

#[test]
fn all_modification_data_sub_range() {
    let count = 5;
    let (_, contract) = set_up(count);

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
