//! `keccak256` helpers shared across handlers.
//!
//! The SBF arm calls the syscall; the host arm panics. Host compilation is
//! still required (mollusk/surfpool test binaries and clippy are host-only),
//! and pinocchio's `syscalls` re-export does not exist on the host target, so
//! the cfg split is mandatory.

/// `keccak256(data)` into `result`. Panics on the host target.
pub(crate) fn keccak256(data: &[u8], result: &mut [u8; 32]) {
    #[cfg(any(target_os = "solana", target_arch = "bpf"))]
    {
        let vals: [&[u8]; 1] = [data];
        // SAFETY: the runtime reads exactly `val_len` `&[u8]` fat pointers
        // starting at `vals_ptr`.
        unsafe {
            pinocchio::syscalls::sol_keccak256(
                vals.as_ptr() as *const u8,
                vals.len() as u64,
                result.as_mut_ptr(),
            );
        }
    }
    #[cfg(not(any(target_os = "solana", target_arch = "bpf")))]
    {
        let _ = (data, result);
        unreachable!("keccak256 is only available on the SBF target");
    }
}

/// `keccak256(keccak256(body))` — the Wormhole VAA digest convention used by
/// guardian signing and the Verify VAA Shim's `VerifyHash`.
pub(crate) fn double_keccak256(body: &[u8]) -> [u8; 32] {
    let mut inner = [0u8; 32];
    keccak256(body, &mut inner);
    let mut outer = [0u8; 32];
    keccak256(&inner, &mut outer);
    outer
}
