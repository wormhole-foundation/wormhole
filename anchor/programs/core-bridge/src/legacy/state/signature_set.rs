use crate::types::MessageHash;
use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, NewAccountSize};

#[legacy_account]
#[derive(Debug, PartialEq, Eq)]
pub struct SignatureSet {
    /// Signatures of validators
    pub sig_verify_successes: Vec<bool>,

    /// Hash of the VAA message body.
    pub message_hash: MessageHash,

    /// Index of the guardian set
    pub guardian_set_index: u32,
}

impl SignatureSet {
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

impl LegacyDiscriminator<0> for SignatureSet {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl NewAccountSize for SignatureSet {
    fn compute_size(num_signatures: usize) -> usize {
        4 // Vec::len
        + num_signatures // signatures
        + MessageHash::INIT_SPACE // hash
        + 4 // guardian_set_index
    }
}
