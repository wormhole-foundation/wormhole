//! `submit_vaas` — permissionless signed-VAA backfill.
//!
//! Stub: reserved in the dispatch table and `Instruction` discriminant set;
//! returns `NotImplemented` until a dedicated PR ports the handler. Planned
//! behavior: consume a fully-signed VAA via the Verify VAA Shim CPI and apply
//! its balance effects directly, bypassing the quorum tracker. Shares NoReplay
//! state with `submit_observations`.

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
