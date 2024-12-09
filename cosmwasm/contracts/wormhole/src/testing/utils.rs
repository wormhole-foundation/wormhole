use crate::{
    contract::{execute, instantiate, query},
    msg::InstantiateMsg,
    state::{GuardianAddress, GuardianSetInfo},
};
use cosmwasm_std::{Addr, Binary, Uint256};
use cw_multi_test::{App, AppBuilder, ContractWrapper, Executor, WasmKeeper};
use k256::ecdsa::SigningKey;
use k256::elliptic_curve::sec1::ToEncodedPoint;
use serde::Serialize;
use tiny_keccak::{Hasher, Keccak};
use wormhole_bindings::fake::WormholeKeeper;
use wormhole_sdk::{
    ibc_receiver::{Action, GovernancePacket},
    token::Message,
    vaa::{Body, Header, Vaa},
    Address, Amount, Chain, GOVERNANCE_EMITTER,
};

static GOV_ADDR: &[u8] = b"GOVERNANCE_ADDRESS";

pub fn sign_vaa_body<P: Serialize>(wh: WormholeKeeper, body: Body<P>) -> (Vaa<P>, Binary) {
    let data = serde_wormhole::to_vec(&body).unwrap();
    let signatures = WormholeKeeper::new().sign(&data);

    let header = Header {
        version: 1,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let v: Vaa<P> = (header, body).into();
    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    (v, data)
}

pub fn create_gov_vaa_body<Packet>(i: usize, packet: Packet) -> Body<Packet> {
    Body {
        timestamp: i as u32,
        nonce: i as u32,
        emitter_chain: Chain::Solana,
        emitter_address: GOVERNANCE_EMITTER,
        sequence: i as u64,
        consistency_level: 0,
        payload: packet,
    }
}

pub fn create_update_channel_chain_packet(
    chain_id: Chain,
    channel_id: [u8; 64],
) -> GovernancePacket {
    GovernancePacket {
        chain: Chain::Wormchain,
        action: Action::UpdateChannelChain {
            channel_id,
            chain_id,
        },
    }
}

pub fn create_update_channel_chain_vaa_body(
    i: usize,
    chain_id: Chain,
    channel_id: [u8; 64],
) -> Body<GovernancePacket> {
    create_gov_vaa_body(i, create_update_channel_chain_packet(chain_id, channel_id))
}

pub fn create_transfer_vaa_body(i: usize, emitter_address: Address) -> Body<Message> {
    Body {
        timestamp: i as u32,
        nonce: i as u32,
        emitter_chain: (i as u16).into(),
        emitter_address,
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

pub fn instantiate_with_guardians(guardians: &[GuardianAddress]) -> InstantiateMsg {
    InstantiateMsg {
        gov_chain: 0,
        gov_address: GOV_ADDR.into(),
        initial_guardian_set: GuardianSetInfo {
            addresses: guardians.to_vec(),
            expiration_time: 100,
        },
        guardian_set_expirity: 50,
        chain_id: 18,
        fee_denom: "uluna".to_string(),
    }
}

/// Instantiate with hardcoded guardian addresses based upon the faker guardian keys in wormhole-bindings
pub fn instantiate_with_faker_guardians() -> InstantiateMsg {
    InstantiateMsg {
        gov_chain: 0,
        gov_address: GOV_ADDR.into(),
        initial_guardian_set: GuardianSetInfo {
            addresses: vec![
                SigningKey::from_bytes(&[
                    93, 217, 189, 224, 168, 81, 157, 93, 238, 38, 143, 8, 182, 94, 69, 77, 232,
                    199, 238, 206, 15, 135, 221, 58, 43, 74, 0, 129, 54, 198, 62, 226,
                ])
                .unwrap()
                .into(),
                SigningKey::from_bytes(&[
                    150, 48, 135, 223, 194, 186, 243, 139, 177, 8, 126, 32, 210, 57, 42, 28, 29,
                    102, 196, 201, 106, 136, 40, 149, 218, 150, 240, 213, 192, 128, 161, 245,
                ])
                .unwrap()
                .into(),
                SigningKey::from_bytes(&[
                    121, 51, 199, 93, 237, 227, 62, 220, 128, 129, 195, 4, 190, 163, 254, 12, 212,
                    224, 188, 76, 141, 242, 229, 121, 192, 5, 161, 176, 136, 99, 83, 53,
                ])
                .unwrap()
                .into(),
                SigningKey::from_bytes(&[
                    224, 180, 4, 114, 215, 161, 184, 12, 218, 96, 20, 141, 154, 242, 46, 230, 167,
                    165, 54, 141, 108, 64, 146, 27, 193, 89, 251, 139, 234, 132, 124, 30,
                ])
                .unwrap()
                .into(),
                SigningKey::from_bytes(&[
                    69, 1, 17, 179, 19, 47, 56, 47, 255, 219, 143, 89, 115, 54, 242, 209, 163, 131,
                    225, 30, 59, 195, 217, 141, 167, 253, 6, 95, 252, 52, 7, 223,
                ])
                .unwrap()
                .into(),
                SigningKey::from_bytes(&[
                    181, 3, 165, 125, 15, 200, 155, 56, 157, 204, 105, 221, 203, 149, 215, 175,
                    220, 228, 200, 37, 169, 39, 68, 127, 132, 196, 203, 232, 155, 55, 67, 253,
                ])
                .unwrap()
                .into(),
                SigningKey::from_bytes(&[
                    72, 81, 175, 107, 23, 108, 178, 66, 32, 53, 14, 117, 233, 33, 114, 102, 68, 89,
                    83, 201, 129, 57, 56, 130, 214, 212, 172, 16, 23, 22, 234, 160,
                ])
                .unwrap()
                .into(),
            ],
            expiration_time: 100,
        },
        guardian_set_expirity: 50,
        chain_id: 18,
        fee_denom: "uluna".to_string(),
    }
}

impl From<SigningKey> for GuardianAddress {
    fn from(value: SigningKey) -> Self {
        // Get the public key bytes
        let public_key = value.verifying_key().to_encoded_point(false);
        let public_key_bytes = public_key.as_bytes();

        // Skip the first byte (0x04 prefix for uncompressed public keys)
        let key_without_prefix = &public_key_bytes[1..];

        // Hash with Keccak-256
        let mut hasher = Keccak::v256();
        let mut hash = [0u8; 32];
        hasher.update(key_without_prefix);
        hasher.finalize(&mut hash);

        // Take last 20 bytes
        let address = &hash[12..32];

        GuardianAddress {
            bytes: address.to_vec().into(),
        }
    }
}

pub struct WormholeApp {
    pub app: App<
        cw_multi_test::BankKeeper,
        cosmwasm_std::testing::MockApi,
        cosmwasm_std::MemoryStorage,
        WormholeKeeper,
        WasmKeeper<cosmwasm_std::Empty, wormhole_bindings::WormholeQuery>,
    >,
    pub admin: Addr,
    pub user: Addr,
    pub wormhole_contract: Addr,
    pub wormhole_keeper: WormholeKeeper,
}

impl WormholeApp {
    pub fn default() -> Self {
        create_wormhole_app(None)
    }
    pub fn new(instantiate_msg: InstantiateMsg) -> Self {
        create_wormhole_app(Some(instantiate_msg))
    }
}

pub fn create_wormhole_app(instantiate_msg: Option<InstantiateMsg>) -> WormholeApp {
    let wormhole_keeper = WormholeKeeper::default();
    let mut app = AppBuilder::new_custom()
        // .with_wasm(WasmKeeper::default())
        .with_custom(wormhole_keeper.clone())
        .build(|_, _, _| {});

    let admin = Addr::unchecked("admin");
    let user = Addr::unchecked("user");

    let cw_wormhole_wrapper = ContractWrapper::new_with_empty(execute, instantiate, query);

    let code_id = app.store_code(Box::new(cw_wormhole_wrapper));

    let contract_addr = app
        .instantiate_contract(
            code_id,
            admin.clone(),
            &instantiate_msg.unwrap_or_else(|| {
                instantiate_with_guardians(&[GuardianAddress {
                    bytes: hex::decode("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")
                        .expect("Decoding failed")
                        .into(),
                }])
            }),
            &[],
            "cw_wormhole",
            None,
        )
        .unwrap();

    WormholeApp {
        app,
        admin,
        user,
        wormhole_contract: contract_addr,
        wormhole_keeper,
    }
}
