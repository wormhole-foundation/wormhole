use std::ops::{Deref, DerefMut};

use crate::types::Duration;
use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, SeedPrefix};

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct BridgeProgramData {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: u32,

    /// Lamports in the collection account
    pub last_lamports: u64,

    /// Bridge configuration, which is set once upon initialization.
    pub config: BridgeConfig,
}

impl Deref for BridgeProgramData {
    type Target = BridgeConfig;

    fn deref(&self) -> &Self::Target {
        &self.config
    }
}

impl DerefMut for BridgeProgramData {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.config
    }
}

impl LegacyDiscriminator<0> for BridgeProgramData {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for BridgeProgramData {
    #[inline]
    fn seed_prefix() -> &'static [u8] {
        b"Bridge"
    }
}

#[derive(Debug, Clone, PartialEq, Eq, AnchorSerialize, AnchorDeserialize, InitSpace)]
pub struct BridgeConfig {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_ttl: Duration,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee_lamports: u64,
}
