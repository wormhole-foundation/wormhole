use std::marker::PhantomData;

use crate::contract::submit_vaas;
use anyhow::Error;
use cosmwasm_std::{
    testing::{mock_info, MockApi, MockQuerier, MockStorage},
    to_binary, Binary, ContractResult, Empty, OwnedDeps, SystemResult, Uint256,
};
use serde::Serialize;
use wormhole_bindings::{fake::WormholeKeeper, WormholeQuery};
use wormhole_sdk::{
    token::Message,
    vaa::{Body, Header, Vaa},
    Address, Amount,
};

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

pub fn sign_vaa_body<P: Serialize>(wh: WormholeKeeper, body: Body<P>) -> (Vaa<P>, Binary) {
    let data = serde_wormhole::to_vec(&body).unwrap();
    let signatures = WormholeKeeper::new().sign(&data);

    let header = Header {
        version: 1,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let v = (header, body).into();
    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    (v, data)
}

fn mock_wormhole_deps(
    wh: &WormholeKeeper,
) -> OwnedDeps<cosmwasm_std::MemoryStorage, MockApi, MockQuerier<WormholeQuery>, WormholeQuery> {
    let wh = WormholeKeeper::new();

    let querier: MockQuerier<WormholeQuery> =
        MockQuerier::new(&[]).with_custom_handler(move |q| match q {
            WormholeQuery::VerifyVaa { vaa } => match wh.verify_vaa(&vaa.0, 0u64) {
                Ok(_) => SystemResult::Ok(if let Ok(data) = to_binary(&Empty {}) {
                    ContractResult::Ok(data)
                } else {
                    ContractResult::Err("Unable to convert to binary".to_string())
                }),
                Err(e) => SystemResult::Ok(ContractResult::Err(e.to_string())),
            },
            _ => cosmwasm_std::SystemResult::Ok(cosmwasm_std::ContractResult::Ok(
                to_binary(&Empty {}).unwrap(),
            )),
        });

    OwnedDeps {
        storage: MockStorage::default(),
        api: MockApi::default(),
        querier,
        custom_query_type: PhantomData::<WormholeQuery>,
    }
}

#[test]
pub fn receiver_test() -> anyhow::Result<(), Error> {
    let wh = WormholeKeeper::new();
    let mut wormhole_deps = mock_wormhole_deps(&wh);
    let mut_deps = wormhole_deps.as_mut();

    let info = mock_info("sender", &[]);

    let vaa_body = create_vaa_body(1);
    let (signed_vaa, vaa_bin) = sign_vaa_body(wh.clone(), vaa_body);

    let submissions = submit_vaas(mut_deps, info, vec![vaa_bin]);

    println!("{:?}", submissions);

    assert!(
        submissions.is_err(),
        "The supplied vaa is not a governance VAA and should fail to be accepted"
    );

    Ok(())
}
