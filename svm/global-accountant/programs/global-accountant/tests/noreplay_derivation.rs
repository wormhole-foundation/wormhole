//! Unit tests for `instructions::noreplay::derive_bucket_pda`, pinning its seed
//! encoding against a host-side reimplementation of
//! `solana_noreplay::pda::BitmapPdaSeeds`.

use {
    global_accountant::instructions::noreplay::derive_bucket_pda,
    global_accountant_definitions::{NOREPLAY_BITS_PER_BUCKET, NOREPLAY_PROGRAM_ID},
    pinocchio::Address,
    solana_pubkey::Pubkey,
};

/// Reference derivation matching `solana_noreplay::pda::BitmapPdaSeeds::new`,
/// built via `solana_pubkey` (vs. the program's pinocchio impl) to catch
/// transcription errors on either side.
fn reference_bucket_pda(
    authority: &[u8; 32],
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
) -> ([u8; 32], u8) {
    let mut namespace = [0u8; 2 + 32];
    namespace[..2].copy_from_slice(&chain.to_be_bytes());
    namespace[2..].copy_from_slice(emitter);
    let mid = namespace.len().min(32);
    let bucket_index = (sequence / NOREPLAY_BITS_PER_BUCKET).to_le_bytes();
    let (pubkey, bump) = Pubkey::find_program_address(
        &[
            authority,
            &namespace[..mid],
            &namespace[mid..],
            &bucket_index,
        ],
        &Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
    );
    (pubkey.to_bytes(), bump)
}

/// `derive_bucket_pda` agrees with the reference derivation (address + bump).
#[test]
fn derive_bucket_pda_matches_reference_for_canonical_inputs() {
    let authority_bytes = [0x7Au8; 32];
    let authority = Address::from(authority_bytes);
    let chain: u16 = 1;
    let mut emitter = [0u8; 32];
    emitter[31] = 0x11;
    let sequence: u64 = 0x1234_5678_DEAD_BEEF;

    let (ours, ours_bump) = derive_bucket_pda(&authority, chain, &emitter, sequence);
    let (reference, ref_bump) = reference_bucket_pda(&authority_bytes, chain, &emitter, sequence);

    assert_eq!(
        ours.as_array(),
        &reference,
        "derive_bucket_pda must agree with the upstream BitmapPdaSeeds scheme",
    );
    assert_eq!(ours_bump, ref_bump, "canonical bumps must agree");
}
