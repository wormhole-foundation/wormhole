use anchor_lang::prelude::*;
use wormhole_solana_common::legacy_account;
use wormhole_solana_common::{LegacyDiscriminator, SeedPrefix};

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct EmitterSequence {
    pub value: u64,
}

impl LegacyDiscriminator<0> for EmitterSequence {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for EmitterSequence {
    #[inline]
    fn seed_prefix() -> &'static [u8] {
        b"Sequence"
    }
}
