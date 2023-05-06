mod helpers;

use cosmwasm_std::{Addr, Binary};

use cw_multi_test::{Executor};

use ibc_wormhole_translator::{
    msg::{ExecuteMsg},
};

use helpers::{instantiate_contracts, OWNER};

static GOV_VAA_TERRA2_CHANNEL_18: &str = "01000000000100f527ab22eba9cb80ea3a673d5ea9034d39a93a48c4c1e415b84e7c8a700b61ca2ce68ef28acf3c5055fa1f85d6881a927267e97f6b86fb38c56a5b3e0fb6f19a0100000000a5567d7d00010000000000000000000000000000000000000000000000000000000000000004ee8c114665f9261f2000000000000000000000000000000000000000000049626352656365697665720100120000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006368616e6e656c2d31380c20";

#[test]
fn submit_illformed_vaa() {
    let (mut router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

    let vaa = "0000000000000000000000000000000000000000000000000000000075757364";
    let vaa = hex::decode(vaa).unwrap();
    let vaa = Binary::from(vaa.clone());

    let execute_msg = ExecuteMsg::SubmitVaa {data: vaa};
    let err = router
        .execute_contract(
            Addr::unchecked(OWNER),
            ibc_wormhole_translator_contract_addr.clone(),    // clone since we'll use it again
            &execute_msg,
            &[],                              // funds
        )
        .expect_err("successfully submitted governance VAA");

    assert_eq!(
        "generic error: querier contract error: generic error: invalidvaa",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn submit_invalid_vaa2() {
    let (mut router, ibc_wormhole_translator_contract_addr) = instantiate_contracts();

    let vaa = hex::decode(GOV_VAA_TERRA2_CHANNEL_18).unwrap();
    let vaa = Binary::from(vaa.clone());

    let execute_msg = ExecuteMsg::SubmitVaa {data: vaa};
    let err = router
        .execute_contract(
            Addr::unchecked(OWNER),
            ibc_wormhole_translator_contract_addr.clone(),    // clone since we'll use it again
            &execute_msg,
            &[],                              // funds
        )
        .expect_err("successfully submitted governance VAA");

    assert_eq!(
        "generic error: this is not a valid module",
        err.root_cause().to_string().to_lowercase()
    );
}

#[test]
fn submit_governance_invalid_emitter_chain() {
}

#[test]
fn submit_governance_invalid_emitter_address() {
}

#[test]
fn submit_governance_invalid_signature() {
}

#[test]
fn submit_governance_invalid_module() {
}

#[test]
fn submit_governance_invalid_target_chain() {
}

#[test]
fn submit_governance_invalid_action() {
}

#[test]
fn submit_governance_new_chain() {
}

#[test]
fn submit_governance_update_chain() {
}

#[test]
fn submit_governance_already_executed() {
}
