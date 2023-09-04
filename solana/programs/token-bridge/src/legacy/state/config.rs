use anchor_lang::prelude::*;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Config {
    pub core_bridge_program: Pubkey,
}

impl Owner for Config {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl core_bridge_program::legacy::utils::LegacyDiscriminator<0> for Config {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl Config {
    pub const SEED_PREFIX: &'static [u8] = b"config";
}
