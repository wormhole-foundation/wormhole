use crate::types::MessageHash;
use anchor_lang::prelude::*;

/// Account used to store information about a guardian set used to sign a VAA. There is only one
/// signature set for each verified VAA (associated with a
/// [PostedVaaV1](crate::legacy::state::PostedVaaV1) account).
///
/// This account is created using the verify signatures legacy instruction.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct SignatureSet {
    /// Signatures of validators
    pub sig_verify_successes: Vec<bool>,

    /// Hash of the VAA message body.
    pub message_hash: MessageHash,

    /// Index of the guardian set
    pub guardian_set_index: u32,
}

impl crate::legacy::utils::LegacyAccount<0> for SignatureSet {
    const DISCRIMINATOR: [u8; 0] = [];

    fn program_id() -> Pubkey {
        crate::ID
    }
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
