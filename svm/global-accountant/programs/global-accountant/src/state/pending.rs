//! Zero-copy load/store helpers for `PendingObservationsLayout` — not yet
//! implemented.

use pinocchio::{error::ProgramError, AccountView};

use crate::definitions::{GlobalAccountantError, PendingObservationsLayout};
use crate::err;

/// Would read a [`PendingObservationsLayout`] from the account's data buffer,
/// copying out by value (the layout is `Pod`) to release the borrow before
/// the caller mutates; `InvalidPda` if the buffer is not exactly `LEN`.
pub fn load(_account: &AccountView) -> Result<PendingObservationsLayout, ProgramError> {
    Err(err(GlobalAccountantError::NotImplemented))
}

/// Would write a [`PendingObservationsLayout`] into the account's data buffer;
/// `InvalidPda` if the buffer is not exactly `LEN`.
pub fn store(
    _account: &mut AccountView,
    _value: &PendingObservationsLayout,
) -> Result<(), ProgramError> {
    Err(err(GlobalAccountantError::NotImplemented))
}
