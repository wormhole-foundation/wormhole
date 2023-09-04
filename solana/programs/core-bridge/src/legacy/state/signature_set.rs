use crate::types::MessageHash;
use anchor_lang::prelude::*;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct SignatureSet {
    /// Signatures of validators
    pub sig_verify_successes: Vec<bool>,

    /// Hash of the VAA message body.
    pub message_hash: MessageHash,

    /// Index of the guardian set
    pub guardian_set_index: u32,
}

impl Owner for SignatureSet {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl crate::legacy::utils::LegacyDiscriminator<0> for SignatureSet {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SignatureSet {
    pub(crate) fn compute_size(num_signatures: usize) -> usize {
        4 // Vec::len
        + num_signatures // signatures
        + MessageHash::INIT_SPACE // hash
        + 4 // guardian_set_index
    }

    pub fn is_initialized(&self) -> bool {
        self.sig_verify_successes.iter().any(|&value| value)
    }

    pub fn num_verified(&self) -> usize {
        self.sig_verify_successes
            .iter()
            .filter(|&&signed| signed)
            .count()
    }
}
