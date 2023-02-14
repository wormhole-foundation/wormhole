use std::marker::PhantomData;

use anyhow::{bail, ensure, Context};
use cosmwasm_std::{CustomQuery, Deps, DepsMut, Event, Order, StdResult};
use cw_storage_plus::Bound;
use thiserror::Error as ThisError;

use crate::{
    msg::Instantiate,
    state::{
        account::{self, Balance},
        transfer, Account, Kind, Modification, Transfer, ACCOUNTS, MODIFICATIONS, TRANSFERS,
    },
};

/// Instantiate the on-chain state for accountant.  Unlike other methods in this crate,
/// `instantiate` does not perform any validation of the data in `init`.
pub fn instantiate<C: CustomQuery>(deps: DepsMut<C>, init: Instantiate) -> anyhow::Result<Event> {
    let num_accounts = init.accounts.len();
    let num_transfers = init.transfers.len();
    let num_modifications = init.modifications.len();

    for a in init.accounts {
        ACCOUNTS
            .save(deps.storage, a.key, &a.balance)
            .context("failed to save `Account`")?;
    }

    for t in init.transfers {
        TRANSFERS
            .save(deps.storage, t.key, &t.data)
            .context("failed to save `Transfer`")?;
    }

    for m in init.modifications {
        MODIFICATIONS
            .save(deps.storage, m.sequence, &m)
            .context("failed to save `Modification`")?;
    }

    Ok(Event::new("InstantiateAccounting")
        .add_attribute("num_accounts", num_accounts.to_string())
        .add_attribute("num_transfers", num_transfers.to_string())
        .add_attribute("num_modifications", num_modifications.to_string()))
}

#[derive(ThisError, Debug)]
pub enum TransferError {
    #[error("transfer already committed")]
    DuplicateTransfer,
    #[error("insufficient balance in destination account")]
    InsufficientDestinationBalance,
    #[error("insufficient balance in source account")]
    InsufficientSourceBalance,
    #[error("cannot unlock native tokens without an existing native account")]
    MissingNativeAccount,
    #[error("cannot burn wrapped tokens without an existing wrapped account")]
    MissingWrappedAccount,
}

/// Validates a transfer without committing it to the on-chain state.  If an error occurs that is
/// not due to the underlying cosmwasm framework, the returned error will be downcastable to
/// `TransferError`.
pub fn validate_transfer<C: CustomQuery>(deps: Deps<C>, t: &Transfer) -> anyhow::Result<()> {
    transfer(deps, t).map(drop)
}

/// Commits a transfer to the on-chain state.  If an error occurs that is not due to the underlying
/// cosmwasm framework, the returned error will be downcastable to `TransferError`.
///
/// # Examples
///
/// ```
/// # fn example() -> anyhow::Result<()> {
/// #     use accountant::{
/// #         commit_transfer,
/// #         state::{transfer, Transfer},
/// #         TransferError,
/// #     };
/// #     use cosmwasm_std::{testing::mock_dependencies, Uint256};
/// #
/// #     let mut deps = mock_dependencies();
/// #     let tx = Transfer {
/// #         key: transfer::Key::new(3, [1u8; 32].into(), 5),
/// #         data: transfer::Data {
/// #             amount: Uint256::from(400u128),
/// #             token_chain: 3,
/// #             token_address: [3u8; 32].into(),
/// #             recipient_chain: 9,
/// #         },
/// #     };
/// #
///       commit_transfer(deps.as_mut(), tx.clone())?;
///  
///       // Repeating the transfer should return an error.
///       let err = commit_transfer(deps.as_mut(), tx)
///           .expect_err("successfully committed duplicate transfer");
///       if let Some(e) = err.downcast_ref::<TransferError>() {
///           assert!(matches!(e, TransferError::DuplicateTransfer));
///       } else {
///           println!("framework error: {err:#}");
///       }
/// #
/// #     Ok(())
/// # }
/// #
/// # example().unwrap();
/// ```
pub fn commit_transfer<C: CustomQuery>(deps: DepsMut<C>, t: Transfer) -> anyhow::Result<Event> {
    let (src, dst) = transfer(deps.as_ref(), &t)?;

    ACCOUNTS
        .save(deps.storage, src.key, &src.balance)
        .context("failed to save updated source account")?;
    ACCOUNTS
        .save(deps.storage, dst.key, &dst.balance)
        .context("failed to save updated destination account")?;

    let evt = cw_transcode::to_event(&t).context("failed to transcode `Transfer` to `Event`")?;

    TRANSFERS
        .save(deps.storage, t.key, &t.data)
        .context("failed to save `transfer::Data`")?;

    Ok(evt)
}

// Carries out the transfer described by `t` and returns the updated source and destination
// accounts.
fn transfer<C: CustomQuery>(deps: Deps<C>, t: &Transfer) -> anyhow::Result<(Account, Account)> {
    if TRANSFERS.has(deps.storage, t.key.clone()) {
        bail!(TransferError::DuplicateTransfer);
    }

    let mut src = {
        let key = account::Key::new(
            t.key.emitter_chain(),
            t.data.token_chain,
            t.data.token_address,
        );
        let balance = match ACCOUNTS
            .may_load(deps.storage, key.clone())
            .context("failed to load source account")?
        {
            Some(s) => s,
            None => {
                ensure!(
                    key.chain_id() == key.token_chain(),
                    TransferError::MissingWrappedAccount
                );

                Balance::zero()
            }
        };
        Account { key, balance }
    };

    let mut dst = if t.key.emitter_chain() == t.data.recipient_chain {
        // This is a self-transfer so the source and destination accounts are the same.
        src.clone()
    } else {
        let key = account::Key::new(
            t.data.recipient_chain,
            t.data.token_chain,
            t.data.token_address,
        );
        let balance = match ACCOUNTS
            .may_load(deps.storage, key.clone())
            .context("failed to load destination account")?
        {
            Some(b) => b,
            None => {
                ensure!(
                    key.chain_id() != key.token_chain(),
                    TransferError::MissingNativeAccount,
                );

                Balance::zero()
            }
        };
        Account { key, balance }
    };

    src.lock_or_burn(t.data.amount)
        .context(TransferError::InsufficientSourceBalance)?;

    // If we're doing a transfer on the same chain then apply both changes to the same account.
    if src.key == dst.key {
        src.unlock_or_mint(t.data.amount)
            .context(TransferError::InsufficientDestinationBalance)?;
    } else {
        dst.unlock_or_mint(t.data.amount)
            .context(TransferError::InsufficientDestinationBalance)?;
    }

    Ok((src, dst))
}

#[derive(ThisError, Debug)]
pub enum ModifyBalanceError {
    #[error("modification already processed")]
    DuplicateModification,
    #[error("insufficient balance in account")]
    InsufficientBalance,
}

/// Modifies the on-chain balance of a single account.  If an error occurs that is not due to the
/// underlying cosmwasm framework, the returned error will be downcastable to `ModifyBalanceError`.
///
/// # Examples
///
/// ```
/// # fn example() {
/// #     use accountant::{
/// #         modify_balance,
/// #         state::{Kind, Modification},
/// #         ModifyBalanceError,
/// #     };
/// #     use cosmwasm_std::{testing::mock_dependencies, Uint256};
/// #     let mut deps = mock_dependencies();
/// #
///       /// Subtract the balance from an account that doesn't exist.
///       let m = Modification {
///           sequence: 0,
///           chain_id: 1,
///           token_chain: 2,
///           token_address: [3u8; 32].into(),
///           kind: Kind::Sub,
///           amount: Uint256::from(4u128),
///           reason: "test".try_into().unwrap(),
///       };
///  
///       let err = modify_balance(deps.as_mut(), m)
///           .expect_err("successfully modified account with insufficient balance");
///       if let Some(e) = err.downcast_ref::<ModifyBalanceError>() {
///           assert!(matches!(e, ModifyBalanceError::InsufficientBalance));
///       } else {
///           println!("framework error: {err:#}");
///       }
/// # }
/// #
/// # example();
/// ```
pub fn modify_balance<C: CustomQuery>(
    deps: DepsMut<C>,
    msg: Modification,
) -> anyhow::Result<Event> {
    if MODIFICATIONS.has(deps.storage, msg.sequence) {
        bail!(ModifyBalanceError::DuplicateModification);
    }

    let key = ACCOUNTS.key(account::Key::new(
        msg.chain_id,
        msg.token_chain,
        msg.token_address,
    ));

    let balance = key
        .may_load(deps.storage)
        .context("failed to load account")?
        .unwrap_or(Balance::zero());

    let new_balance = match msg.kind {
        Kind::Add => balance.checked_add(msg.amount),
        Kind::Sub => balance.checked_sub(msg.amount),
    }
    .map(Balance::from)
    .context(ModifyBalanceError::InsufficientBalance)?;

    key.save(deps.storage, &new_balance)
        .context("failed to save account")?;

    MODIFICATIONS
        .save(deps.storage, msg.sequence, &msg)
        .context("failed to store `Modification`")?;

    cw_transcode::to_event(&msg).context("failed to transcode `Modification` to `Event`")
}

/// Query the balance for the account associated with `key`.
pub fn query_balance<C: CustomQuery>(deps: Deps<C>, key: account::Key) -> StdResult<Balance> {
    ACCOUNTS.load(deps.storage, key)
}

/// Query information for all accounts.
pub fn query_all_accounts<C: CustomQuery>(
    deps: Deps<C>,
    start_after: Option<account::Key>,
) -> impl Iterator<Item = StdResult<Account>> + '_ {
    let start = start_after.map(|key| Bound::Exclusive((key, PhantomData)));

    ACCOUNTS
        .range(deps.storage, start, None, Order::Ascending)
        .map(|item| item.map(|(key, balance)| Account { key, balance }))
}

/// Query the data associated with a transfer.
pub fn query_transfer<C: CustomQuery>(
    deps: Deps<C>,
    key: transfer::Key,
) -> StdResult<transfer::Data> {
    TRANSFERS.load(deps.storage, key)
}

/// Check if a transfer with the associated key exists.
pub fn has_transfer<C: CustomQuery>(deps: Deps<C>, key: transfer::Key) -> bool {
    TRANSFERS.has(deps.storage, key)
}

/// Query information for all transfers.
pub fn query_all_transfers<C: CustomQuery>(
    deps: Deps<C>,
    start_after: Option<transfer::Key>,
) -> impl Iterator<Item = StdResult<Transfer>> + '_ {
    let start = start_after.map(|key| Bound::Exclusive((key, PhantomData)));

    TRANSFERS
        .range(deps.storage, start, None, Order::Ascending)
        .map(|item| item.map(|(key, data)| Transfer { key, data }))
}

/// Query the data associated with a modification.
pub fn query_modification<C: CustomQuery>(deps: Deps<C>, sequence: u64) -> StdResult<Modification> {
    MODIFICATIONS.load(deps.storage, sequence)
}

/// Query information for all modifications.
pub fn query_all_modifications<C: CustomQuery>(
    deps: Deps<C>,
    start_after: Option<u64>,
) -> impl Iterator<Item = StdResult<Modification>> + '_ {
    let start = start_after.map(|seq| Bound::Exclusive((seq, PhantomData)));

    MODIFICATIONS
        .range(deps.storage, start, None, Order::Ascending)
        .map(|item| item.map(|(_, v)| v))
}

#[cfg(test)]
mod tests {
    use std::collections::BTreeMap;

    use cosmwasm_std::{testing::mock_dependencies, StdError, Uint256};

    use super::*;

    fn create_accounts(count: usize) -> Vec<Account> {
        let mut out = Vec::with_capacity(count * count);
        for i in 0..count {
            for j in 0..count {
                let key = account::Key::new(i as u16, j as u16, [i as u8; 32].into());
                let balance = Uint256::from(j as u128).into();
                out.push(Account { key, balance });
            }
        }

        out
    }

    fn save_accounts(deps: DepsMut, accounts: &[Account]) {
        for a in accounts {
            ACCOUNTS
                .save(deps.storage, a.key.clone(), &a.balance)
                .unwrap();
        }
    }

    fn create_transfers(count: usize) -> Vec<Transfer> {
        let mut out = Vec::with_capacity(count);
        for i in 0..count {
            let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
            let data = transfer::Data {
                amount: Uint256::from(i as u128),
                token_chain: i as u16,
                token_address: [i as u8; 32].into(),
                recipient_chain: i as u16,
            };

            out.push(Transfer { key, data });
        }

        out
    }

    fn save_transfers(deps: DepsMut, transfers: &[Transfer]) {
        for t in transfers {
            TRANSFERS
                .save(deps.storage, t.key.clone(), &t.data)
                .unwrap();
        }
    }

    fn create_modifications(count: usize) -> Vec<Modification> {
        let mut out = Vec::with_capacity(count);
        for i in 0..count {
            let m = Modification {
                sequence: i as u64,
                chain_id: i as u16,
                token_chain: i as u16,
                token_address: [i as u8; 32].into(),
                kind: if i % 2 == 0 { Kind::Add } else { Kind::Sub },
                amount: Uint256::from(i as u128),
                reason: "test".into(),
            };
            out.push(m);
        }

        out
    }

    fn save_modifications(deps: DepsMut, modifications: &[Modification]) {
        for m in modifications {
            MODIFICATIONS.save(deps.storage, m.sequence, m).unwrap();
        }
    }

    #[test]
    fn instantiate_accountant() {
        let mut deps = mock_dependencies();
        let count = 3;
        let msg = Instantiate {
            accounts: create_accounts(count),
            transfers: create_transfers(count),
            modifications: create_modifications(count),
        };

        instantiate(deps.as_mut(), msg.clone()).unwrap();

        for a in msg.accounts {
            assert_eq!(a.balance, query_balance(deps.as_ref(), a.key).unwrap());
        }

        for t in msg.transfers {
            assert_eq!(t.data, query_transfer(deps.as_ref(), t.key).unwrap());
        }

        for m in msg.modifications {
            assert_eq!(m, query_modification(deps.as_ref(), m.sequence).unwrap());
        }
    }

    #[test]
    fn simple_transfer() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        let evt = commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        let src = account::Key::new(
            tx.key.emitter_chain(),
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(tx.data.amount, *query_balance(deps.as_ref(), src).unwrap());

        let dst = account::Key::new(
            tx.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(tx.data.amount, *query_balance(deps.as_ref(), dst).unwrap());

        let expected = Event::new("Transfer")
            .add_attribute("key", serde_json_wasm::to_string(&tx.key).unwrap())
            .add_attribute("data", serde_json_wasm::to_string(&tx.data).unwrap());
        assert_eq!(expected, evt);

        assert_eq!(tx.data, query_transfer(deps.as_ref(), tx.key).unwrap());
    }

    #[test]
    fn local_native_transfer() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 3,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        // Since we transfered within the same chain, the balance should be 0.
        let src = account::Key::new(
            tx.key.emitter_chain(),
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(Balance::zero(), query_balance(deps.as_ref(), src).unwrap());

        let dst = account::Key::new(
            tx.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(Balance::zero(), query_balance(deps.as_ref(), dst).unwrap());

        assert_eq!(tx.data, query_transfer(deps.as_ref(), tx.key).unwrap());
    }

    #[test]
    fn local_wrapped_transfer() {
        let mut deps = mock_dependencies();
        let mut tx = Transfer {
            key: transfer::Key::new(9, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        // A local wrapped transfer is only allowed if we've already issued wrapped tokens.
        let e = validate_transfer(deps.as_ref(), &tx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::MissingWrappedAccount
        ));

        let e = commit_transfer(deps.as_mut(), tx.clone())
            .expect_err("successfully committed duplicate transfer");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::MissingWrappedAccount
        ));

        // Issue some wrapped tokens.
        let wrapped = tx.data.amount.checked_div(Uint256::from(2u128)).unwrap();
        let m = Modification {
            sequence: 1,
            chain_id: tx.key.emitter_chain(),
            token_chain: tx.data.token_chain,
            token_address: tx.data.token_address,
            kind: Kind::Add,
            amount: wrapped,
            reason: "test".into(),
        };
        modify_balance(deps.as_mut(), m).unwrap();

        // The transfer should still fail because we're trying to move more wrapped tokens than
        // were issued.
        let e = validate_transfer(deps.as_ref(), &tx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::InsufficientSourceBalance
        ));

        let e = commit_transfer(deps.as_mut(), tx.clone())
            .expect_err("successfully committed duplicate transfer");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::InsufficientSourceBalance
        ));

        // Lower the amount in the transfer.
        tx.data.amount = wrapped;

        // Now the transfer should be fine.
        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        // The balance should not have changed.
        let src = account::Key::new(
            tx.key.emitter_chain(),
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(wrapped, *query_balance(deps.as_ref(), src).unwrap());

        let dst = account::Key::new(
            tx.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(wrapped, *query_balance(deps.as_ref(), dst).unwrap());

        assert_eq!(tx.data, query_transfer(deps.as_ref(), tx.key).unwrap());
    }

    #[test]
    fn duplicate_transfer() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        // Repeating the transfer should return an error and should not change the balances.
        let e = validate_transfer(deps.as_ref(), &tx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::DuplicateTransfer
        ));

        let e = commit_transfer(deps.as_mut(), tx.clone())
            .expect_err("successfully committed duplicate transfer");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::DuplicateTransfer
        ));

        let src = account::Key::new(
            tx.key.emitter_chain(),
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(tx.data.amount, *query_balance(deps.as_ref(), src).unwrap());

        let dst = account::Key::new(
            tx.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(tx.data.amount, *query_balance(deps.as_ref(), dst).unwrap());

        assert_eq!(tx.data, query_transfer(deps.as_ref(), tx.key).unwrap());
    }

    #[test]
    fn round_trip_transfer() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        let rx = Transfer {
            key: transfer::Key::new(tx.data.recipient_chain, [6u8; 32].into(), 2),
            data: transfer::Data {
                amount: tx.data.amount,
                token_chain: tx.data.token_chain,
                token_address: tx.data.token_address,
                recipient_chain: tx.key.emitter_chain(),
            },
        };

        validate_transfer(deps.as_ref(), &rx).unwrap();
        commit_transfer(deps.as_mut(), rx.clone()).unwrap();

        let src = account::Key::new(
            tx.key.emitter_chain(),
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(Balance::zero(), query_balance(deps.as_ref(), src).unwrap());

        let dst = account::Key::new(
            tx.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(Balance::zero(), query_balance(deps.as_ref(), dst).unwrap());

        assert_eq!(tx.data, query_transfer(deps.as_ref(), tx.key).unwrap());
        assert_eq!(rx.data, query_transfer(deps.as_ref(), rx.key).unwrap());
    }

    #[test]
    fn missing_wrapped_account() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(9, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 3,
            },
        };

        let e = validate_transfer(deps.as_ref(), &tx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::MissingWrappedAccount
        ));
        let e = commit_transfer(deps.as_mut(), tx)
            .expect_err("successfully committed transfer with missing wrapped account");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::MissingWrappedAccount
        ));
    }

    #[test]
    fn missing_native_account() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(9, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 3,
            },
        };

        // Set up a fake wrapped account so the check for the wrapped account succeeds.
        ACCOUNTS
            .save(
                &mut deps.storage,
                account::Key::new(
                    tx.key.emitter_chain(),
                    tx.data.token_chain,
                    tx.data.token_address,
                ),
                &tx.data.amount.into(),
            )
            .unwrap();

        let e = validate_transfer(deps.as_ref(), &tx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::MissingNativeAccount
        ));

        let e = commit_transfer(deps.as_mut(), tx)
            .expect_err("successfully committed transfer with missing native account");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::MissingNativeAccount
        ));
    }

    #[test]
    fn repeated_transfer() {
        const ITERATIONS: usize = 10;
        let mut deps = mock_dependencies();
        let emitter_chain = 3;
        let emitter_address = [3u8; 32].into();
        let data = transfer::Data {
            amount: Uint256::from(400u128),
            token_chain: 3,
            token_address: [3u8; 32].into(),
            recipient_chain: 9,
        };

        for i in 0..ITERATIONS {
            let tx = Transfer {
                key: transfer::Key::new(emitter_chain, emitter_address, i as u64),
                data: data.clone(),
            };

            validate_transfer(deps.as_ref(), &tx).unwrap();
            commit_transfer(deps.as_mut(), tx).unwrap();
        }

        let src = account::Key::new(emitter_chain, data.token_chain, data.token_address);
        assert_eq!(
            data.amount * Uint256::from(ITERATIONS as u128),
            *query_balance(deps.as_ref(), src).unwrap()
        );

        let dst = account::Key::new(data.recipient_chain, data.token_chain, data.token_address);
        assert_eq!(
            data.amount * Uint256::from(ITERATIONS as u128),
            *query_balance(deps.as_ref(), dst).unwrap()
        );
    }

    #[test]
    fn wrapped_transfer() {
        let mut deps = mock_dependencies();

        // Do an initial simple transfer.
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        // Now transfer some of the wrapped tokens to a new chain.
        let wrapped = Transfer {
            key: transfer::Key::new(tx.data.recipient_chain, [2u8; 32].into(), 9),
            data: transfer::Data {
                amount: Uint256::from(200u128),
                token_chain: tx.data.token_chain,
                token_address: tx.data.token_address,
                recipient_chain: 11,
            },
        };

        validate_transfer(deps.as_ref(), &wrapped).unwrap();
        commit_transfer(deps.as_mut(), wrapped.clone()).unwrap();

        // The balance on the original chain should not have changed.
        let src = account::Key::new(
            tx.key.emitter_chain(),
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(tx.data.amount, *query_balance(deps.as_ref(), src).unwrap());

        // The destination chain should have the difference between the two transfers.
        let dst = account::Key::new(
            tx.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(
            tx.data.amount - wrapped.data.amount,
            *query_balance(deps.as_ref(), dst).unwrap()
        );

        // The third chain should only have the wrapped amount.
        let w = account::Key::new(
            wrapped.data.recipient_chain,
            tx.data.token_chain,
            tx.data.token_address,
        );
        assert_eq!(
            wrapped.data.amount,
            *query_balance(deps.as_ref(), w).unwrap()
        );
    }

    #[test]
    fn insufficient_wrapped_balance() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        // Now try to transfer back more tokens than were originally sent.
        let rx = Transfer {
            key: transfer::Key::new(tx.data.recipient_chain, [6u8; 32].into(), 2),
            data: transfer::Data {
                amount: tx.data.amount * Uint256::from(2u128),
                token_chain: tx.data.token_chain,
                token_address: tx.data.token_address,
                recipient_chain: tx.key.emitter_chain(),
            },
        };

        let e = validate_transfer(deps.as_ref(), &rx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::InsufficientSourceBalance
        ));

        let e = commit_transfer(deps.as_mut(), rx)
            .expect_err("successfully transferred more tokens than available");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::InsufficientSourceBalance
        ));
    }

    #[test]
    fn insufficient_native_balance() {
        let mut deps = mock_dependencies();
        let tx = Transfer {
            key: transfer::Key::new(3, [1u8; 32].into(), 5),
            data: transfer::Data {
                amount: Uint256::from(400u128),
                token_chain: 3,
                token_address: [3u8; 32].into(),
                recipient_chain: 9,
            },
        };

        validate_transfer(deps.as_ref(), &tx).unwrap();
        commit_transfer(deps.as_mut(), tx.clone()).unwrap();

        // Artificially increase the wrapped balance so that the check for wrapped tokens passes.
        ACCOUNTS
            .update(
                &mut deps.storage,
                account::Key::new(
                    tx.data.recipient_chain,
                    tx.data.token_chain,
                    tx.data.token_address,
                ),
                |b| {
                    b.unwrap()
                        .checked_mul(Uint256::from(2u128))
                        .map(From::from)
                        .map_err(|source| StdError::Overflow { source })
                },
            )
            .unwrap();

        // Now try to transfer back more tokens than were originally sent.
        let rx = Transfer {
            key: transfer::Key::new(tx.data.recipient_chain, [6u8; 32].into(), 2),
            data: transfer::Data {
                amount: tx.data.amount * Uint256::from(2u128),
                token_chain: tx.data.token_chain,
                token_address: tx.data.token_address,
                recipient_chain: tx.key.emitter_chain(),
            },
        };

        let e = validate_transfer(deps.as_ref(), &rx).unwrap_err();
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::InsufficientDestinationBalance
        ));

        let e = commit_transfer(deps.as_mut(), rx)
            .expect_err("successfully transferred more tokens than available");
        assert!(matches!(
            e.downcast().unwrap(),
            TransferError::InsufficientDestinationBalance
        ));
    }

    #[test]
    fn simple_modify() {
        let mut deps = mock_dependencies();
        let m = Modification {
            sequence: 0,
            chain_id: 1,
            token_chain: 2,
            token_address: [3u8; 32].into(),
            kind: Kind::Add,
            amount: Uint256::from(4u128),
            reason: "test".into(),
        };

        let evt = modify_balance(deps.as_mut(), m.clone()).unwrap();

        let acc = account::Key::new(m.chain_id, m.token_chain, m.token_address);
        assert_eq!(m.amount, *query_balance(deps.as_ref(), acc).unwrap());

        assert_eq!(m, query_modification(deps.as_ref(), m.sequence).unwrap());

        let expected = Event::new("Modification")
            .add_attribute("sequence", serde_json_wasm::to_string(&m.sequence).unwrap())
            .add_attribute("chain_id", serde_json_wasm::to_string(&m.chain_id).unwrap())
            .add_attribute(
                "token_chain",
                serde_json_wasm::to_string(&m.token_chain).unwrap(),
            )
            .add_attribute(
                "token_address",
                serde_json_wasm::to_string(&m.token_address).unwrap(),
            )
            .add_attribute("kind", serde_json_wasm::to_string(&m.kind).unwrap())
            .add_attribute("amount", serde_json_wasm::to_string(&m.amount).unwrap())
            .add_attribute("reason", serde_json_wasm::to_string(&m.reason).unwrap());
        assert_eq!(expected, evt);
    }

    #[test]
    fn duplicate_modify() {
        let mut deps = mock_dependencies();
        let m = Modification {
            sequence: 0,
            chain_id: 1,
            token_chain: 2,
            token_address: [3u8; 32].into(),
            kind: Kind::Add,
            amount: Uint256::from(4u128),
            reason: "test".into(),
        };

        modify_balance(deps.as_mut(), m.clone()).unwrap();

        // Trying the same modification again should fail.
        let e = modify_balance(deps.as_mut(), m).expect_err("successfully modified balance twice");
        assert!(matches!(
            e.downcast().unwrap(),
            ModifyBalanceError::DuplicateModification
        ));
    }

    #[test]
    fn round_trip_modify() {
        let mut deps = mock_dependencies();
        let mut m = Modification {
            sequence: 0,
            chain_id: 1,
            token_chain: 2,
            token_address: [3u8; 32].into(),
            kind: Kind::Add,
            amount: Uint256::from(4u128),
            reason: "test".into(),
        };

        modify_balance(deps.as_mut(), m.clone()).unwrap();

        m.sequence += 1;
        m.kind = Kind::Sub;
        modify_balance(deps.as_mut(), m.clone()).unwrap();

        let acc = account::Key::new(m.chain_id, m.token_chain, m.token_address);
        assert_eq!(Balance::zero(), query_balance(deps.as_ref(), acc).unwrap());
    }

    #[test]
    fn repeated_modify() {
        const ITERATIONS: u64 = 10;
        let mut deps = mock_dependencies();
        let mut m = Modification {
            sequence: 0,
            chain_id: 1,
            token_chain: 2,
            token_address: [3u8; 32].into(),
            kind: Kind::Add,
            amount: Uint256::from(4u128),
            reason: "test".into(),
        };

        for i in 0..ITERATIONS {
            m.sequence = i;
            modify_balance(deps.as_mut(), m.clone()).unwrap();
        }

        let acc = account::Key::new(m.chain_id, m.token_chain, m.token_address);
        assert_eq!(
            m.amount * Uint256::from(ITERATIONS),
            *query_balance(deps.as_ref(), acc).unwrap()
        );
    }

    #[test]
    fn modify_insufficient_balance() {
        let mut deps = mock_dependencies();
        let m = Modification {
            sequence: 0,
            chain_id: 1,
            token_chain: 2,
            token_address: [3u8; 32].into(),
            kind: Kind::Sub,
            amount: Uint256::from(4u128),
            reason: "test".into(),
        };

        let e = modify_balance(deps.as_mut(), m)
            .expect_err("successfully modified account with insufficient balance");
        assert!(matches!(
            e.downcast().unwrap(),
            ModifyBalanceError::InsufficientBalance
        ));
    }

    #[test]
    fn query_account_balance() {
        let mut deps = mock_dependencies();
        let count = 2;
        save_accounts(deps.as_mut(), &create_accounts(count));

        for i in 0..count {
            for j in 0..count {
                let key = account::Key::new(i as u16, j as u16, [i as u8; 32].into());
                let balance = query_balance(deps.as_ref(), key).unwrap();
                assert_eq!(balance, Balance::new(Uint256::from(j as u128)))
            }
        }
    }

    #[test]
    fn query_missing_account() {
        let mut deps = mock_dependencies();
        let count = 2;
        save_accounts(deps.as_mut(), &create_accounts(count));

        let missing = account::Key::new(
            (count + 1) as u16,
            (count + 2) as u16,
            [(count + 3) as u8; 32].into(),
        );

        query_balance(deps.as_ref(), missing)
            .expect_err("successfully queried missing account key");
    }

    #[test]
    fn query_all_balances() {
        let mut deps = mock_dependencies();
        let count = 3;
        save_accounts(deps.as_mut(), &create_accounts(count));

        let found = query_all_accounts(deps.as_ref(), None)
            .map(|item| item.map(|acc| (acc.key, acc.balance)))
            .collect::<StdResult<BTreeMap<_, _>>>()
            .unwrap();
        assert_eq!(found.len(), count * count);

        for i in 0..count {
            for j in 0..count {
                let key = account::Key::new(i as u16, j as u16, [i as u8; 32].into());
                assert!(found.contains_key(&key));
            }
        }
    }

    #[test]
    fn query_all_balances_start_after() {
        let mut deps = mock_dependencies();
        let count = 3;
        save_accounts(deps.as_mut(), &create_accounts(count));

        for i in 0..count {
            for j in 0..count {
                let start_after = Some(account::Key::new(i as u16, j as u16, [i as u8; 32].into()));
                let found = query_all_accounts(deps.as_ref(), start_after)
                    .map(|item| item.map(|acc| (acc.key, acc.balance)))
                    .collect::<StdResult<BTreeMap<_, _>>>()
                    .unwrap();
                assert_eq!(found.len(), (count - i - 1) * count + (count - j - 1),);

                for y in j + 1..count {
                    let key = account::Key::new(i as u16, y as u16, [i as u8; 32].into());
                    assert!(found.contains_key(&key));
                }

                for x in i + 1..count {
                    for y in 0..count {
                        let key = account::Key::new(x as u16, y as u16, [x as u8; 32].into());
                        assert!(found.contains_key(&key));
                    }
                }
            }
        }
    }

    #[test]
    fn query_transfer_data() {
        let mut deps = mock_dependencies();
        let count = 2;
        save_transfers(deps.as_mut(), &create_transfers(count));

        for i in 0..count {
            let expected = transfer::Data {
                amount: Uint256::from(i as u128),
                token_chain: i as u16,
                token_address: [i as u8; 32].into(),
                recipient_chain: i as u16,
            };

            let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
            let actual = query_transfer(deps.as_ref(), key).unwrap();

            assert_eq!(expected, actual);
        }
    }

    #[test]
    fn query_missing_transfer() {
        let mut deps = mock_dependencies();
        let count = 2;
        save_transfers(deps.as_mut(), &create_transfers(count));

        let missing = transfer::Key::new(
            (count + 1) as u16,
            [(count + 2) as u8; 32].into(),
            (count + 3) as u64,
        );

        query_transfer(deps.as_ref(), missing)
            .expect_err("successfully queried missing transfer key");
    }

    #[test]
    fn query_all_transfer_data() {
        let mut deps = mock_dependencies();
        let count = 3;
        save_transfers(deps.as_mut(), &create_transfers(count));

        let found = query_all_transfers(deps.as_ref(), None)
            .map(|item| item.map(|acc| (acc.key, acc.data)))
            .collect::<StdResult<BTreeMap<_, _>>>()
            .unwrap();
        assert_eq!(found.len(), count);

        for i in 0..count {
            let key = transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64);
            assert!(found.contains_key(&key));
        }
    }

    #[test]
    fn query_all_transfer_data_start_after() {
        let mut deps = mock_dependencies();
        let count = 3;
        save_transfers(deps.as_mut(), &create_transfers(count));

        for i in 0..count {
            let start_after = Some(transfer::Key::new(i as u16, [i as u8; 32].into(), i as u64));
            let found = query_all_transfers(deps.as_ref(), start_after)
                .map(|item| item.map(|acc| (acc.key, acc.data)))
                .collect::<StdResult<BTreeMap<_, _>>>()
                .unwrap();
            assert_eq!(found.len(), count - i - 1);

            for x in i + 1..count {
                let key = transfer::Key::new(x as u16, [x as u8; 32].into(), x as u64);
                assert!(found.contains_key(&key));
            }
        }
    }

    #[test]
    fn query_modification_data() {
        let mut deps = mock_dependencies();
        let count = 2;
        save_modifications(deps.as_mut(), &create_modifications(count));

        for i in 0..count {
            let expected = Modification {
                sequence: i as u64,
                chain_id: i as u16,
                token_chain: i as u16,
                token_address: [i as u8; 32].into(),
                kind: if i % 2 == 0 { Kind::Add } else { Kind::Sub },
                amount: Uint256::from(i as u128),
                reason: "test".into(),
            };

            let key = i as u64;
            let actual = query_modification(deps.as_ref(), key).unwrap();

            assert_eq!(expected, actual);
        }
    }

    #[test]
    fn query_missing_modification() {
        let mut deps = mock_dependencies();
        let count = 2;
        save_modifications(deps.as_mut(), &create_modifications(count));

        let missing = (count + 1) as u64;

        query_modification(deps.as_ref(), missing)
            .expect_err("successfully queried missing modification key");
    }

    #[test]
    fn query_all_modification_data() {
        let mut deps = mock_dependencies();
        let count = 3;
        save_modifications(deps.as_mut(), &create_modifications(count));

        let found = query_all_modifications(deps.as_ref(), None)
            .map(|item| item.map(|m| (m.sequence, m)))
            .collect::<StdResult<BTreeMap<_, _>>>()
            .unwrap();
        assert_eq!(found.len(), count);

        for i in 0..count {
            let key = i as u64;
            assert!(found.contains_key(&key));
        }
    }

    #[test]
    fn query_all_modification_data_start_after() {
        let mut deps = mock_dependencies();
        let count = 3;
        save_modifications(deps.as_mut(), &create_modifications(count));

        for i in 0..count {
            let start_after = Some(i as u64);
            let found = query_all_modifications(deps.as_ref(), start_after)
                .map(|item| item.map(|m| (m.sequence, m)))
                .collect::<StdResult<BTreeMap<_, _>>>()
                .unwrap();
            assert_eq!(found.len(), count - i - 1);

            for x in i + 1..count {
                let key = x as u64;
                assert!(found.contains_key(&key));
            }
        }
    }
}
