//! NoReplay integration — direct-read pre-check (`is_marked`) plus the
//! `MarkUsed` CPI into `solana-noreplay`. `is_marked` is shared by
//! `submit_observations`, `submit_vaas`, `register_chain`, and `close_pending`;
//! `mark_used` fires only from the first three (`close_pending` never holds the
//! authority PDA).
//!
//! `solana-noreplay` wire format:
//!
//! - Account size: 129 bytes (`[bump: u8][bitmap: 128 B]`).
//! - Bit `sequence % 1024` of `bitmap` marks the sequence.
//! - `MarkUsed` data: `[disc=1u8][ns_len: u16 LE][ns][seq: u64 LE]`.
//! - `MarkUsed` accounts: payer (signer, writable), authority (signer, readonly),
//!   bitmap PDA (writable), system program (readonly).
//! - Bitmap PDA seeds: `[authority, ns[..min(len, 32)], ns[min(len, 32)..],
//!   (seq / 1024) LE]`.
//!
//! Our authority is a PDA at `[NOREPLAY_AUTHORITY_SEED_PREFIX]`, so the bitmap is
//! write-controlled exclusively by this program.

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::{GlobalAccountantError, NOREPLAY_BITS_PER_BUCKET, NOREPLAY_PROGRAM_ID};
use crate::err;

/// NoReplay namespace: `chain_be (2 B) ‖ emitter (32 B)`. The noreplay program
/// splits namespaces > 32 bytes at this boundary into two seed chunks.
const NAMESPACE_TOTAL_LEN: usize = 2 + 32;
const NAMESPACE_CHUNK_BOUNDARY: usize = 32;

/// Re-derive the canonical noreplay bitmap PDA for
/// `(authority, chain, emitter, sequence)`. Seeds:
/// `[authority, namespace[..32], namespace[32..], (sequence / 1024) LE]` where
/// `namespace = chain_be ‖ emitter`. Returns `(address, bump)`.
pub fn derive_bucket_pda(
    noreplay_authority: &Address,
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
) -> (Address, u8) {
    let mut namespace = [0u8; NAMESPACE_TOTAL_LEN];
    namespace[..2].copy_from_slice(&chain.to_be_bytes());
    namespace[2..].copy_from_slice(emitter);
    let bucket_index_bytes = (sequence / NOREPLAY_BITS_PER_BUCKET).to_le_bytes();
    let noreplay_program_id_addr = Address::from(NOREPLAY_PROGRAM_ID);
    let authority_bytes: &[u8] = noreplay_authority.as_array();
    Address::find_program_address(
        &[
            authority_bytes,
            &namespace[..NAMESPACE_CHUNK_BOUNDARY],
            &namespace[NAMESPACE_CHUNK_BOUNDARY..],
            &bucket_index_bytes,
        ],
        &noreplay_program_id_addr,
    )
}

// Pre-check (direct account read).

/// Direct-read pre-check against the noreplay bitmap PDA. Returns:
///   - `Ok(false)` if uninitialised or the bit is clear — proceed.
///   - `Ok(true)` if the bit at `sequence % 1024` is set — `AlreadyAccounted`.
///   - `Err(InvalidPda)` on non-canonical address or malformed data.
///
/// The bucket address is re-derived and a non-canonical account is rejected —
/// otherwise a caller could trick the bit lookup into reading another namespace.
pub fn is_marked(
    bucket: &AccountView,
    noreplay_authority: &Address,
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
) -> Result<bool, pinocchio::error::ProgramError> {
    let (expected_bucket, _) = derive_bucket_pda(noreplay_authority, chain, emitter, sequence);
    if bucket.address() != &expected_bucket {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    // Uninitialised bucket (system-owned) — no bit set yet, the normal first-
    // message case.
    if bucket.owner() == &pinocchio_system::ID {
        return Ok(false);
    }
    // Initialised: bitmap PDA must be exactly 129 bytes.
    let data = bucket.try_borrow()?;
    if data.len()
        != crate::definitions::NOREPLAY_BITMAP_OFFSET + crate::definitions::NOREPLAY_BITMAP_BYTES
    {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    let bit = (sequence % NOREPLAY_BITS_PER_BUCKET) as usize;
    let byte = data[crate::definitions::NOREPLAY_BITMAP_OFFSET + bit / 8];
    Ok(byte & (1 << (bit % 8)) != 0)
}

// Mark-used: CPIs `MarkUsed` into solana-noreplay with invoke_signed authority.

/// `MarkUsed` instruction-data length:
/// `[disc=1u8][ns_len: u16 LE][ns: 34 B][seq: u64 LE]`.
const MARK_USED_DATA_LEN: usize = 1 + 2 + NAMESPACE_TOTAL_LEN + 8;

#[allow(clippy::too_many_arguments)]
pub fn mark_used(
    payer: &AccountView,
    bucket: &mut AccountView,
    _noreplay_program: &AccountView,
    noreplay_authority: &AccountView,
    system_program: &AccountView,
    program_id: &pinocchio::Address,
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
) -> ProgramResult {
    use pinocchio::cpi::{Seed, Signer};
    use pinocchio::instruction::{InstructionAccount, InstructionView};

    use crate::definitions::{NOREPLAY_AUTHORITY_SEED_PREFIX, NOREPLAY_MARK_USED_DISCRIMINATOR};

    // SECURITY: the CPI target is built from the hardcoded `NOREPLAY_PROGRAM_ID`
    // constant, never `_noreplay_program.address()` — a caller-controlled target
    // would let an attacker fake `MarkUsed` success and bypass replay protection.

    // Derive and verify the canonical noreplay-authority PDA.
    let noreplay_program_id_addr = Address::from(NOREPLAY_PROGRAM_ID);
    let (expected_authority, authority_bump) =
        Address::find_program_address(&[NOREPLAY_AUTHORITY_SEED_PREFIX], program_id);
    if noreplay_authority.address() != &expected_authority {
        return Err(err(GlobalAccountantError::InvalidPda));
    }

    // Namespace: chain_be (2 B) ‖ emitter (32 B).
    let mut namespace = [0u8; NAMESPACE_TOTAL_LEN];
    namespace[..2].copy_from_slice(&chain.to_be_bytes());
    namespace[2..].copy_from_slice(emitter);

    let mut ix_data = [0u8; MARK_USED_DATA_LEN];
    ix_data[0] = NOREPLAY_MARK_USED_DISCRIMINATOR;
    ix_data[1..3].copy_from_slice(&(NAMESPACE_TOTAL_LEN as u16).to_le_bytes());
    ix_data[3..3 + NAMESPACE_TOTAL_LEN].copy_from_slice(&namespace);
    ix_data[3 + NAMESPACE_TOTAL_LEN..].copy_from_slice(&sequence.to_le_bytes());

    // `MarkUsed` account list:
    //   0. [signer, writable] payer
    //   1. [signer, readonly] authority (our PDA)
    //   2. [writable]         bitmap PDA
    //   3. [readonly]         system program
    let ix_accounts = [
        InstructionAccount::writable_signer(payer.address()),
        InstructionAccount::readonly_signer(noreplay_authority.address()),
        InstructionAccount::writable(bucket.address()),
        InstructionAccount::readonly(system_program.address()),
    ];

    let instruction = InstructionView {
        program_id: &noreplay_program_id_addr,
        data: &ix_data,
        accounts: &ix_accounts,
    };

    // invoke_signed seeds for the noreplay-authority PDA.
    let bump_seed = [authority_bump];
    let signer_seeds = [
        Seed::from(NOREPLAY_AUTHORITY_SEED_PREFIX),
        Seed::from(bump_seed.as_slice()),
    ];
    let signers = [Signer::from(&signer_seeds)];

    // `invoke_signed` itself only returns `Err(...)` for *pre-CPI validation*
    // failures — `NotEnoughAccountKeys`, `InvalidArgument` (address mismatch),
    // borrow-check conflicts. If the inner noreplay program returns a
    // `ProgramError` (e.g. `AccountAlreadyInitialized` on a race-loss), the
    // SBF runtime aborts THIS program with the inner exit code directly;
    // pinocchio's `invoke_signed` never sees that error. So mapping the
    // returned `Result` to a custom "CPI failed" code would only fire on
    // caller bugs in this helper, which are better surfaced as their natural
    // variant for debuggability. Pass through unchanged.
    pinocchio::cpi::invoke_signed(
        &instruction,
        &[payer, noreplay_authority, bucket, system_program],
        &signers,
    )
}
