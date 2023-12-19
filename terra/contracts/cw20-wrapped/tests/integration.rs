use cosmwasm_std::{
    from_slice,
    testing::{mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage},
    Addr, Api, OwnedDeps, Response, Storage, Uint128,
};
use cosmwasm_storage::to_length_prefixed;
use cw20::TokenInfoResponse;
use cw20_wrapped::{
    contract::{execute, instantiate, query},
    msg::{ExecuteMsg, InstantiateMsg, QueryMsg, WrappedAssetInfoResponse},
    state::{WrappedAssetInfo, KEY_WRAPPED_ASSET},
    ContractError,
};

static INITIALIZER: &str = "addr0000";
static RECIPIENT: &str = "addr2222";
static SENDER: &str = "addr3333";

fn get_wrapped_asset_info<S: Storage>(storage: &S) -> WrappedAssetInfo {
    let key = to_length_prefixed(KEY_WRAPPED_ASSET);
    let data = storage.get(&key).expect("data should exist");
    from_slice(&data).expect("invalid data")
}

fn do_init() -> OwnedDeps<MockStorage, MockApi, MockQuerier> {
    let mut deps = mock_dependencies();
    let init_msg = InstantiateMsg {
        name: "Integers".into(),
        symbol: "INT".into(),
        asset_chain: 1,
        asset_address: vec![1; 32].into(),
        decimals: 10,
        mint: None,
        init_hook: None,
    };
    let env = mock_env();
    let info = mock_info(INITIALIZER, &[]);
    let res: Response = instantiate(deps.as_mut(), env, info, init_msg).unwrap();
    assert_eq!(0, res.messages.len());

    // query the store directly
    let bridge = deps.api.addr_canonicalize(INITIALIZER).unwrap();
    assert_eq!(
        get_wrapped_asset_info(&deps.storage),
        WrappedAssetInfo {
            asset_chain: 1,
            asset_address: vec![1; 32].into(),
            bridge,
        }
    );

    deps
}

fn do_mint(
    deps: &mut OwnedDeps<MockStorage, MockApi, MockQuerier>,
    recipient: &Addr,
    amount: &Uint128,
) {
    let mint_msg = ExecuteMsg::Mint {
        recipient: recipient.to_string(),
        amount: *amount,
    };
    let info = mock_info(INITIALIZER, &[]);
    let handle_response: Response = execute(deps.as_mut(), mock_env(), info, mint_msg).unwrap();
    assert_eq!(0, handle_response.messages.len());
}

fn do_transfer(
    deps: &mut OwnedDeps<MockStorage, MockApi, MockQuerier>,
    sender: &Addr,
    recipient: &Addr,
    amount: &Uint128,
) {
    let transfer_msg = ExecuteMsg::Transfer {
        recipient: recipient.to_string(),
        amount: *amount,
    };
    let env = mock_env();
    let info = mock_info(sender.as_str(), &[]);
    let handle_response: Response = execute(deps.as_mut(), env, info, transfer_msg).unwrap();
    assert_eq!(0, handle_response.messages.len());
}

fn check_balance(
    deps: &OwnedDeps<MockStorage, MockApi, MockQuerier>,
    address: &Addr,
    amount: &Uint128,
) {
    let query_response = query(
        deps.as_ref(),
        mock_env(),
        QueryMsg::Balance {
            address: address.to_string(),
        },
    )
    .unwrap();
    assert_eq!(
        query_response.as_slice(),
        format!("{{\"balance\":\"{}\"}}", amount.u128()).as_bytes()
    );
}

fn check_token_details(deps: &OwnedDeps<MockStorage, MockApi, MockQuerier>, supply: Uint128) {
    let query_response = query(deps.as_ref(), mock_env(), QueryMsg::TokenInfo {}).unwrap();
    assert_eq!(
        from_slice::<TokenInfoResponse>(query_response.as_slice()).unwrap(),
        TokenInfoResponse {
            name: "Integers (Wormhole)".into(),
            symbol: "INT".into(),
            decimals: 10,
            total_supply: supply,
        }
    );
}

#[test]
fn init_works() {
    let deps = do_init();
    check_token_details(&deps, Uint128::new(0));
}

#[test]
fn query_works() {
    let deps = do_init();

    let query_response = query(deps.as_ref(), mock_env(), QueryMsg::WrappedAssetInfo {}).unwrap();
    assert_eq!(
        from_slice::<WrappedAssetInfoResponse>(query_response.as_slice()).unwrap(),
        WrappedAssetInfoResponse {
            asset_chain: 1,
            asset_address: vec![1; 32].into(),
            bridge: Addr::unchecked(INITIALIZER),
        }
    );
}

#[test]
fn mint_works() {
    let mut deps = do_init();

    let recipient = Addr::unchecked(RECIPIENT);
    do_mint(&mut deps, &recipient, &Uint128::new(123_123_123));

    check_balance(&deps, &recipient, &Uint128::new(123_123_123));
    check_token_details(&deps, Uint128::new(123_123_123));
}

#[test]
fn others_cannot_mint() {
    let mut deps = do_init();

    let mint_msg = ExecuteMsg::Mint {
        recipient: RECIPIENT.into(),
        amount: Uint128::new(123_123_123),
    };
    let env = mock_env();
    let info = mock_info(RECIPIENT, &[]);
    let handle_result = execute(deps.as_mut(), env, info, mint_msg);
    assert_eq!(
        format!("{}", handle_result.unwrap_err()),
        format!("{}", ContractError::Unauthorized {})
    );
}

#[test]
fn transfer_works() {
    let mut deps = do_init();

    let sender = Addr::unchecked(SENDER);
    let recipient = Addr::unchecked(RECIPIENT);
    do_mint(&mut deps, &sender, &Uint128::new(123_123_123));
    do_transfer(&mut deps, &sender, &recipient, &Uint128::new(123_123_000));

    check_balance(&deps, &sender, &Uint128::new(123));
    check_balance(&deps, &recipient, &Uint128::new(123_123_000));
}
