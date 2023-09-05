use std::ops::Deref;

use crate::types::Timestamp;
use anchor_lang::{prelude::*, solana_program::keccak};

/// A.K.A. "PostedVAA".
pub const POSTED_VAA_V1_SEED_PREFIX: &[u8] = b"PostedVAA";

/// A.K.A. "vaa\1".
pub const POSTED_VAA_V1_DISCRIMINATOR: [u8; 4] = *b"vaa\x01";

/// VAA metadata defining information about a Wormhole message attested for by an active guardian
/// set.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct PostedVaaV1Info {
    /// Level of consistency requested by the emitter.
    pub consistency_level: u8,

    /// Time the message was submitted.
    pub timestamp: Timestamp,

    /// Pubkey of `SignatureSet` account that represent this VAA's signature verification.
    pub signature_set: Pubkey,

    /// Guardian set index used to verify signatures for `SignatureSet`.
    ///
    /// NOTE: In the previous implementation, this member was referred to as the `posted_timestamp`,
    /// which is zero for VAA data (posted messages and VAAs resemble the same account schema). By
    /// changing this to the guardian set index, we patch a bug with verifying governance VAAs for
    /// the Core Bridge (other Core Bridge implementations require that the guardian set that
    /// attested for the governance VAA is the current one).
    pub guardian_set_index: u32,

    /// Unique id for this message.
    pub nonce: u32,

    /// Sequence number of this message.
    pub sequence: u64,

    /// The Wormhole chain ID denoting the origin of this message.
    pub emitter_chain: u16,

    /// Emitter of the message.
    pub emitter_address: [u8; 32],
}

/// Account used to store a verified VAA.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct PostedVaaV1 {
    pub info: PostedVaaV1Info,
    pub payload: Vec<u8>,
}

impl Owner for PostedVaaV1 {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl crate::legacy::utils::LegacyDiscriminator<4> for PostedVaaV1 {
    const LEGACY_DISCRIMINATOR: [u8; 4] = POSTED_VAA_V1_DISCRIMINATOR;
}

impl PostedVaaV1 {
    /// A.K.A. "PostedVAA".
    pub const SEED_PREFIX: &'static [u8] = POSTED_VAA_V1_SEED_PREFIX;

    /// Recompute the message hash, which is used derive the PostedVaaV1 PDA address.
    pub fn message_hash(&self) -> keccak::Hash {
        keccak::hashv(&[
            &self.timestamp.to_be_bytes(),
            &self.nonce.to_be_bytes(),
            &self.emitter_chain.to_be_bytes(),
            &self.emitter_address,
            &self.sequence.to_be_bytes(),
            &[self.consistency_level],
            &self.payload,
        ])
    }

    pub(crate) fn compute_size(payload_len: usize) -> usize {
        4 // LEGACY_DISCRIMINATOR
        + PostedVaaV1Info::INIT_SPACE
        + 4 // payload.len()
        + payload_len
    }
}

impl Deref for PostedVaaV1 {
    type Target = PostedVaaV1Info;

    fn deref(&self) -> &Self::Target {
        &self.info
    }
}
