use cosmwasm_std::Addr;
use cw_multi_test::{App, AppBuilder, ContractWrapper, Executor, WasmKeeper};
use cw_wormhole::{
    contract::{execute, instantiate, query},
    msg::InstantiateMsg,
    state::{GuardianAddress, GuardianSetInfo},
};
use k256::ecdsa::SigningKey;
use k256::elliptic_curve::sec1::ToEncodedPoint;
use std::convert::TryInto;
use tiny_keccak::{Hasher, Keccak};
use wormhole_bindings::fake::{default_guardian_keys, WormholeKeeper};
use wormhole_sdk::{Chain, GOVERNANCE_EMITTER};

pub struct WormholeApp {
    pub app: App<
        cw_multi_test::BankKeeper,
        cosmwasm_std::testing::MockApi,
        cosmwasm_std::MemoryStorage,
        WormholeKeeper,
        WasmKeeper<cosmwasm_std::Empty, wormhole_bindings::WormholeQuery>,
    >,
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
                    .map(key_to_guardian_address)
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
                    .map(key_to_guardian_address)
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
        user,
        wormhole_contract: contract_addr,
        wormhole_keeper,
    }
}

pub fn key_to_guardian_address(value: &SigningKey) -> GuardianAddress {
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

pub trait IntoGuardianAddress {
    fn into_guardian_address(self) -> wormhole_sdk::GuardianAddress;
}

impl IntoGuardianAddress for SigningKey {
    fn into_guardian_address(self) -> wormhole_sdk::GuardianAddress {
        let guardian: GuardianAddress = key_to_guardian_address(&self);

        // Take last 20 bytes
        let address: [u8; 20] = guardian.bytes.0.try_into().unwrap();

        wormhole_sdk::GuardianAddress(address)
    }
}
