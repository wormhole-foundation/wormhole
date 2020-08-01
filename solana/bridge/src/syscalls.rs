#[repr(C)]
pub struct SchnorrifyInput {
    message: [u8; 32],
    addr: [u8; 20],
    signature: [u8; 32],
    pub_key: [u8; 64],
}

impl SchnorrifyInput {
    pub fn new(pub_key: [u8; 64], message: [u8; 32], signature: [u8; 32], addr: [u8; 20]) -> SchnorrifyInput {
        SchnorrifyInput {
            message,
            addr,
            signature,
            pub_key,
        }
    }
}

/// Verify an ETH optimized Schnorr signature
///
/// @param input - Input for signature verification
#[inline]
pub fn sol_verify_schnorr(input: &SchnorrifyInput) -> bool {
    let res = unsafe {
        sol_verify_ethschnorr(input as *const _ as *const u8)
    };

    res == 1
}

extern "C" {
    fn sol_verify_ethschnorr(input: *const u8) -> u64;
}
