//! `modify_balance` — Accountant governance handler.
//!
//! Stub: reserved in the dispatch table and `Instruction` discriminant set;
//! returns `NotImplemented` until a dedicated PR ports the handler. Planned
//! behavior: apply a manual Add / Subtract delta to a `BalanceAccount` PDA via a
//! Wormchain-emitted governance VAA, with per-sequence replay protection via a
//! `ModificationLog` PDA.

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
