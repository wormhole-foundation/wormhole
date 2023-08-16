use crate::types::Duration;
use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, SeedPrefix};

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct Config {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: u32,

    /// Lamports in the collection account
    pub last_lamports: u64,

    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_ttl: Duration,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee_lamports: u64,
}

impl LegacyDiscriminator<0> for Config {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for Config {
    const SEED_PREFIX: &'static [u8] = b"Bridge";
}
