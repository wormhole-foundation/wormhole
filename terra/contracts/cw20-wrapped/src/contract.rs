use cosmwasm_std::{
    to_binary,
    Api,
    Binary,
    CosmosMsg,
    Env,
    Extern,
    HandleResponse,
    HumanAddr,
    InitResponse,
    Querier,
    StdError,
    StdResult,
    Storage,
    Uint128,
    WasmMsg,
};

use cw20_base::{
    allowances::{
        handle_burn_from,
        handle_decrease_allowance,
        handle_increase_allowance,
        handle_send_from,
        handle_transfer_from,
        query_allowance,
    },
    contract::{
        handle_mint,
        handle_send,
        handle_transfer,
        query_balance,
    },
    state::{
        token_info,
        token_info_read,
        MinterData,
        TokenInfo,
    },
};

use crate::{
    msg::{
        HandleMsg,
        InitMsg,
        QueryMsg,
        WrappedAssetInfoResponse,
    },
    state::{
        wrapped_asset_info,
        wrapped_asset_info_read,
        WrappedAssetInfo,
    },
};
use cw20::TokenInfoResponse;
use std::string::String;

pub fn init<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    msg: InitMsg,
) -> StdResult<InitResponse> {
    // store token info using cw20-base format
    let data = TokenInfo {
        name: msg.name,
        symbol: msg.symbol,
        decimals: msg.decimals,
        total_supply: Uint128(0),
        // set creator as minter
        mint: Some(MinterData {
            minter: deps.api.canonical_address(&env.message.sender)?,
            cap: None,
        }),
    };
    token_info(&mut deps.storage).save(&data)?;

    // save wrapped asset info
    let data = WrappedAssetInfo {
        asset_chain: msg.asset_chain,
        asset_address: msg.asset_address,
        bridge: deps.api.canonical_address(&env.message.sender)?,
    };
    wrapped_asset_info(&mut deps.storage).save(&data)?;

    if let Some(mint_info) = msg.mint {
        handle_mint(deps, env, mint_info.recipient, mint_info.amount)?;
    }

    if let Some(hook) = msg.init_hook {
        Ok(InitResponse {
            messages: vec![CosmosMsg::Wasm(WasmMsg::Execute {
                contract_addr: hook.contract_addr,
                msg: hook.msg,
                send: vec![],
            })],
            log: vec![],
        })
    } else {
        Ok(InitResponse::default())
    }
}

pub fn handle<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    msg: HandleMsg,
) -> StdResult<HandleResponse> {
    match msg {
        // these all come from cw20-base to implement the cw20 standard
        HandleMsg::Transfer { recipient, amount } => {
            Ok(handle_transfer(deps, env, recipient, amount)?)
        }
        HandleMsg::Burn { account, amount } => Ok(handle_burn_from(deps, env, account, amount)?),
        HandleMsg::Send {
            contract,
            amount,
            msg,
        } => Ok(handle_send(deps, env, contract, amount, msg)?),
        HandleMsg::Mint { recipient, amount } => handle_mint_wrapped(deps, env, recipient, amount),
        HandleMsg::IncreaseAllowance {
            spender,
            amount,
            expires,
        } => Ok(handle_increase_allowance(
            deps, env, spender, amount, expires,
        )?),
        HandleMsg::DecreaseAllowance {
            spender,
            amount,
            expires,
        } => Ok(handle_decrease_allowance(
            deps, env, spender, amount, expires,
        )?),
        HandleMsg::TransferFrom {
            owner,
            recipient,
            amount,
        } => Ok(handle_transfer_from(deps, env, owner, recipient, amount)?),
        HandleMsg::BurnFrom { owner, amount } => Ok(handle_burn_from(deps, env, owner, amount)?),
        HandleMsg::SendFrom {
            owner,
            contract,
            amount,
            msg,
        } => Ok(handle_send_from(deps, env, owner, contract, amount, msg)?),
    }
}

fn handle_mint_wrapped<S: Storage, A: Api, Q: Querier>(
    deps: &mut Extern<S, A, Q>,
    env: Env,
    recipient: HumanAddr,
    amount: Uint128,
) -> StdResult<HandleResponse> {
    // Only bridge can mint
    let wrapped_info = wrapped_asset_info_read(&deps.storage).load()?;
    if wrapped_info.bridge != deps.api.canonical_address(&env.message.sender)? {
        return Err(StdError::unauthorized());
    }

    Ok(handle_mint(deps, env, recipient, amount)?)
}

pub fn query<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
    msg: QueryMsg,
) -> StdResult<Binary> {
    match msg {
        QueryMsg::WrappedAssetInfo {} => to_binary(&query_wrapped_asset_info(deps)?),
        // inherited from cw20-base
        QueryMsg::TokenInfo {} => to_binary(&query_token_info(deps)?),
        QueryMsg::Balance { address } => to_binary(&query_balance(deps, address)?),
        QueryMsg::Allowance { owner, spender } => {
            to_binary(&query_allowance(deps, owner, spender)?)
        }
    }
}

pub fn query_token_info<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
) -> StdResult<TokenInfoResponse> {
    let info = token_info_read(&deps.storage).load()?;
    let res = TokenInfoResponse {
        name: String::from("Wormhole:") + info.name.as_str(),
        symbol: String::from("wh") + info.symbol.as_str(),
        decimals: info.decimals,
        total_supply: info.total_supply,
    };
    Ok(res)
}

pub fn query_wrapped_asset_info<S: Storage, A: Api, Q: Querier>(
    deps: &Extern<S, A, Q>,
) -> StdResult<WrappedAssetInfoResponse> {
    let info = wrapped_asset_info_read(&deps.storage).load()?;
    let res = WrappedAssetInfoResponse {
        asset_chain: info.asset_chain,
        asset_address: info.asset_address,
        bridge: deps.api.human_address(&info.bridge)?,
    };
    Ok(res)
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::{
        testing::{
            mock_dependencies,
            mock_env,
        },
        HumanAddr,
    };
    use cw20::TokenInfoResponse;

    const CANONICAL_LENGTH: usize = 20;

    fn get_balance<S: Storage, A: Api, Q: Querier, T: Into<HumanAddr>>(
        deps: &Extern<S, A, Q>,
        address: T,
    ) -> Uint128 {
        query_balance(&deps, address.into()).unwrap().balance
    }

    fn do_init<S: Storage, A: Api, Q: Querier>(deps: &mut Extern<S, A, Q>, creator: &HumanAddr) {
        let init_msg = InitMsg {
            asset_chain: 1,
            asset_address: vec![1; 32].into(),
            decimals: 10,
            mint: None,
            init_hook: None,
        };
        let env = mock_env(creator, &[]);
        let res = init(deps, env, init_msg).unwrap();
        assert_eq!(0, res.messages.len());

        assert_eq!(
            query_token_info(&deps).unwrap(),
            TokenInfoResponse {
                name: "Wormhole Wrapped".to_string(),
                symbol: "WWT".to_string(),
                decimals: 10,
                total_supply: Uint128::from(0u128),
            }
        );

        assert_eq!(
            query_wrapped_asset_info(&deps).unwrap(),
            WrappedAssetInfoResponse {
                asset_chain: 1,
                asset_address: vec![1; 32].into(),
                bridge: creator.clone(),
            }
        );
    }

    fn do_init_and_mint<S: Storage, A: Api, Q: Querier>(
        deps: &mut Extern<S, A, Q>,
        creator: &HumanAddr,
        mint_to: &HumanAddr,
        amount: Uint128,
    ) {
        do_init(deps, creator);

        let msg = HandleMsg::Mint {
            recipient: mint_to.clone(),
            amount,
        };

        let env = mock_env(&creator, &[]);
        let res = handle(deps, env, msg.clone()).unwrap();
        assert_eq!(0, res.messages.len());
        assert_eq!(get_balance(deps, mint_to), amount);

        assert_eq!(
            query_token_info(&deps).unwrap(),
            TokenInfoResponse {
                name: "Wormhole Wrapped".to_string(),
                symbol: "WWT".to_string(),
                decimals: 10,
                total_supply: amount,
            }
        );
    }

    #[test]
    fn can_mint_by_minter() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let minter = HumanAddr::from("minter");
        let recipient = HumanAddr::from("recipient");
        let amount = Uint128(222_222_222);
        do_init_and_mint(&mut deps, &minter, &recipient, amount);
    }

    #[test]
    fn others_cannot_mint() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let minter = HumanAddr::from("minter");
        let recipient = HumanAddr::from("recipient");
        do_init(&mut deps, &minter);

        let amount = Uint128(222_222_222);
        let msg = HandleMsg::Mint {
            recipient: recipient.clone(),
            amount,
        };

        let other_address = HumanAddr::from("other");
        let env = mock_env(&other_address, &[]);
        let res = handle(&mut deps, env, msg);
        assert_eq!(
            format!("{}", res.unwrap_err()),
            format!("{}", crate::error::ContractError::Unauthorized {})
        );
    }

    #[test]
    fn transfer_balance_success() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let minter = HumanAddr::from("minter");
        let owner = HumanAddr::from("owner");
        let amount_initial = Uint128(222_222_222);
        do_init_and_mint(&mut deps, &minter, &owner, amount_initial);

        // Transfer
        let recipient = HumanAddr::from("recipient");
        let amount_transfer = Uint128(222_222);
        let msg = HandleMsg::Transfer {
            recipient: recipient.clone(),
            amount: amount_transfer,
        };

        let env = mock_env(&owner, &[]);
        let res = handle(&mut deps, env, msg.clone()).unwrap();
        assert_eq!(0, res.messages.len());
        assert_eq!(get_balance(&deps, owner), Uint128(222_000_000));
        assert_eq!(get_balance(&deps, recipient), amount_transfer);
    }

    #[test]
    fn transfer_balance_not_enough() {
        let mut deps = mock_dependencies(CANONICAL_LENGTH, &[]);
        let minter = HumanAddr::from("minter");
        let owner = HumanAddr::from("owner");
        let amount_initial = Uint128(222_221);
        do_init_and_mint(&mut deps, &minter, &owner, amount_initial);

        // Transfer
        let recipient = HumanAddr::from("recipient");
        let amount_transfer = Uint128(222_222);
        let msg = HandleMsg::Transfer {
            recipient: recipient.clone(),
            amount: amount_transfer,
        };

        let env = mock_env(&owner, &[]);
        let _ = handle(&mut deps, env, msg.clone()).unwrap_err(); // Will panic if no error
    }
}
