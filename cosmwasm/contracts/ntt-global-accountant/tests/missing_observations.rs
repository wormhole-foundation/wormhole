mod helpers;

use cosmwasm_std::Uint256;
use ntt_global_accountant::msg::Observation;
use wormhole_sdk::{token::Message, Address, Amount, Chain};

fn _create_observation() -> Observation {
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

// TODO: port missing_observations test

// TODO: port different_observations test

// TODO: port guardian_set_change test
