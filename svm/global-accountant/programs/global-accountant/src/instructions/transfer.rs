//! Balance-mutation helpers — not yet implemented.
//!
//! Will serve the quorum-completing branch of `submit_observations` and later
//! the signed-VAA backfill in `submit_vaas`.

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::{GlobalAccountantError, Uint256};
use crate::err;

/// Would verify both Account PDAs against their canonical
/// `(b"account", chain_be, token_chain_be, token_address)` derivations
/// (rejecting mismatches as `InvalidAccountPda`), lazy-init them with the
/// payer funding rent, then apply source-side `lock_or_burn` and
/// destination-side `unlock_or_mint`; same-chain self-transfers collapse onto
/// one in-memory layout so the second mutation observes the first.
#[allow(clippy::too_many_arguments)]
pub fn apply_transfer(
    _program_id: &Address,
    _payer: &AccountView,
    _source_account: &mut AccountView,
    _dest_account: &mut AccountView,
    _source_chain: u16,
    _recipient_chain: u16,
    _token_chain: u16,
    _token_address: &[u8; 32],
    _amount: Uint256,
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
