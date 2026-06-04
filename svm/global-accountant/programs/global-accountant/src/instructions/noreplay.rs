//! NoReplay integration — not yet implemented.
//!
//! Will provide the direct-read pre-check (`is_marked`) plus the `MarkUsed`
//! CPI into `solana-noreplay`, with the program's `[b"noreplay-authority"]`
//! PDA as the CPI authority so the bitmap is write-controlled exclusively by
//! this program.

use pinocchio::{error::ProgramError, AccountView, Address, ProgramResult};

use crate::definitions::GlobalAccountantError;
use crate::err;

/// Would re-derive the canonical noreplay bitmap PDA for
/// `(authority, chain_be ‖ emitter, sequence / 1024)`, reject a non-canonical
/// account as `InvalidPda`, and report whether bit `sequence % 1024` is set:
/// `Ok(true)` ⇒ already accounted; an uninitialised (system-owned) bucket is
/// the normal first-message case and reads as `Ok(false)`.
pub fn is_marked(
    _bucket: &AccountView,
    _noreplay_authority: &Address,
    _chain: u16,
    _emitter: &[u8; 32],
    _sequence: u64,
) -> Result<bool, ProgramError> {
    Err(err(GlobalAccountantError::NotImplemented))
}

/// Would CPI `MarkUsed` into the hardcoded `solana-noreplay` program (never
/// the caller-supplied account, which would let an attacker fake success),
/// `invoke_signed` by the canonical `[b"noreplay-authority"]` PDA, setting bit
/// `sequence % 1024` in the bitmap PDA. Any CPI failure would map to
/// `NoReplayCpiFailed`, distinct from the pre-check's `AlreadyAccounted`.
#[allow(clippy::too_many_arguments)]
pub fn mark_used(
    _payer: &AccountView,
    _bucket: &mut AccountView,
    _noreplay_program: &AccountView,
    _noreplay_authority: &AccountView,
    _system_program: &AccountView,
    _program_id: &Address,
    _chain: u16,
    _emitter: &[u8; 32],
    _sequence: u64,
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
