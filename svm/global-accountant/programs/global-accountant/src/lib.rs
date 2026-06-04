//! Wormhole Global Accountant — Solana port (Pinocchio).

// `no_std` on the SBF target (and `bpf`, so nightly clippy can lint the
// SBF-shaped code — platform-tools ships no clippy).
#![cfg_attr(any(target_os = "solana", target_arch = "bpf"), no_std)]
// `target_os = "solana"` is unknown to the host toolchain.
#![allow(unexpected_cfgs)]

pub mod entrypoint;
pub(crate) mod hash;
pub mod instructions;
pub mod state;

pub use global_accountant_definitions as definitions;

use pinocchio::error::ProgramError;

use crate::definitions::GlobalAccountantError;

/// Convert a `GlobalAccountantError` into a `ProgramError::Custom`. A free
/// function rather than a `From` impl because orphan rules forbid the impl
/// (both types are foreign) and `definitions` must stay pinocchio-free.
#[inline]
pub(crate) fn err(e: GlobalAccountantError) -> ProgramError {
    ProgramError::Custom(e as u32)
}
