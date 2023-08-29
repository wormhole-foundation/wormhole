use anchor_lang::prelude::*;

#[account]
#[derive(InitSpace)]
pub struct SignerSequence {
    pub value: u128,
}

impl SignerSequence {
    pub const SEED_PREFIX: &'static [u8] = b"seq";
}
