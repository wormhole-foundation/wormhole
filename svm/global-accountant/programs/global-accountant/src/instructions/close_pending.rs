//! `close_pending` — permissionless cleanup of stranded
//! `PendingObservationsLayout` PDAs.
//!
//! Stub: reserved in the dispatch table and `Instruction` discriminant set;
//! returns `NotImplemented` until a dedicated PR ports the handler. Planned
//! behavior: anyone may close a pending PDA when either trigger holds, refunding
//! lamports to the recorded `payer` — (a) the recorded guardian set has expired,
//! or (b) NoReplay is already marked for `(chain, emitter, sequence)`.

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
