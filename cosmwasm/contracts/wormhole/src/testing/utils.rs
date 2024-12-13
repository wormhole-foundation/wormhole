use std::convert::TryInto;

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
use wormhole_bindings::fake::{create_vaa_body, default_guardian_keys, WormholeKeeper};
use wormhole_sdk::{
    token::Message,
    vaa::{Body, Header, Vaa},
    Address, Amount, Chain, GOVERNANCE_EMITTER,
};

/// Sign a VAA body with version 2 in the header.
pub fn sign_vaa_body_version_2<P: Serialize>(
    wh: WormholeKeeper,
    body: Body<P>,
) -> (Vaa<P>, Binary) {
    let data = serde_wormhole::to_vec(&body).unwrap();
    let signatures = WormholeKeeper::new().sign(&data);

    let header = Header {
        version: 2,
        guardian_set_index: wh.guardian_set_index(),
        signatures,
    };

    let v: Vaa<P> = (header, body).into();
    let data = serde_wormhole::to_vec(&v).map(From::from).unwrap();

    (v, data)
}

pub fn create_transfer_vaa_body(i: usize, emitter_address: Address) -> Body<Message> {
    create_vaa_body(
        i,
        i as u16,
        emitter_address,
        Message::Transfer {
            amount: Amount(Uint256::from(i as u128).to_be_bytes()),
            token_address: Address([(i + 1) as u8; 32]),
            token_chain: (i as u16).into(),
            recipient: Address([i as u8; 32]),
            recipient_chain: ((i + 2) as u16).into(),
            fee: Amount([0u8; 32]),
        },
    )
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
    pub fn new_with_guardians(guardians: Vec<SigningKey>) -> Self {
        create_wormhole_app(Some((
            instantiate_with_guardians(
                guardians
                    .iter()
                    .map(|k| k.clone().into())
                    .collect::<Vec<GuardianAddress>>()
                    .as_slice(),
            ),
            guardians,
        )))
    }
    pub fn new_with_faker_guardians() -> Self {
        create_wormhole_app(Some((
            instantiate_with_guardians(
                default_guardian_keys()
                    .iter()
                    .map(|k| k.clone().into())
                    .collect::<Vec<GuardianAddress>>()
                    .as_slice(),
            ),
            default_guardian_keys().to_vec(),
        )))
    }
}

pub fn instantiate_with_guardians(guardians: &[GuardianAddress]) -> InstantiateMsg {
    InstantiateMsg {
        gov_chain: Chain::Solana.into(),
        gov_address: GOVERNANCE_EMITTER.0.into(),
        initial_guardian_set: GuardianSetInfo {
            addresses: guardians.to_vec(),
            expiration_time: 1571797500,
        },
        guardian_set_expirity: 50,
        chain_id: Chain::Terra2.into(),
        fee_denom: "uluna".to_string(),
    }
}

pub fn create_wormhole_app(
    instantiate_msg: Option<(InstantiateMsg, Vec<SigningKey>)>,
) -> WormholeApp {
    let (instantiate_msg, keys) = instantiate_msg.unwrap_or_else(|| {
        let key_bytes =
            hex::decode("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe").expect("Decoding failed");
        (
            instantiate_with_guardians(&[GuardianAddress {
                bytes: key_bytes.clone().into(),
            }]),
            vec![SigningKey::from_bytes(key_bytes.as_slice()).unwrap()],
        )
    });

    let wormhole_keeper: WormholeKeeper = keys.to_vec().into();

    let mut app = AppBuilder::new_custom()
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
            &instantiate_msg,
            &[],
            "cw_wormhole",
            Some(admin.to_string()),
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

pub trait IntoGuardianAddress {
    fn into_guardian_address(self) -> wormhole_sdk::GuardianAddress;
}

impl IntoGuardianAddress for SigningKey {
    fn into_guardian_address(self) -> wormhole_sdk::GuardianAddress {
        // Get the public key bytes
        let public_key = self.verifying_key().to_encoded_point(false);
        let public_key_bytes = public_key.as_bytes();

        // Skip the first byte (0x04 prefix for uncompressed public keys)
        let key_without_prefix = &public_key_bytes[1..];

        // Hash with Keccak-256
        let mut hasher = Keccak::v256();
        let mut hash = [0u8; 32];
        hasher.update(key_without_prefix);
        hasher.finalize(&mut hash);

        // Take last 20 bytes
        let address: [u8; 20] = hash[12..32].try_into().unwrap();

        wormhole_sdk::GuardianAddress(address)
    }
}
