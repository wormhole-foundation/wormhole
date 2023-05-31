use crate::types::Timestamp;
use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, NewAccountSize, SeedPrefix};

#[legacy_account]
#[derive(Debug, PartialEq, Eq)]
pub struct GuardianSet {
    /// Index representing an incrementing version number for this guardian set.
    pub index: u32,

    /// Ethereum-style public keys.
    pub keys: Vec<[u8; 20]>,

    /// Timestamp representing the time this guardian became active.
    pub creation_time: Timestamp,

    /// Expiration time when VAAs issued by this set are no longer valid.
    pub expiration_time: Timestamp,
}

impl GuardianSet {
    pub fn is_active(&self, timestamp: &Timestamp) -> bool {
        // Note: This is a fix for Wormhole on mainnet.  The initial guardian set was never expired
        // so we block it here.
        if self.index == 0 && self.creation_time == 1628099186.into() {
            false
        } else {
            self.expiration_time == Default::default() || self.expiration_time >= *timestamp
        }
    }
}

impl LegacyDiscriminator<0> for GuardianSet {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for GuardianSet {
    #[inline]
    fn seed_prefix() -> &'static [u8] {
        b"GuardianSet"
    }
}

impl NewAccountSize for GuardianSet {
    fn compute_size(num_guardians: usize) -> usize {
        4 // index
        + 4 + num_guardians * 20 // keys
        + Timestamp::INIT_SPACE // creation_time
        + Timestamp::INIT_SPACE // expiration_time
    }
}
