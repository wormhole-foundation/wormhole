//! Shared PDA initialisation helper — not yet implemented.

use pinocchio::{cpi::Signer, AccountView, Address, ProgramResult};

use crate::definitions::GlobalAccountantError;
use crate::err;

/// Would create the PDA with `space` bytes owned by `program_id`, defending
/// against the dust-DoS grief vector (an attacker pre-funding the address so a
/// naive `CreateAccount` fails): empty + zero-lamport ⇒ `CreateAccount`;
/// pre-funded but system-owned and data-empty ⇒ Transfer top-up + Allocate +
/// Assign; anything else ⇒ `InvalidPda`.
pub fn init_or_upgrade_pda(
    _payer: &AccountView,
    _pda: &AccountView,
    _program_id: &Address,
    _signer: Signer,
    _space: u64,
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
