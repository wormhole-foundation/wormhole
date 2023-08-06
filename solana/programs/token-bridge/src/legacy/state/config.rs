use anchor_lang::prelude::*;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Config {
    pub core_bridge_program: Pubkey,
}

impl core_bridge_program::legacy::utils::LegacyAccount<0> for Config {
    const DISCRIMINATOR: [u8; 0] = [];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl Config {
    pub const SEED_PREFIX: &'static [u8] = b"config";
}
