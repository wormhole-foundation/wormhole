//! `close_pending` — not yet implemented.
//!
//! Would be a permissionless cleanup of stranded `PendingObservationsLayout`
//! PDAs, refunding lamports to the recorded `payer` when either trigger holds:
//! the recorded guardian set has expired, or NoReplay is already marked for
//! `(chain, emitter, sequence)` (the entry was accounted via another path).

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::GlobalAccountantError;
use crate::err;

pub fn process(
    _program_id: &Address,
    _accounts: &mut [AccountView],
    _data: &[u8],
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
