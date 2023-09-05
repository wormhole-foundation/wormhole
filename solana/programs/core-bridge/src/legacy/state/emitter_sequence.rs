use anchor_lang::prelude::*;

/// Account used to store the current sequence number for a given emitter.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct EmitterSequence {
    /// Current sequence number, which will be used the next time this emitter publishes a message.
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
    /// A.K.A. "Sequence".
    pub const SEED_PREFIX: &'static [u8] = b"Sequence";
}

impl std::ops::Deref for EmitterSequence {
    type Target = u64;

    fn deref(&self) -> &Self::Target {
        &self.value
    }
}
