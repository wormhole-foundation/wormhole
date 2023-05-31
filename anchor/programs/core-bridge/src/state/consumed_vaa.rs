use anchor_lang::prelude::*;
use wormhole_solana_common::SeedPrefix;

#[account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct ConsumedVaa {}

impl SeedPrefix for ConsumedVaa {
    fn seed_prefix() -> &'static [u8] {
        b"Consumed"
    }
}
