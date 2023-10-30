use anchor_lang::prelude::*;

/// Account used to store the current sequence number for a given emitter.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct EmitterSequence {
    /// Current sequence number, which will be used the next time this emitter publishes a message.
    pub value: u64,
}

impl crate::legacy::utils::LegacyAccount for EmitterSequence {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
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
