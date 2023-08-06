use anchor_lang::prelude::*;

#[account]
#[derive(InitSpace)]
pub struct SignerSequence {
    pub value: u128,
}

impl SignerSequence {
    pub const SEED_PREFIX: &'static [u8] = b"seq";

    pub fn take_and_uptick(&mut self) -> [u8; 16] {
        let seq = self.value;

        self.value += 1;

        seq.to_le_bytes()
    }
}

impl std::ops::Deref for SignerSequence {
    type Target = u128;

    fn deref(&self) -> &Self::Target {
        &self.value
    }
}
