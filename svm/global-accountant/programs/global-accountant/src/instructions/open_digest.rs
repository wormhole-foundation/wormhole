//! Shared DigestAccount-open implementation — not yet implemented.
//!
//! Will be called by `submit_observations` (on quorum) and later by
//! `submit_vaas` (after Shim verification).

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::GlobalAccountantError;
use crate::err;

/// Would allocate + write a `DigestAccountLayout` at the canonical
/// `(b"digest", chain, emitter, sequence)` PDA (bump derived on-chain, so a
/// non-canonical address is impossible), stamping emitter, digest, payer,
/// sequence, guardian-set index, and the current slot.
#[allow(clippy::too_many_arguments)]
pub(crate) fn open_digest_inner(
    _program_id: &Address,
    _payer: &AccountView,
    _digest_pda: &mut AccountView,
    _chain_be: [u8; 2],
    _emitter: [u8; 32],
    _sequence_be: [u8; 8],
    _digest_bytes: [u8; 32],
    _guardian_set_index: u32,
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
