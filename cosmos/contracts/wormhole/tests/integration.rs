static WASM: &[u8] = include_bytes!("../../../target/wasm32-unknown-unknown/release/wormhole.wasm");

use cosmwasm_std::{from_slice, HumanAddr, Env, InitResponse};
use cosmwasm_storage::to_length_prefixed;
use cosmwasm_vm::testing::{mock_instance, mock_env, init, MockStorage, MockApi, MockQuerier};
use cosmwasm_vm::{Storage, Instance};

use wormhole::msg::{InitMsg, GuardianSetMsg};
use wormhole::state::{ConfigInfo, CONFIG_KEY};

enum TestAddress {
    INITIALIZER,
    GUARDIAN1,
    GUARDIAN2
}

impl TestAddress {
    fn value(&self) -> HumanAddr {
        match self {
            TestAddress::INITIALIZER => HumanAddr::from("initializer"),
            TestAddress::GUARDIAN1 => HumanAddr::from("guardian1"),
            TestAddress::GUARDIAN2 => HumanAddr::from("quardian2"),
        }
    }
}

fn mock_env_height(signer: &HumanAddr, height: u64, time: u64) -> Env {
    let mut env = mock_env(signer, &[]);
    env.block.height = height;
    env.block.time = time;
    env
}

fn get_config_info<S: Storage>(storage: &S) -> ConfigInfo {
    let key = to_length_prefixed(CONFIG_KEY);
    let data = storage
        .get(&key)
        .0
        .expect("error getting data")
        .expect("data should exist");
    from_slice(&data).expect("invalid data")
}

fn do_init(height: u64, guardians: &Vec<HumanAddr>) -> Instance<MockStorage, MockApi, MockQuerier> {
    let mut deps = mock_instance(WASM, &[]);
    let init_msg = InitMsg {
        initial_guardian_set: GuardianSetMsg {
            addresses: guardians.clone(),
            expiration_time: 100
        },
        guardian_set_expirity: 50
    };
    let env = mock_env_height(&TestAddress::INITIALIZER.value(), height, 0);
    let res: InitResponse = init(&mut deps, env, init_msg).unwrap();
    assert_eq!(0, res.messages.len());

    // query the store directly
    deps.with_storage(|storage| {
        assert_eq!(
            get_config_info(storage),
            ConfigInfo {
                guardian_set_index: 0,
                guardian_set_expirity: 50 
            }
        );
        Ok(())
    })
    .unwrap();
    deps
}

#[test]
fn init_works() {
    let guardians = vec![TestAddress::GUARDIAN1.value(), TestAddress::GUARDIAN2.value()];
    let _deps = do_init(111, &guardians);
}