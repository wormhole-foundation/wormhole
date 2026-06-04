//! `close_digest` — not yet implemented.
//!
//! Would be a permissionless close gated by a VAA whose digest matches the one
//! stored in the PDA, refunding rent to the recorded payer and verifying the
//! digest via a CPI to the Wormhole Verify VAA Shim.

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
