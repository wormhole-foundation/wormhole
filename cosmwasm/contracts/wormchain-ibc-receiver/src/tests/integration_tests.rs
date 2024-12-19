use crate::{
    contract::{execute, query},
    msg::{AllChannelChainsResponse, ExecuteMsg, QueryMsg},
    tests::test_utils::{create_gov_vaa_body, create_transfer_vaa_body, sign_vaa_body},
};
use anyhow::Error;
use cosmwasm_std::{
    from_json,
    testing::{mock_env, mock_info, MockApi, MockQuerier, MockStorage},
    to_json_binary, Binary, ContractResult, Deps, DepsMut, Empty, QuerierWrapper, SystemResult,
};
use wormhole_bindings::{fake::WormholeKeeper, WormholeQuery};
use wormhole_sdk::{
    ibc_receiver::{Action, GovernancePacket},
    vaa::Body,
    Chain, GOVERNANCE_EMITTER,
};

#[test]
pub fn add_channel_chain_happy_path() -> anyhow::Result<(), Error> {
    let wh = WormholeKeeper::new();

    let querier: MockQuerier<WormholeQuery> =
        MockQuerier::new(&[]).with_custom_handler(|q| match q {
            WormholeQuery::VerifyVaa { vaa } => {
                match WormholeKeeper::new().verify_vaa(&vaa.0, 0u64) {
                    Ok(_) => SystemResult::Ok(if let Ok(data) = to_json_binary(&Empty {}) {
                        ContractResult::Ok(data)
                    } else {
                        ContractResult::Err("Unable to convert to binary".to_string())
                    }),
                    Err(e) => SystemResult::Ok(ContractResult::Err(e.to_string())),
                }
            }
            _ => cosmwasm_std::SystemResult::Ok(cosmwasm_std::ContractResult::Ok(
                to_json_binary(&Empty {}).unwrap(),
            )),
        });

    let mut mut_deps = DepsMut {
        storage: &mut MockStorage::default(),
        api: &MockApi::default(),
        querier: QuerierWrapper::new(&querier),
    };
    let info = mock_info("sender", &[]);
    let env = mock_env();

    let add_sei_channel_body = create_gov_vaa_body(1, Chain::Sei, *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-0");
    let (_, add_sei_vaa_binary) = sign_vaa_body(wh.clone(), add_sei_channel_body);

    let submissions = execute(
        mut_deps.branch(),
        env.clone(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_sei_vaa_binary],
        },
    );

    assert!(
        submissions.is_ok(),
        "A proper UpdateChannelChain gov vaa should be accepted"
    );

    // create a readonly deps to use for querying the state
    let empty_mock_querier = MockQuerier::<Empty>::new(&[]);
    let readonly_deps = Deps {
        storage: mut_deps.storage,
        api: mut_deps.api,
        querier: QuerierWrapper::new(&empty_mock_querier),
    };

    let channel_binary = query(readonly_deps, env, QueryMsg::AllChannelChains {})?;
    let channel: AllChannelChainsResponse = from_json(&channel_binary)?;

    assert_eq!(channel.channels_chains.len(), 1);
    let channel_entry = channel.channels_chains.first().unwrap();
    assert_eq!(
        channel_entry.0,
        Binary::from(*b"channel-0"),
        "the stored channel for sei should initially be channel-0"
    );
    assert_eq!(
        channel_entry.1,
        Into::<u16>::into(Chain::Sei),
        "the stored channel should be for sei's chain id"
    );

    Ok(())
}

#[test]
pub fn add_channel_chain_happy_path_multiple() -> anyhow::Result<(), Error> {
    let wh = WormholeKeeper::new();

    let querier: MockQuerier<WormholeQuery> =
        MockQuerier::new(&[]).with_custom_handler(|q| match q {
            WormholeQuery::VerifyVaa { vaa } => {
                match WormholeKeeper::new().verify_vaa(&vaa.0, 0u64) {
                    Ok(_) => SystemResult::Ok(if let Ok(data) = to_json_binary(&Empty {}) {
                        ContractResult::Ok(data)
                    } else {
                        ContractResult::Err("Unable to convert to binary".to_string())
                    }),
                    Err(e) => SystemResult::Ok(ContractResult::Err(e.to_string())),
                }
            }
            _ => cosmwasm_std::SystemResult::Ok(cosmwasm_std::ContractResult::Ok(
                to_json_binary(&Empty {}).unwrap(),
            )),
        });

    let mut mut_deps = DepsMut {
        storage: &mut MockStorage::default(),
        api: &MockApi::default(),
        querier: QuerierWrapper::new(&querier),
    };
    let info = mock_info("sender", &[]);

    let add_inj_channel_body = create_gov_vaa_body(2, Chain::Injective, *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-1");
    let (_, add_inj_vaa_bin) = sign_vaa_body(wh.clone(), add_inj_channel_body);
    let add_sei_channel_body = create_gov_vaa_body(3, Chain::Sei, *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-2");
    let (_, add_sei_vaa_binary) = sign_vaa_body(wh.clone(), add_sei_channel_body);

    // add a channel for injective and update the channel set for sei
    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_sei_vaa_binary, add_inj_vaa_bin],
        },
    );

    assert!(
        submissions.is_ok(),
        "A pair of proper UpdateChannelChain gov vaas should be accepted"
    );

    // create a readonly deps to use for querying the state
    let empty_mock_querier = MockQuerier::<Empty>::new(&[]);
    let readonly_deps = Deps {
        storage: mut_deps.storage,
        api: mut_deps.api,
        querier: QuerierWrapper::new(&empty_mock_querier),
    };

    // refetch all the channels that are in state
    let channel_binary = query(readonly_deps, mock_env(), QueryMsg::AllChannelChains {})?;
    let AllChannelChainsResponse {
        channels_chains: mut channels,
    }: AllChannelChainsResponse = from_json(&channel_binary)?;

    channels.sort_by(|(_, a_chain_id), (_, b_chain_id)| a_chain_id.cmp(b_chain_id));

    assert_eq!(channels.len(), 2);

    let channel_entry = channels.first().unwrap();
    assert_eq!(
        channel_entry.0,
        Binary::from(*b"channel-1"),
        "the stored channel should be channel-1 "
    );
    assert_eq!(
        channel_entry.1,
        Into::<u16>::into(Chain::Injective),
        "the stored channel should be for injective's chain id"
    );

    let channel_entry = channels.last().unwrap();
    assert_eq!(
        channel_entry.0,
        Binary::from(*b"channel-2"),
        "the stored channel should be channel-2"
    );
    assert_eq!(
        channel_entry.1,
        Into::<u16>::into(Chain::Sei),
        "the stored channel should be for sei's chain id"
    );

    Ok(())
}

#[test]
pub fn reject_invalid_add_channel_chain_vaas() {
    let wh = WormholeKeeper::new();

    let querier: MockQuerier<WormholeQuery> =
        MockQuerier::new(&[]).with_custom_handler(|q| match q {
            WormholeQuery::VerifyVaa { vaa } => {
                match WormholeKeeper::new().verify_vaa(&vaa.0, 0u64) {
                    Ok(_) => SystemResult::Ok(if let Ok(data) = to_json_binary(&Empty {}) {
                        ContractResult::Ok(data)
                    } else {
                        ContractResult::Err("Unable to convert to binary".to_string())
                    }),
                    Err(e) => SystemResult::Ok(ContractResult::Err(e.to_string())),
                }
            }
            _ => cosmwasm_std::SystemResult::Ok(cosmwasm_std::ContractResult::Ok(
                to_json_binary(&Empty {}).unwrap(),
            )),
        });

    let mut mut_deps = DepsMut {
        storage: &mut MockStorage::default(),
        api: &MockApi::default(),
        querier: QuerierWrapper::new(&querier),
    };
    let info = mock_info("sender", &[]);

    let add_channel_body = create_gov_vaa_body(1, Chain::Wormchain, *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-0");
    let (_, add_vaa_binary) = sign_vaa_body(wh.clone(), add_channel_body);

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_vaa_binary],
        },
    );

    assert!(
        submissions.is_err(),
        "Cannot add a channel from Gateway to Gateway"
    );

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![Binary::from(vec![0u8; 32])],
        },
    );

    assert!(
        submissions.is_err(),
        "VAA should be rejected if it cannot be parsed because it's too short"
    );

    let add_channel_body = create_transfer_vaa_body(1);
    let (_, add_vaa_binary) = sign_vaa_body(wh.clone(), add_channel_body);

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_vaa_binary],
        },
    );

    assert!(submissions.is_err(), "Can only execute governance vaas");

    let add_channel_body = create_gov_vaa_body(1, Chain::Osmosis, *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-0");
    let (_, add_vaa_binary) = sign_vaa_body(wh.clone(), add_channel_body);

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_vaa_binary],
        },
    );

    assert!(
        submissions.is_ok(),
        "Can add a channel from Osmosis to Gateway"
    );

    let add_channel_body: Body<GovernancePacket> = Body {
        timestamp: 1u32,
        nonce: 1u32,
        emitter_chain: Chain::Solana,
        emitter_address: GOVERNANCE_EMITTER,
        sequence: 1u64,
        consistency_level: 0,
        payload: GovernancePacket {
            chain: Chain::Osmosis,
            action: Action::UpdateChannelChain {
                channel_id: *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-0",
                chain_id: Chain::CosmosHub,
            },
        },
    };
    let (_, add_vaa_binary) = sign_vaa_body(wh.clone(), add_channel_body);

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_vaa_binary],
        },
    );

    assert!(
        submissions.is_err(),
        "Cannot add a update a chain besides Gateway"
    );
}

#[test]
pub fn reject_replayed_add_channel_chain_vaas() {
    let wh = WormholeKeeper::new();

    let querier: MockQuerier<WormholeQuery> =
        MockQuerier::new(&[]).with_custom_handler(|q| match q {
            WormholeQuery::VerifyVaa { vaa } => {
                match WormholeKeeper::new().verify_vaa(&vaa.0, 0u64) {
                    Ok(_) => SystemResult::Ok(if let Ok(data) = to_json_binary(&Empty {}) {
                        ContractResult::Ok(data)
                    } else {
                        ContractResult::Err("Unable to convert to binary".to_string())
                    }),
                    Err(e) => SystemResult::Ok(ContractResult::Err(e.to_string())),
                }
            }
            _ => cosmwasm_std::SystemResult::Ok(cosmwasm_std::ContractResult::Ok(
                to_json_binary(&Empty {}).unwrap(),
            )),
        });

    let mut mut_deps = DepsMut {
        storage: &mut MockStorage::default(),
        api: &MockApi::default(),
        querier: QuerierWrapper::new(&querier),
    };
    let info = mock_info("sender", &[]);

    let add_channel_body = create_gov_vaa_body(1, Chain::Osmosis, *b"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00channel-0");
    let (_, add_vaa_binary) = sign_vaa_body(wh.clone(), add_channel_body);

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_vaa_binary.clone()],
        },
    );

    assert!(
        submissions.is_ok(),
        "Can add a channel from Osmosis to Gateway"
    );

    let submissions = execute(
        mut_deps.branch(),
        mock_env(),
        info.clone(),
        ExecuteMsg::SubmitUpdateChannelChain {
            vaas: vec![add_vaa_binary],
        },
    );

    assert!(submissions.is_err(), "Cannot replay the same VAA");
}
