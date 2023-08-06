use crate::{state, types::Timestamp};
use anchor_lang::{
    prelude::{require, ErrorCode, Pubkey},
    solana_program::keccak,
};

/// Account used to store a verified VAA.
pub struct PostedVaaV1<'a>(&'a [u8]);

impl<'a> PostedVaaV1<'a> {
    pub const DISCRIMINATOR: [u8; 4] = state::POSTED_VAA_V1_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 91;
    pub const SEED_PREFIX: &'static [u8] = state::POSTED_VAA_V1_SEED_PREFIX;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    /// Level of consistency requested by the emitter.
    pub fn consistency_level(&self) -> u8 {
        self.0[0]
    }

    /// Time the message was submitted.
    pub fn timestamp(&self) -> Timestamp {
        u32::from_le_bytes(self.0[1..5].try_into().unwrap()).into()
    }

    /// Pubkey of `SignatureSet` account that represent this VAA's signature verification.
    pub fn signature_set(&self) -> Pubkey {
        Pubkey::try_from(&self.0[5..37]).unwrap()
    }

    /// Guardian set index used to verify signatures for `SignatureSet`.
    ///
    /// NOTE: In the previous implementation, this member was referred to as the `posted_timestamp`,
    /// which is zero for VAA data (posted messages and VAAs resemble the same account schema). By
    /// changing this to the guardian set index, we patch a bug with verifying governance VAAs for
    /// the Core Bridge (other Core Bridge implementations require that the guardian set that
    /// attested for the governance VAA is the current one).
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_le_bytes(self.0[37..41].try_into().unwrap())
    }

    /// Unique ID for this message.
    pub fn nonce(&self) -> u32 {
        u32::from_le_bytes(self.0[41..45].try_into().unwrap())
    }

    /// Sequence number of this message.
    pub fn sequence(&self) -> u64 {
        u64::from_le_bytes(self.0[45..53].try_into().unwrap())
    }

    /// The Wormhole chain ID denoting the origin of this message.
    pub fn emitter_chain(&self) -> u16 {
        u16::from_le_bytes(self.0[53..55].try_into().unwrap())
    }

    /// Emitter of the message.
    pub fn emitter_address(&self) -> [u8; 32] {
        self.0[55..87].try_into().unwrap()
    }

    pub fn payload_size(&self) -> usize {
        u32::from_le_bytes(self.0[87..Self::PAYLOAD_START].try_into().unwrap())
            .try_into()
            .unwrap()
    }

    /// Message payload.
    pub fn payload(&self) -> &'a [u8] {
        &self.0[Self::PAYLOAD_START..]
    }

    /// Recompute the message hash, which is used derive the [PostedVaaV1] PDA address.
    pub fn message_hash(&self) -> keccak::Hash {
        keccak::hashv(&[
            self.timestamp().to_be_bytes().as_ref(),
            self.nonce().to_be_bytes().as_ref(),
            self.emitter_chain().to_be_bytes().as_ref(),
            &self.emitter_address(),
            &self.sequence().to_be_bytes(),
            &[self.consistency_level()],
            self.payload(),
        ])
    }

    /// Compute digest (hash of [message_hash](Self::message_hash)).
    pub fn digest(&self) -> keccak::Hash {
        keccak::hash(self.message_hash().as_ref())
    }

    /// Parse account data assumed to match the [PostedVaaV1](state::PostedVaaV1) schema.
    ///
    /// NOTE: There is no ownership check because [AccountInfo](anchor_lang::prelude::AccountInfo)
    /// is not passed into this method.
    pub fn parse(span: &'a [u8]) -> anchor_lang::Result<Self> {
        require!(
            span.len() > Self::DISC_LEN,
            ErrorCode::AccountDidNotDeserialize
        );
        require!(
            span[..Self::DISC_LEN] == Self::DISCRIMINATOR,
            ErrorCode::AccountDidNotDeserialize
        );

        Ok(Self(&span[Self::DISC_LEN..]))
    }
}
