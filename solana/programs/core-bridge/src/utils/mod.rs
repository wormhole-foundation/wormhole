//! Utilities for the Core Bridge Program.

use anchor_lang::{prelude::*, system_program};

/// Compute quorum based on the number of guardians in a guardian set.
#[inline]
pub fn quorum(num_guardians: usize) -> usize {
    (2 * num_guardians) / 3 + 1
}

/// Close an account by transferring all its lamports to another account.
pub fn close_account<'info>(
    info: AccountInfo<'info>,
    sol_destination: AccountInfo<'info>,
) -> Result<()> {
    // Transfer tokens from the account to the sol_destination.
    let dest_starting_lamports = sol_destination.lamports();
    **sol_destination.lamports.borrow_mut() =
        dest_starting_lamports.checked_add(info.lamports()).unwrap();
    **info.lamports.borrow_mut() = 0;

    info.assign(&solana_program::system_program::ID);
    info.realloc(0, false).map_err(Into::into)
}

/// Trait for invoking the system program's create account instruction.
pub trait CreateAccount<'info> {
    fn system_program(&self) -> AccountInfo<'info>;

    /// Signer that has the lamports to transfer to new account.
    fn payer(&self) -> AccountInfo<'info>;
}

pub fn create_account<'info, A>(
    accounts: &A,
    new_account: AccountInfo<'info>,
    data_len: usize,
    owner: &Pubkey,
    signer_seeds: Option<&[&[&[u8]]]>,
) -> Result<()>
where
    A: CreateAccount<'info>,
{
    match signer_seeds {
        Some(signer_seeds) => system_program::create_account(
            CpiContext::new_with_signer(
                accounts.system_program(),
                system_program::CreateAccount {
                    from: accounts.payer(),
                    to: new_account,
                },
                signer_seeds,
            ),
            Rent::get().map(|rent| rent.minimum_balance(data_len))?,
            data_len.try_into().unwrap(),
            owner,
        ),
        None => system_program::create_account(
            CpiContext::new(
                accounts.system_program(),
                system_program::CreateAccount {
                    from: accounts.payer(),
                    to: new_account,
                },
            ),
            Rent::get().map(|rent| rent.minimum_balance(data_len))?,
            data_len.try_into().unwrap(),
            owner,
        ),
    }
}
