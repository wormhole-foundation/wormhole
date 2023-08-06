use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator};

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct RegisteredEmitter {
    pub chain: u16,
    pub contract: [u8; 32],
}

impl LegacyDiscriminator<0> for RegisteredEmitter {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}
