mod helpers;

use accountant::state::{account, Kind, Modification};
use cosmwasm_std::{Event, Uint256};
use global_accountant::msg::InstantiateMsg;
use helpers::*;

#[test]
fn simple_modify() {
    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };

    // Test modification via both instantiate + migrate channels.
    let instantiate_msgs = vec![
        None,
        Some(InstantiateMsg {
            modifications: vec![m.clone()],
        }),
    ];

    for instantiate_msg in instantiate_msgs {
        let (_wh, contract) = if instantiate_msg.is_some() {
            // pass modification via instantiate
            proper_instantiate_with(instantiate_msg.as_ref().unwrap())
            // there is no event returned from test instantiate so we pass on checking the event for now.
        } else {
            let (_wh, mut contract) = proper_instantiate();
            // pass modification via migrate after instantiate
            let resp = contract.modify_balance(m.clone()).unwrap();

            let evt = Event::new("wasm-Modification")
                .add_attribute("sequence", serde_json_wasm::to_string(&m.sequence).unwrap())
                .add_attribute("chain_id", serde_json_wasm::to_string(&m.chain_id).unwrap())
                .add_attribute(
                    "token_chain",
                    serde_json_wasm::to_string(&m.token_chain).unwrap(),
                )
                .add_attribute(
                    "token_address",
                    serde_json_wasm::to_string(&m.token_address).unwrap(),
                )
                .add_attribute("kind", serde_json_wasm::to_string(&m.kind).unwrap())
                .add_attribute("amount", serde_json_wasm::to_string(&m.amount).unwrap())
                .add_attribute("reason", serde_json_wasm::to_string(&m.reason).unwrap());

            resp.assert_event(&evt);
            (_wh, contract)
        };

        let actual = contract.query_modification(m.sequence).unwrap();
        assert_eq!(m.clone(), actual);

        let balance = contract
            .query_balance(account::Key::new(
                m.chain_id,
                m.token_chain,
                m.token_address,
            ))
            .unwrap();
        assert_eq!(m.amount, *balance);
    }
}

#[test]
fn duplicate_modify() {
    let (_wh, mut contract) = proper_instantiate();

    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };

    contract.modify_balance(m.clone()).unwrap();

    contract
        .modify_balance(m)
        .expect_err("successfully submitted duplicate modification");
}

#[test]
fn round_trip() {
    let (_wh, mut contract) = proper_instantiate();

    let mut m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    contract.modify_balance(m.clone()).unwrap();

    let actual = contract.query_modification(m.sequence).unwrap();
    assert_eq!(m, actual);

    // Now reverse the modification.
    m.sequence += 1;
    m.kind = Kind::Sub;
    m.reason = "reverse".into();

    contract.modify_balance(m.clone()).unwrap();

    let actual = contract.query_modification(m.sequence).unwrap();
    assert_eq!(m, actual);

    let balance = contract
        .query_balance(account::Key::new(
            m.chain_id,
            m.token_chain,
            m.token_address,
        ))
        .unwrap();
    assert_eq!(Uint256::zero(), *balance);
}

#[test]
fn repeat() {
    const ITERATIONS: usize = 10;

    let (_wh, mut contract) = proper_instantiate();

    let mut m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };

    for _ in 0..ITERATIONS {
        m.sequence += 1;

        contract.modify_balance(m.clone()).unwrap();

        let actual = contract.query_modification(m.sequence).unwrap();
        assert_eq!(m, actual);
    }

    let balance = contract
        .query_balance(account::Key::new(
            m.chain_id,
            m.token_chain,
            m.token_address,
        ))
        .unwrap();
    assert_eq!(m.amount * Uint256::from(ITERATIONS as u128), *balance);
}
