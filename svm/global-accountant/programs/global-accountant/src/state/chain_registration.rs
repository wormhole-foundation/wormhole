//! Chain-registration cross-check for the submit paths — not yet implemented.
//!
//! Read-only on the submit paths; written only by `register_chain` (which
//! lands in a later commit together with load/store helpers).

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::GlobalAccountantError;
use crate::err;

/// Would verify that the account lives at the canonical
/// `(b"chain_registration", chain_be)` PDA (rejecting a foreign account
/// masquerading as the registration PDA with `InvalidPda`), that a
/// registration exists (system-owned ⇒ `MissingChainRegistration`), and that
/// the on-disk `emitter_address` equals the body header's emitter
/// (`UnregisteredEmitter` otherwise).
pub fn verify(
    _program_id: &Address,
    _registration_pda: &AccountView,
    _body_chain: u16,
    _body_emitter: &[u8; 32],
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
