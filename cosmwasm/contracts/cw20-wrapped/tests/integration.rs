use cosmwasm_std::{
    from_json,
    testing::{mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage},
    to_json_binary, Addr, Api, OwnedDeps, Response, Storage, Uint128,
};
use cosmwasm_storage::to_length_prefixed;
use counter::msg as counter_msgs;
use cw20::{AllowanceResponse, BalanceResponse, Expiration, TokenInfoResponse};
use cw20_wrapped_2::{
    contract::{execute, instantiate, query},
    msg::{ExecuteMsg, InitHook, InitMint, InstantiateMsg, QueryMsg, WrappedAssetInfoResponse},
    state::{WrappedAssetInfo, KEY_WRAPPED_ASSET},
    ContractError,
};
use cw_multi_test::{App, AppBuilder, ContractWrapper, Executor};

static INITIALIZER: &str = "addr0000";
static RECIPIENT: &str = "addr2222";
static SENDER: &str = "addr3333";

fn get_wrapped_asset_info<S: Storage>(storage: &S) -> WrappedAssetInfo {
    let key = to_length_prefixed(KEY_WRAPPED_ASSET);
    let data = storage.get(&key).expect("data should exist");
    from_json(&data).expect("invalid data")
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
        from_json::<TokenInfoResponse>(query_response.as_slice()).unwrap(),
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
        from_json::<WrappedAssetInfoResponse>(query_response.as_slice()).unwrap(),
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

pub(crate) struct Cw20App {
    app: App,
    admin: Addr,
    user: Addr,
    cw20_wrapped_contract: Addr,
    cw20_code_id: u64,
}

pub(crate) fn create_cw20_wrapped_app(instantiate_msg: Option<InstantiateMsg>) -> Cw20App {
    let instantiate_msg = instantiate_msg.unwrap_or(InstantiateMsg {
        name: "Integers".into(),
        symbol: "INT".into(),
        asset_chain: 1,
        asset_address: vec![1; 32].into(),
        decimals: 10,
        mint: None,
        init_hook: None,
    });

    let mut app = AppBuilder::new().build(|_, _, _| {});

    let admin = Addr::unchecked("admin");
    let user = Addr::unchecked("user");

    let cw20_wrapped_wrapper = ContractWrapper::new_with_empty(execute, instantiate, query);

    let code_id = app.store_code(Box::new(cw20_wrapped_wrapper));

    let contract_addr = app
        .instantiate_contract(
            code_id,
            admin.clone(),
            &instantiate_msg,
            &[],
            "cw20_wrapped",
            Some(admin.to_string()),
        )
        .unwrap();

    Cw20App {
        app,
        admin,
        user,
        cw20_wrapped_contract: contract_addr,
        cw20_code_id: code_id,
    }
}

pub(crate) struct Cw20AndCounterApp {
    app: App,
    admin: Addr,
    user: Addr,
    cw20_wrapped_contract: Addr,
    cw20_code_id: u64,
    counter_address: Addr,
}

pub(crate) fn create_cw20_and_counter() -> Cw20AndCounterApp {
    let Cw20App {
        mut app,
        admin,
        user,
        cw20_code_id,
        cw20_wrapped_contract,
    } = create_cw20_wrapped_app(None);

    let counter_code_id = app.store_code(Box::new(ContractWrapper::new(
        counter::contract::execute,
        counter::contract::instantiate,
        counter::contract::query,
    )));

    let counter_address = app
        .instantiate_contract(
            counter_code_id,
            user.clone(),
            &counter_msgs::InstantiateMsg {},
            &[],
            "counter",
            None,
        )
        .unwrap();

    let initial_count: counter_msgs::CountResponse = app
        .wrap()
        .query_wasm_smart(
            counter_address.clone(),
            &counter_msgs::QueryMsg::GetCount {},
        )
        .unwrap();
    assert_eq!(initial_count.count, 0, "Initial count should be 0");

    Cw20AndCounterApp {
        app,
        admin,
        user,
        cw20_wrapped_contract,
        cw20_code_id,
        counter_address,
    }
}

#[test]
pub fn instantiate_works() -> Result<(), anyhow::Error> {
    let Cw20App {
        cw20_wrapped_contract,
        app,
        admin,
        user: _user,
        ..
    } = create_cw20_wrapped_app(None);

    let asset_info: WrappedAssetInfoResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::WrappedAssetInfo {},
    )?;

    assert_eq!(
        asset_info,
        WrappedAssetInfoResponse {
            asset_chain: 1,
            asset_address: vec![1; 32].into(),
            bridge: admin,
        }
    );

    let token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;

    assert_eq!(
        token_info,
        TokenInfoResponse {
            name: "Integers (Wormhole)".into(),
            symbol: "INT".into(),
            decimals: 10,
            total_supply: Uint128::zero(),
        }
    );

    Ok(())
}

#[test]
pub fn instantiate_with_minter() -> Result<(), anyhow::Error> {
    let initial_balance_user = Addr::unchecked("user_with_initial_balance");

    let Cw20App {
        cw20_wrapped_contract,
        mut app,
        admin,
        user,
        ..
    } = create_cw20_wrapped_app(Some(InstantiateMsg {
        name: "Integers".into(),
        symbol: "INT".into(),
        asset_chain: 1,
        asset_address: vec![1; 32].into(),
        decimals: 6,
        mint: Some(InitMint {
            recipient: initial_balance_user.to_string(),
            amount: Uint128::from(1_000_000u128),
        }),
        init_hook: None,
    }));

    let init_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: initial_balance_user.to_string(),
        },
    )?;
    assert_eq!(
        init_balance.balance,
        Uint128::from(1_000_000u128),
        "User should have balance from tge"
    );

    let mint_info = ExecuteMsg::Mint {
        recipient: user.to_string(),
        amount: 750_000u128.into(),
    };

    let unauthorized_user_mint =
        app.execute_contract(user.clone(), cw20_wrapped_contract.clone(), &mint_info, &[]);
    assert!(
        unauthorized_user_mint.is_err(),
        "Only Admin should be able to mint"
    );

    // Admin mints to user
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &mint_info,
        &[],
    )?;

    let user_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: user.to_string(),
        },
    )?;

    assert_eq!(
        user_balance.balance,
        Uint128::from(750_000u128),
        "User should have a balance since the admin minted to the user's address"
    );

    Ok(())
}

#[test]
fn instantiate_with_inithook() -> Result<(), anyhow::Error> {
    let Cw20AndCounterApp {
        mut app,
        admin,
        cw20_code_id,
        counter_address,
        ..
    } = create_cw20_and_counter();

    let instantiation_with_init_hook = InstantiateMsg {
        name: "Floats".into(),
        symbol: "FLT".into(),
        asset_chain: 1,
        asset_address: vec![2; 32].into(),
        decimals: 6,
        mint: None,
        init_hook: Some(InitHook {
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
            contract_addr: counter_address.to_string(),
        }),
    };

    app.instantiate_contract(
        cw20_code_id,
        admin.clone(),
        &instantiation_with_init_hook,
        &[],
        "floats_cw20",
        None,
    )?;

    let incremented_count: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        incremented_count.count, 1,
        "After FLT is instantiated, the count should be incremented"
    );

    Ok(())
}

#[test]
fn update_metadata_functionality() -> Result<(), anyhow::Error> {
    let Cw20App {
        cw20_wrapped_contract,
        mut app,
        admin,
        user,
        ..
    } = create_cw20_wrapped_app(None);

    // Verify initial metadata
    let initial_token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;

    assert_eq!(
        initial_token_info,
        TokenInfoResponse {
            name: "Integers (Wormhole)".into(),
            symbol: "INT".into(),
            decimals: 10,
            total_supply: Uint128::zero(),
        }
    );

    // Try to update metadata as non-admin (should fail)
    let unauthorized_update = app.execute_contract(
        user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::UpdateMetadata {
            name: "Updated Token".into(),
            symbol: "UPDT".into(),
        },
        &[],
    );
    assert!(
        unauthorized_update.is_err(),
        "Non-admin should not be able to update metadata"
    );

    // Verify metadata hasn't changed after failed update
    let unchanged_token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;
    assert_eq!(unchanged_token_info, initial_token_info);

    // Update metadata as admin
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::UpdateMetadata {
            name: "Updated Integers".into(),
            symbol: "UINT".into(),
        },
        &[],
    )?;

    // Verify metadata was updated
    let updated_token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;

    assert_eq!(
        updated_token_info,
        TokenInfoResponse {
            name: "Updated Integers (Wormhole)".into(),
            symbol: "UINT".into(),
            decimals: 10,
            total_supply: Uint128::zero(),
        }
    );

    // Update metadata again to verify multiple updates work
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::UpdateMetadata {
            name: "Final Name".into(),
            symbol: "FINAL".into(),
        },
        &[],
    )?;

    // Verify final metadata update
    let final_token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;

    assert_eq!(
        final_token_info,
        TokenInfoResponse {
            name: "Final Name (Wormhole)".into(),
            symbol: "FINAL".into(),
            decimals: 10,
            total_supply: Uint128::zero(),
        }
    );

    Ok(())
}

#[test]
fn send_tokens_functionality() -> Result<(), anyhow::Error> {
    let Cw20AndCounterApp {
        mut app,
        admin,
        user,
        counter_address,
        cw20_wrapped_contract: cw20_address,
        ..
    } = create_cw20_and_counter();

    app.execute_contract(
        admin.clone(),
        cw20_address.clone(),
        &ExecuteMsg::Mint {
            recipient: user.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    let send_to_counter_app = ExecuteMsg::Send {
        contract: counter_address.to_string(),
        amount: Uint128::from(1_000_000u128),
        msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
    };

    let failing_send = app.execute_contract(
        admin.clone(),
        cw20_address.clone(),
        &send_to_counter_app,
        &[],
    );
    assert!(
        failing_send.is_err(),
        "Admin should not be able to send to the counter contract since it has no balance"
    );

    // Now send it as user who actually has a balance
    let send_resp = app.execute_contract(
        user.clone(),
        cw20_address.clone(),
        &send_to_counter_app,
        &[],
    );
    println!("send response {:?}", send_resp);
    send_resp?;

    let incremented_count: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        incremented_count.count, 1,
        "Post-send to counter contract, the count should be incremented"
    );

    // Try to send it again as User to double check that things fail properly
    let failing_send = app.execute_contract(
        user.clone(),
        cw20_address.clone(),
        &send_to_counter_app,
        &[],
    );
    assert!(
        failing_send.is_err(),
        "User should not be able to send to the counter contract since it has no balance"
    );

    let incremented_count: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        incremented_count.count, 1,
        "Count should still be 1 since it hasn't been incremented again"
    );

    Ok(())
}

// Test that transfer works correctly
#[test]
fn transfer_tokens_functionality() -> Result<(), anyhow::Error> {
    let Cw20App {
        cw20_wrapped_contract,
        mut app,
        admin,
        user,
        ..
    } = create_cw20_wrapped_app(None);

    // Mint some tokens to the user
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Mint {
            recipient: user.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    // Validate the user has the correct balance
    let user_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: user.to_string(),
        },
    )?;
    assert_eq!(user_balance.balance, Uint128::from(1_000_000u128));

    // Attempt to transfer as the admin which should fail
    let admin_transfer = app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Transfer {
            recipient: admin.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    );
    assert!(
        admin_transfer.is_err(),
        "Admin should not be able to transfer tokens since it has no balance"
    );

    // Attempt to transfer too much as the user which should fail
    let user_transfer = app.execute_contract(
        user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Transfer {
            recipient: admin.to_string(),
            amount: 1_000_001u128.into(),
        },
        &[],
    );
    assert!(
        user_transfer.is_err(),
        "User should not be able to transfer more tokens than they have"
    );

    // Transfer 500,000 tokens from the user to the admin
    app.execute_contract(
        user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Transfer {
            recipient: admin.to_string(),
            amount: 500_000u128.into(),
        },
        &[],
    )?;

    // Validate the user has the correct balance
    let user_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: user.to_string(),
        },
    )?;
    assert_eq!(user_balance.balance, Uint128::from(500_000u128));

    // Validate the admin has the correct balance
    let admin_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: admin.to_string(),
        },
    )?;
    assert_eq!(admin_balance.balance, Uint128::from(500_000u128));

    Ok(())
}

#[test]
fn burn_functionality() -> Result<(), anyhow::Error> {
    let Cw20App {
        cw20_wrapped_contract,
        mut app,
        admin,
        ..
    } = create_cw20_wrapped_app(None);

    let burn_user = Addr::unchecked("burner");
    let approved_burner = Addr::unchecked("approved_burner");

    // Mint some tokens to the burn user
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Mint {
            recipient: burn_user.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    // Verify initial balance
    let initial_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: burn_user.to_string(),
        },
    )?;
    assert_eq!(initial_balance.balance, Uint128::from(1_000_000u128));

    // Try to burn tokens without allowance (should fail)
    let unauthorized_self_burn = app.execute_contract(
        burn_user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Burn {
            account: burn_user.to_string(),
            amount: 500_000u128.into(),
        },
        &[],
    );
    assert!(
        unauthorized_self_burn.is_err(),
        "Should not be able to burn tokens without allowance from another address"
    );

    // Try to set allowance for self (should fail)
    let self_allowance = app.execute_contract(
        burn_user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: burn_user.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    );
    assert!(
        self_allowance.is_err(),
        "Should not be able to set allowance for self"
    );

    // Set up allowance for approved burner
    app.execute_contract(
        burn_user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: approved_burner.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Try to burn more tokens than allowance (should fail)
    let excessive_burn = app.execute_contract(
        approved_burner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Burn {
            account: burn_user.to_string(),
            amount: 750_000u128.into(),
        },
        &[],
    );
    assert!(
        excessive_burn.is_err(),
        "Should not be able to burn more tokens than allowed"
    );

    // Successfully burn tokens with proper allowance
    app.execute_contract(
        approved_burner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Burn {
            account: burn_user.to_string(),
            amount: 500_000u128.into(),
        },
        &[],
    )?;

    // Verify remaining balance
    let remaining_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: burn_user.to_string(),
        },
    )?;
    assert_eq!(remaining_balance.balance, Uint128::from(500_000u128));

    // Verify total supply has decreased
    let token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;
    assert_eq!(
        token_info.total_supply,
        Uint128::from(500_000u128),
        "Total supply should reflect burned tokens"
    );

    // Verify allowance was properly decreased
    let allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Allowance {
            owner: burn_user.to_string(),
            spender: approved_burner.to_string(),
        },
    )?;
    assert_eq!(
        allowance.allowance,
        Uint128::zero(),
        "Allowance should be depleted after burn"
    );

    Ok(())
}

#[test]
/// BurnFrom uses identical logic to Burn
fn burn_from_functionality() -> Result<(), anyhow::Error> {
    let Cw20App {
        cw20_wrapped_contract,
        mut app,
        admin,
        ..
    } = create_cw20_wrapped_app(None);

    let burn_user = Addr::unchecked("burner");
    let approved_burner = Addr::unchecked("approved_burner");

    // Mint some tokens to the burn user
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Mint {
            recipient: burn_user.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    // Verify initial balance
    let initial_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: burn_user.to_string(),
        },
    )?;
    assert_eq!(initial_balance.balance, Uint128::from(1_000_000u128));

    // Try to burn tokens without allowance (should fail)
    let unauthorized_self_burn = app.execute_contract(
        burn_user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::BurnFrom {
            owner: burn_user.to_string(),
            amount: 500_000u128.into(),
        },
        &[],
    );
    assert!(
        unauthorized_self_burn.is_err(),
        "Should not be able to burn tokens without allowance from another address"
    );

    // Try to set allowance for self (should fail)
    let self_allowance = app.execute_contract(
        burn_user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: burn_user.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    );
    assert!(
        self_allowance.is_err(),
        "Should not be able to set allowance for self"
    );

    // Set up allowance for approved burner
    app.execute_contract(
        burn_user.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: approved_burner.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Try to burn more tokens than allowance (should fail)
    let excessive_burn = app.execute_contract(
        approved_burner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::BurnFrom {
            owner: burn_user.to_string(),
            amount: 750_000u128.into(),
        },
        &[],
    );
    assert!(
        excessive_burn.is_err(),
        "Should not be able to burn more tokens than allowed"
    );

    // Successfully burn tokens with proper allowance
    app.execute_contract(
        approved_burner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::BurnFrom {
            owner: burn_user.to_string(),
            amount: 500_000u128.into(),
        },
        &[],
    )?;

    // Verify remaining balance
    let remaining_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Balance {
            address: burn_user.to_string(),
        },
    )?;
    assert_eq!(remaining_balance.balance, Uint128::from(500_000u128));

    // Verify total supply has decreased
    let token_info: TokenInfoResponse = app
        .wrap()
        .query_wasm_smart(cw20_wrapped_contract.clone(), &QueryMsg::TokenInfo {})?;
    assert_eq!(
        token_info.total_supply,
        Uint128::from(500_000u128),
        "Total supply should reflect burned tokens"
    );

    // Verify allowance was properly decreased
    let allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Allowance {
            owner: burn_user.to_string(),
            spender: approved_burner.to_string(),
        },
    )?;
    assert_eq!(
        allowance.allowance,
        Uint128::zero(),
        "Allowance should be depleted after burn"
    );

    Ok(())
}

#[test]
fn allowance_management() -> Result<(), anyhow::Error> {
    let Cw20App {
        cw20_wrapped_contract,
        mut app,
        admin,
        ..
    } = create_cw20_wrapped_app(None);

    let token_owner = Addr::unchecked("token_owner");
    let spender = Addr::unchecked("spender");

    // Mint tokens to the token owner
    app.execute_contract(
        admin.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Mint {
            recipient: token_owner.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    // Test that setting allowance for self fails
    let self_allowance_increase = app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: token_owner.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    );
    assert!(
        self_allowance_increase.is_err(),
        "Should not be able to increase allowance for self"
    );

    // Test that decreasing allowance for self also fails
    let self_allowance_decrease = app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::DecreaseAllowance {
            spender: token_owner.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    );
    assert!(
        self_allowance_decrease.is_err(),
        "Should not be able to decrease allowance for self"
    );

    // Test increasing allowance for another address
    app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: spender.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Verify initial allowance
    let allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(allowance.allowance, Uint128::from(500_000u128));

    // Test increasing allowance again (should add to existing)
    app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: spender.to_string(),
            amount: 200_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Verify cumulative allowance
    let increased_allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(increased_allowance.allowance, Uint128::from(700_000u128));

    // Test decreasing allowance
    app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::DecreaseAllowance {
            spender: spender.to_string(),
            amount: 300_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Verify decreased allowance
    let decreased_allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(decreased_allowance.allowance, Uint128::from(400_000u128));

    // Decreasing allowance bellow zero results in a zero allowance
    let _ = app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::DecreaseAllowance {
            spender: spender.to_string(),
            amount: 5_000_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Verify allowance is 0
    let zero_allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_wrapped_contract.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(zero_allowance.allowance, Uint128::zero());

    // Test expiration
    let expiration = Expiration::AtHeight(app.block_info().height + 1);

    app.execute_contract(
        token_owner.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: spender.to_string(),
            amount: 100_000u128.into(),
            expires: Some(expiration),
        },
        &[],
    )?;

    // Move to next block
    app.update_block(|block| block.height += 2);

    // Try to use expired allowance (should fail)
    let burn_with_expired = app.execute_contract(
        spender.clone(),
        cw20_wrapped_contract.clone(),
        &ExecuteMsg::Burn {
            account: token_owner.to_string(),
            amount: 100_000u128.into(),
        },
        &[],
    );
    assert!(
        burn_with_expired.is_err(),
        "Should not be able to use expired allowance"
    );

    Ok(())
}

#[test]
fn send_from_functionality() -> Result<(), anyhow::Error> {
    let Cw20AndCounterApp {
        mut app,
        admin,
        user: spender,
        counter_address,
        cw20_wrapped_contract: cw20_address,
        ..
    } = create_cw20_and_counter();

    let token_owner = Addr::unchecked("token_owner");

    // Mint tokens to the token owner
    app.execute_contract(
        admin.clone(),
        cw20_address.clone(),
        &ExecuteMsg::Mint {
            recipient: token_owner.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    // Try to send_from without allowance (should fail)
    let unauthorized_send = app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 500_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    );
    assert!(
        unauthorized_send.is_err(),
        "Should not be able to send_from without allowance"
    );

    // Set up allowance for spender
    app.execute_contract(
        token_owner.clone(),
        cw20_address.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: spender.to_string(),
            amount: 500_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // Try to send_from more than allowance (should fail)
    let excessive_send = app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 600_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    );
    assert!(
        excessive_send.is_err(),
        "Should not be able to send_from more than allowance"
    );

    // Successfully send tokens and increment counter
    app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 300_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    )?;

    // Verify counter was incremented
    let counter_response: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(counter_response.count, 1, "Counter should be incremented");

    // Verify token owner's balance was decreased
    let owner_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_address.clone(),
        &QueryMsg::Balance {
            address: token_owner.to_string(),
        },
    )?;
    assert_eq!(owner_balance.balance, Uint128::from(700_000u128));

    // Verify allowance was decreased
    let updated_allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_address.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(updated_allowance.allowance, Uint128::from(200_000u128));

    // Test with expired allowance
    let expiration = Expiration::AtHeight(app.block_info().height + 1);

    // Set new allowance with expiration
    app.execute_contract(
        token_owner.clone(),
        cw20_address.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: spender.to_string(),
            amount: 100_000u128.into(),
            expires: Some(expiration),
        },
        &[],
    )?;

    // Move past expiration
    app.update_block(|block| block.height += 2);

    // Try to send_from with expired allowance (should fail)
    let expired_send = app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 50_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    );
    assert!(
        expired_send.is_err(),
        "Should not be able to send_from with expired allowance"
    );

    // Verify counter wasn't incremented again
    let final_count: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        final_count.count, 1,
        "Counter should not be incremented again"
    );

    Ok(())
}

#[test]
fn multiple_send_from_functionality() -> Result<(), anyhow::Error> {
    let Cw20AndCounterApp {
        mut app,
        admin,
        user: spender,
        counter_address,
        cw20_wrapped_contract: cw20_address,
        ..
    } = create_cw20_and_counter();

    let token_owner = Addr::unchecked("token_owner");

    // Mint tokens to the token owner
    app.execute_contract(
        admin.clone(),
        cw20_address.clone(),
        &ExecuteMsg::Mint {
            recipient: token_owner.to_string(),
            amount: 1_000_000u128.into(),
        },
        &[],
    )?;

    // Set up allowance for spender
    app.execute_contract(
        token_owner.clone(),
        cw20_address.clone(),
        &ExecuteMsg::IncreaseAllowance {
            spender: spender.to_string(),
            amount: 750_000u128.into(),
            expires: None,
        },
        &[],
    )?;

    // First send_from operation
    app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 250_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    )?;

    // Verify first operation results
    let count_after_first: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        count_after_first.count, 1,
        "Counter should be incremented once"
    );

    let allowance_after_first: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_address.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(
        allowance_after_first.allowance,
        Uint128::from(500_000u128),
        "Allowance should be decreased by first send"
    );

    // Second send_from operation
    app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 200_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    )?;

    // Verify second operation results
    let count_after_second: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        count_after_second.count, 2,
        "Counter should be incremented twice"
    );

    let allowance_after_second: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_address.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(
        allowance_after_second.allowance,
        Uint128::from(300_000u128),
        "Allowance should be decreased by second send"
    );

    // Third send_from operation
    app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 300_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    )?;

    // Verify final state
    let final_count: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        final_count.count, 3,
        "Counter should be incremented three times"
    );

    let final_allowance: AllowanceResponse = app.wrap().query_wasm_smart(
        cw20_address.clone(),
        &QueryMsg::Allowance {
            owner: token_owner.to_string(),
            spender: spender.to_string(),
        },
    )?;
    assert_eq!(
        final_allowance.allowance,
        Uint128::zero(),
        "Allowance should be fully spent"
    );

    let final_balance: BalanceResponse = app.wrap().query_wasm_smart(
        cw20_address.clone(),
        &QueryMsg::Balance {
            address: token_owner.to_string(),
        },
    )?;
    assert_eq!(
        final_balance.balance,
        Uint128::from(250_000u128),
        "Owner balance should reflect all three sends"
    );

    // Try one more send_from operation with depleted allowance (should fail)
    let depleted_allowance_send = app.execute_contract(
        spender.clone(),
        cw20_address.clone(),
        &ExecuteMsg::SendFrom {
            owner: token_owner.to_string(),
            contract: counter_address.to_string(),
            amount: 100_000u128.into(),
            msg: to_json_binary(&counter_msgs::ExecuteMsg::Increment {})?,
        },
        &[],
    );
    assert!(
        depleted_allowance_send.is_err(),
        "Should not be able to send_from after allowance is depleted"
    );

    // Verify counter wasn't incremented by failed transaction
    let final_failed_count: counter_msgs::CountResponse = app.wrap().query_wasm_smart(
        counter_address.clone(),
        &counter_msgs::QueryMsg::GetCount {},
    )?;
    assert_eq!(
        final_failed_count.count, 3,
        "Counter should not be incremented after failed send"
    );

    Ok(())
}
