mod helpers;

use accounting::state::{account, Kind, Modification};
use cosmwasm_std::{to_binary, Event, Uint256};
use helpers::*;

#[test]
fn simple_modify() {
    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    let modification = to_binary(&m).unwrap();

    let signatures = wh.sign(&modification);
    let resp = contract
        .modify_balance(modification, index, signatures)
        .unwrap();

    let evt = Event::new("wasm-ModifyBalance")
        .add_attribute("sequence", m.sequence.to_string())
        .add_attribute("chain_id", m.chain_id.to_string())
        .add_attribute("token_chain", m.token_chain.to_string())
        .add_attribute("token_address", m.token_address.to_string())
        .add_attribute("kind", m.kind.to_string())
        .add_attribute("amount", m.amount)
        .add_attribute("reason", m.reason.clone());

    resp.assert_event(&evt);

    let actual = contract.query_modification(m.sequence).unwrap();
    assert_eq!(m, actual);

    let balance = contract
        .query_balance(account::Key::new(
            m.chain_id,
            m.token_chain,
            m.token_address,
        ))
        .unwrap();
    assert_eq!(m.amount, *balance);
}

#[test]
fn duplicate_modify() {
    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    let modification = to_binary(&m).unwrap();

    let signatures = wh.sign(&modification);
    contract
        .modify_balance(modification.clone(), index, signatures.clone())
        .unwrap();

    contract
        .modify_balance(modification, index, signatures)
        .expect_err("successfully submitted duplicate modification");
}

#[test]
fn round_trip() {
    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
    let mut m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    let modification = to_binary(&m).unwrap();

    let signatures = wh.sign(&modification);
    contract
        .modify_balance(modification, index, signatures)
        .unwrap();

    let actual = contract.query_modification(m.sequence).unwrap();
    assert_eq!(m, actual);

    // Now reverse the modification.
    m.sequence += 1;
    m.kind = Kind::Sub;
    m.reason = "reverse".into();

    let modification = to_binary(&m).unwrap();

    let signatures = wh.sign(&modification);
    contract
        .modify_balance(modification, index, signatures)
        .unwrap();

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
fn missing_guardian_set() {
    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    let modification = to_binary(&m).unwrap();

    let signatures = wh.sign(&modification);
    contract
        .modify_balance(modification, index + 1, signatures)
        .expect_err("successfully modified balance with invalid guardian set");
}

#[test]
fn expired_guardian_set() {
    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
    let mut block = contract.app().block_info();
    wh.set_expiration(block.height);
    block.height += 1;
    contract.app_mut().set_block(block);

    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    let modification = to_binary(&m).unwrap();

    let signatures = wh.sign(&modification);
    contract
        .modify_balance(modification, index, signatures)
        .expect_err("successfully modified balance with expired guardian set");
}

#[test]
fn no_quorum() {
    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
    let m = Modification {
        sequence: 0,
        chain_id: 1,
        token_chain: 1,
        token_address: [0x7c; 32].into(),
        kind: Kind::Add,
        amount: Uint256::from(300u128),
        reason: "test".into(),
    };
    let modification = to_binary(&m).unwrap();

    let mut signatures = wh.sign(&modification);
    let newlen = wh
        .calculate_quorum(0, contract.app().block_info().height)
        .map(|q| (q - 1) as usize)
        .unwrap();
    signatures.truncate(newlen);

    contract
        .modify_balance(modification, index, signatures)
        .expect_err("successfully submitted modification without quorum");
}

#[test]
fn repeat() {
    const ITERATIONS: usize = 10;

    let (wh, mut contract) = proper_instantiate();

    let index = wh.guardian_set_index();
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

        let modification = to_binary(&m).unwrap();

        let signatures = wh.sign(&modification);
        contract
            .modify_balance(modification, index, signatures)
            .unwrap();

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
