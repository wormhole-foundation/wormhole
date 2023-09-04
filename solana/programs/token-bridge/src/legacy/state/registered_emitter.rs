use anchor_lang::prelude::*;

#[derive(Debug, AnchorDeserialize, AnchorSerialize, Clone, Copy, PartialEq, Eq, InitSpace)]
pub struct RegisteredEmitter {
    pub chain: u16,
    pub contract: [u8; 32],
}

impl Owner for RegisteredEmitter {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl core_bridge_program::legacy::utils::LegacyDiscriminator<0> for RegisteredEmitter {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}
