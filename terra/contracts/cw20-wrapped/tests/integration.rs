static WASM: &[u8] =
    include_bytes!("../../../target/wasm32-unknown-unknown/release/cw20_wrapped.wasm");

use cosmwasm_std::{
    from_slice,
    Binary,
    Env,
    HandleResponse,
    HandleResult,
    HumanAddr,
    InitResponse,
    Uint128,
};
use cosmwasm_storage::to_length_prefixed;
use cosmwasm_vm::{
    testing::{
        handle,
        init,
        mock_env,
        mock_instance,
        query,
        MockApi,
        MockQuerier,
        MockStorage,
    },
    Api,
    Instance,
    Storage,
};
use cw20_wrapped::{
    msg::{
        HandleMsg,
        InitMsg,
        QueryMsg,
    },
    state::{
        WrappedAssetInfo,
        KEY_WRAPPED_ASSET,
    },
    ContractError,
};

enum TestAddress {
    INITIALIZER,
    RECIPIENT,
    SENDER,
}

impl TestAddress {
    fn value(&self) -> HumanAddr {
        match self {
            TestAddress::INITIALIZER => HumanAddr::from("addr0000"),
            TestAddress::RECIPIENT => HumanAddr::from("addr2222"),
            TestAddress::SENDER => HumanAddr::from("addr3333"),
        }
    }
}

fn mock_env_height(signer: &HumanAddr, height: u64, time: u64) -> Env {
    let mut env = mock_env(signer, &[]);
    env.block.height = height;
    env.block.time = time;
    env
}

fn get_wrapped_asset_info<S: Storage>(storage: &S) -> WrappedAssetInfo {
    let key = to_length_prefixed(KEY_WRAPPED_ASSET);
    let data = storage
        .get(&key)
        .0
        .expect("error getting data")
        .expect("data should exist");
    from_slice(&data).expect("invalid data")
}

fn do_init(height: u64) -> Instance<MockStorage, MockApi, MockQuerier> {
    let mut deps = mock_instance(WASM, &[]);
    let init_msg = InitMsg {
        asset_chain: 1,
        asset_address: vec![1; 32].into(),
        decimals: 10,
        mint: None,
        init_hook: None,
    };
    let env = mock_env_height(&TestAddress::INITIALIZER.value(), height, 0);
    let res: InitResponse = init(&mut deps, env, init_msg).unwrap();
    assert_eq!(0, res.messages.len());

    // query the store directly
    let api = deps.api;
    deps.with_storage(|storage| {
        assert_eq!(
            get_wrapped_asset_info(storage),
            WrappedAssetInfo {
                asset_chain: 1,
                asset_address: vec![1; 32].into(),
                bridge: api.canonical_address(&TestAddress::INITIALIZER.value()).0?,
            }
        );
        Ok(())
    })
    .unwrap();
    deps
}

fn do_mint(
    deps: &mut Instance<MockStorage, MockApi, MockQuerier>,
    height: u64,
    recipient: &HumanAddr,
    amount: &Uint128,
) {
    let mint_msg = HandleMsg::Mint {
        recipient: recipient.clone(),
        amount: amount.clone(),
    };
    let env = mock_env_height(&TestAddress::INITIALIZER.value(), height, 0);
    let handle_response: HandleResponse = handle(deps, env, mint_msg).unwrap();
    assert_eq!(0, handle_response.messages.len());
}

fn do_transfer(
    deps: &mut Instance<MockStorage, MockApi, MockQuerier>,
    height: u64,
    sender: &HumanAddr,
    recipient: &HumanAddr,
    amount: &Uint128,
) {
    let transfer_msg = HandleMsg::Transfer {
        recipient: recipient.clone(),
        amount: amount.clone(),
    };
    let env = mock_env_height(sender, height, 0);
    let handle_response: HandleResponse = handle(deps, env, transfer_msg).unwrap();
    assert_eq!(0, handle_response.messages.len());
}

fn check_balance(
    deps: &mut Instance<MockStorage, MockApi, MockQuerier>,
    address: &HumanAddr,
    amount: &Uint128,
) {
    let query_response = query(
        deps,
        QueryMsg::Balance {
            address: address.clone(),
        },
    )
    .unwrap();
    assert_eq!(
        query_response.as_slice(),
        format!("{{\"balance\":\"{}\"}}", amount.u128()).as_bytes()
    );
}

fn check_token_details(deps: &mut Instance<MockStorage, MockApi, MockQuerier>, supply: &Uint128) {
    let query_response = query(deps, QueryMsg::TokenInfo {}).unwrap();
    assert_eq!(
        query_response.as_slice(),
        format!(
            "{{\"name\":\"Wormhole Wrapped\",\
        \"symbol\":\"WWT\",\
        \"decimals\":10,\
        \"total_supply\":\"{}\"}}",
            supply.u128()
        )
        .as_bytes()
    );
}

#[test]
fn init_works() {
    let mut deps = do_init(111);
    check_token_details(&mut deps, &Uint128(0));
}

#[test]
fn query_works() {
    let mut deps = do_init(111);

    let query_response = query(&mut deps, QueryMsg::WrappedAssetInfo {}).unwrap();
    assert_eq!(
        query_response.as_slice(),
        format!(
            "{{\"asset_chain\":1,\
        \"asset_address\":\"{}\",\
        \"bridge\":\"{}\"}}",
            Binary::from(vec![1; 32]).to_base64(),
            TestAddress::INITIALIZER.value().as_str()
        )
        .as_bytes()
    );
}

#[test]
fn mint_works() {
    let mut deps = do_init(111);

    do_mint(
        &mut deps,
        112,
        &TestAddress::RECIPIENT.value(),
        &Uint128(123_123_123),
    );

    check_balance(
        &mut deps,
        &TestAddress::RECIPIENT.value(),
        &Uint128(123_123_123),
    );
    check_token_details(&mut deps, &Uint128(123_123_123));
}

#[test]
fn others_cannot_mint() {
    let mut deps = do_init(111);

    let mint_msg = HandleMsg::Mint {
        recipient: TestAddress::RECIPIENT.value(),
        amount: Uint128(123_123_123),
    };
    let env = mock_env_height(&TestAddress::RECIPIENT.value(), 112, 0);
    let handle_result: HandleResult<HandleResponse> = handle(&mut deps, env, mint_msg);
    assert_eq!(
        format!("{}", handle_result.unwrap_err()),
        format!("{}", ContractError::Unauthorized {})
    );
}

#[test]
fn transfer_works() {
    let mut deps = do_init(111);

    do_mint(
        &mut deps,
        112,
        &TestAddress::SENDER.value(),
        &Uint128(123_123_123),
    );
    do_transfer(
        &mut deps,
        113,
        &TestAddress::SENDER.value(),
        &TestAddress::RECIPIENT.value(),
        &Uint128(123_123_000),
    );

    check_balance(&mut deps, &TestAddress::SENDER.value(), &Uint128(123));
    check_balance(
        &mut deps,
        &TestAddress::RECIPIENT.value(),
        &Uint128(123_123_000),
    );
}
