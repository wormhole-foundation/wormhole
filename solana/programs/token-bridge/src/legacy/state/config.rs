use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, SeedPrefix};

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct Config {
    pub core_bridge_program: Pubkey,
}

impl LegacyDiscriminator<0> for Config {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for Config {
    const SEED_PREFIX: &'static [u8] = b"config";
}