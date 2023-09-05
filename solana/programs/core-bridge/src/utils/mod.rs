//! Utilities for the Core Bridge Program.

mod vaa;
pub use vaa::*;

use anchor_lang::{prelude::AccountInfo, Result};

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
