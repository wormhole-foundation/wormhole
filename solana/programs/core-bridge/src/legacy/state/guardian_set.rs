use crate::types::Timestamp;
use anchor_lang::prelude::*;

/// Account used to store a guardian set. The keys encoded in this account are Ethereum pubkeys.
/// Its expiration time is determined at the time a guardian set is updated to a new set, where the
/// current network clock time is used with
/// [guardian_set_ttl](crate::state::Config::guardian_set_ttl).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
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

impl crate::legacy::utils::LegacyAccount<0> for GuardianSet {
    const DISCRIMINATOR: [u8; 0] = [];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl GuardianSet {
    pub const SEED_PREFIX: &'static [u8] = b"GuardianSet";

    pub(crate) fn compute_size(num_guardians: usize) -> usize {
        4 // index
        + 4 + num_guardians * 20 // keys
        + Timestamp::INIT_SPACE // creation_time
        + Timestamp::INIT_SPACE // expiration_time
    }

    pub fn is_active(&self, timestamp: &Timestamp) -> bool {
        // Note: This is a fix for Wormhole on mainnet.  The initial guardian set was never expired
        // so we block it here.
        if self.index == 0 && self.creation_time == 1628099186 {
            false
        } else {
            self.expiration_time == 0 || self.expiration_time >= *timestamp
        }
    }
}
