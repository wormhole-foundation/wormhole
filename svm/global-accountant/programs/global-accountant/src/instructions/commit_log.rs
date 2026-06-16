//! Canonical commit-log emit, invoked by `submit_observations` on quorum.
//!
//! Off-chain indexers consume the program-log line carrying the
//! [`ACCOUNTANT_DIGEST_LOG_TAG`] prefix. The payload layout is the single
//! source of truth in [`crate::definitions`]; this module merely marshals
//! caller-supplied fields into the 86-byte buffer and hands it to
//! `sol_log_data`.

use crate::definitions::{ACCOUNTANT_DIGEST_LOG_LEN, ACCOUNTANT_DIGEST_LOG_TAG};

/// Emit one canonical commit log entry. The host build is a no-op so mollusk
/// host-side compilation continues to work; the SBF build calls the
/// `sol_log_data` syscall directly.
pub(crate) fn emit(
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
    digest: &[u8; 32],
    guardian_set_index: u32,
) {
    let mut buf = [0u8; ACCOUNTANT_DIGEST_LOG_LEN];
    buf[..8].copy_from_slice(&ACCOUNTANT_DIGEST_LOG_TAG);
    buf[8..10].copy_from_slice(&chain.to_be_bytes());
    buf[10..42].copy_from_slice(emitter);
    buf[42..50].copy_from_slice(&sequence.to_be_bytes());
    buf[50..82].copy_from_slice(digest);
    buf[82..86].copy_from_slice(&guardian_set_index.to_le_bytes());

    log_data(&buf);
}

#[cfg(any(target_os = "solana", target_arch = "bpf"))]
fn log_data(buf: &[u8]) {
    let slices: [&[u8]; 1] = [buf];
    // SAFETY: `sol_log_data` reads exactly `slices.len()` fat-pointer slice
    // references starting at `slices.as_ptr()`. The buffer outlives the call.
    unsafe {
        pinocchio::syscalls::sol_log_data(slices.as_ptr() as *const u8, slices.len() as u64);
    }
}

#[cfg(not(any(target_os = "solana", target_arch = "bpf")))]
fn log_data(_buf: &[u8]) {
    // Host build: no-op. Mollusk drives the SBF .so where the syscall is real;
    // host-only `cargo check` / clippy must still link.
}
