static WASM: &[u8] = include_bytes!("../../../target/wasm32-unknown-unknown/release/wormhole.wasm");

use cosmwasm_std::{
    from_slice,
    Coin,
    Env,
    HumanAddr,
    InitResponse,
};
use cosmwasm_storage::to_length_prefixed;
use cosmwasm_vm::{
    testing::{
        init,
        mock_env,
        mock_instance,
        MockApi,
        MockQuerier,
        MockStorage,
    },
    Api,
    Instance,
    Storage,
};

use wormhole::{
    msg::InitMsg,
    state::{
        ConfigInfo,
        GuardianAddress,
        GuardianSetInfo,
        CONFIG_KEY,
    },
};

use hex;

enum TestAddress {
    INITIALIZER,
}

impl TestAddress {
    fn value(&self) -> HumanAddr {
        match self {
            TestAddress::INITIALIZER => HumanAddr::from("initializer"),
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

fn do_init(
    height: u64,
    guardians: &Vec<GuardianAddress>,
) -> Instance<MockStorage, MockApi, MockQuerier> {
    let mut deps = mock_instance(WASM, &[]);
    let init_msg = InitMsg {
        initial_guardian_set: GuardianSetInfo {
            addresses: guardians.clone(),
            expiration_time: 100,
        },
        guardian_set_expirity: 50,
        wrapped_asset_code_id: 999,
        chain_id: 18,
        fee_denom: "uluna".to_string(),
    };
    let env = mock_env_height(&TestAddress::INITIALIZER.value(), height, 0);
    let owner = deps
        .api
        .canonical_address(&TestAddress::INITIALIZER.value())
        .0
        .unwrap();
    let res: InitResponse = init(&mut deps, env, init_msg).unwrap();
    assert_eq!(0, res.messages.len());

    // query the store directly
    deps.with_storage(|storage| {
        assert_eq!(
            get_config_info(storage),
            ConfigInfo {
                guardian_set_index: 0,
                guardian_set_expirity: 50,
                wrapped_asset_code_id: 999,
                owner,
                fee: Coin::new(10000, "uluna"),
                chain_id: 18,
            }
        );
        Ok(())
    })
    .unwrap();
    deps
}

#[test]
fn init_works() {
    let guardians = vec![GuardianAddress::from(GuardianAddress {
        bytes: hex::decode("beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")
            .expect("Decoding failed")
            .into(),
    })];
    let _deps = do_init(111, &guardians);
}
