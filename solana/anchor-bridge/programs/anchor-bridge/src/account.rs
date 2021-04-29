use anchor_lang::{prelude::*, solana_program};

use crate::{types::Version, MAX_LEN_GUARDIAN_KEYS};

#[account]
pub struct BridgeInfo {}

#[account]
pub struct GuardianSetInfo {
    /// Version number of this guardian set.
    pub version: Version,
    /// Number of keys stored
    pub len_keys: u8,
    /// public key hashes of the guardian set
    pub keys: [[u8; 20]; MAX_LEN_GUARDIAN_KEYS],
    /// creation time
    pub creation_time: u32,
    /// expiration time when VAAs issued by this set are no longer valid
    pub expiration_time: u32,
}
