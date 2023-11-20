use crate::legacy::utils::LegacyAccount;
use anchor_lang::prelude::*;

/// Account used to store the current sequence number for a given emitter.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct LegacyEmitterSequence {
    /// Current sequence number, which will be used the next time this emitter publishes a message.
    pub value: u64,
}

impl LegacyAccount for LegacyEmitterSequence {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl LegacyEmitterSequence {
    pub const SEED_PREFIX: &'static [u8] = b"Sequence";
}

impl std::ops::Deref for LegacyEmitterSequence {
    type Target = u64;

    fn deref(&self) -> &Self::Target {
        &self.value
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub enum EmitterType {
    Unset,
    Legacy,
    Executable,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct EmitterSequence {
    pub legacy: LegacyEmitterSequence,
    pub bump: u8,
    pub emitter_type: EmitterType,
}

impl std::ops::Deref for EmitterSequence {
    type Target = LegacyEmitterSequence;

    fn deref(&self) -> &Self::Target {
        &self.legacy
    }
}

impl std::ops::DerefMut for EmitterSequence {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.legacy
    }
}

impl EmitterSequence {
    pub const SEED_PREFIX: &'static [u8] = LegacyEmitterSequence::SEED_PREFIX;
}

impl LegacyAccount for EmitterSequence {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
}
