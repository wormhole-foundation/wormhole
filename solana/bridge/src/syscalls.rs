use solana_sdk::program_error::ProgramError;

use crate::error::Error;

#[repr(C)]
pub struct EcrecoverInput {
    pub r: [u8; 32],
    pub s: [u8; 32],
    pub v: u8,
    pub message: [u8; 32],
}

#[repr(C)]
pub struct EcrecoverOutput {
    pub address: [u8; 20],
}

impl EcrecoverInput {
    pub fn new(r: [u8; 32], s: [u8; 32], v: u8, message: [u8; 32]) -> EcrecoverInput {
        EcrecoverInput { r, s, v, message }
    }
}

/// Verify an ETH optimized Schnorr signature
///
/// @param input - Input for signature verification
//#[cfg(target_arch = "bpf")]
#[inline]
pub fn sol_syscall_ecrecover(input: &EcrecoverInput) -> Result<EcrecoverOutput, Error> {
    let mut output = EcrecoverOutput { address: [0; 20] };
    let res = unsafe {
        sol_ecrecover(
            input as *const _ as *const u8,
            (&mut output) as *mut _ as *mut u8,
        )
    };
    if res == 1 {
        Ok(output)
    } else {
        Err(Error::InvalidVAASignature)
    }
}

//#[cfg(target_arch = "bpf")]
extern "C" {
    fn sol_ecrecover(input: *const u8, output: *mut u8) -> u64;
}
