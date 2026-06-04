//! `submit_vaas` — not yet implemented.
//!
//! Would be the permissionless signed-VAA backfill: consumes a fully-signed
//! VAA via the Verify VAA Shim CPI and applies its balance effects directly,
//! bypassing the quorum tracker. Shares only NoReplay state with
//! `submit_observations`: once `(chain, emitter, sequence)` is marked, any
//! later caller on either path is rejected as `AlreadyAccounted`.

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
