use anchor_lang::prelude::*;

#[derive(Debug, AnchorDeserialize, AnchorSerialize, Clone, Copy, PartialEq, Eq, InitSpace)]
pub struct RegisteredEmitter {
    pub chain: u16,
    pub contract: [u8; 32],
}

impl core_bridge_program::legacy::utils::LegacyAccount<0> for RegisteredEmitter {
    const DISCRIMINATOR: [u8; 0] = [];

    fn program_id() -> Pubkey {
        crate::ID
    }
}
