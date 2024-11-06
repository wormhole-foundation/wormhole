use cosmwasm_std::{Binary, Uint256};
use serde::Serialize;
use wormhole_bindings::fake::WormholeKeeper;
use wormhole_sdk::{
    ibc_receiver::{Action, GovernancePacket},
    token::Message,
    vaa::{Body, Header, Vaa},
    Address, Amount, Chain, GOVERNANCE_EMITTER,
};

pub fn create_transfer_vaa_body(i: usize) -> Body<Message> {
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

pub fn create_gov_vaa_body(
    i: usize,
    chain_id: Chain,
    channel_id: [u8; 64],
) -> Body<GovernancePacket> {
    Body {
        timestamp: i as u32,
        nonce: i as u32,
        emitter_chain: Chain::Solana,
        emitter_address: GOVERNANCE_EMITTER,
        sequence: i as u64,
        consistency_level: 0,
        payload: GovernancePacket {
            chain: Chain::Wormchain,
            action: Action::UpdateChannelChain {
                channel_id,
                chain_id,
            },
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
