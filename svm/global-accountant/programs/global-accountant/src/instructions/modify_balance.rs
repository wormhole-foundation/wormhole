//! `modify_balance` — not yet implemented.
//!
//! Would be the Accountant governance handler: applies a manual Add/Subtract
//! delta to a `BalanceAccount` PDA via a Wormchain-emitted governance VAA, for
//! post-incident ledger reconciliation. Replay protection keys on the payload
//! `sequence` via a per-sequence `ModificationLog` PDA.

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
