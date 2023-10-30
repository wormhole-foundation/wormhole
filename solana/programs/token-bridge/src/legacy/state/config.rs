use anchor_lang::prelude::*;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Config {
    pub core_bridge_program: Pubkey,
}

impl core_bridge_program::legacy::utils::LegacyAccount for Config {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl Config {
    pub const SEED_PREFIX: &'static [u8] = b"config";
}
