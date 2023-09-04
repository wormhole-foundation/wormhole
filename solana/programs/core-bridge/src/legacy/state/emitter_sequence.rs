use anchor_lang::prelude::*;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct EmitterSequence {
    pub value: u64,
}

impl Owner for EmitterSequence {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl crate::legacy::utils::LegacyDiscriminator<0> for EmitterSequence {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl EmitterSequence {
    pub const SEED_PREFIX: &'static [u8] = b"Sequence";
}

impl std::ops::Deref for EmitterSequence {
    type Target = u64;

    fn deref(&self) -> &Self::Target {
        &self.value
    }
}
