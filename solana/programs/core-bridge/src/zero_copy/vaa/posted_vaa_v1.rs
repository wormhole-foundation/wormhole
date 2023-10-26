use std::cell::Ref;

use crate::{state, types::Timestamp};
use anchor_lang::{
    prelude::{
        error, require, require_eq, require_keys_eq, AccountInfo, ErrorCode, Pubkey, Result,
    },
    solana_program::keccak,
    Key,
};
use state::POSTED_VAA_V1_SEED_PREFIX;

/// Account used to store a verified VAA.
pub struct PostedVaaV1<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> PostedVaaV1<'a> {
    pub const DISC: [u8; 4] = state::POSTED_VAA_V1_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 95;
    pub const SEED_PREFIX: &'static [u8] = state::POSTED_VAA_V1_SEED_PREFIX;

    pub fn discriminator(&self) -> [u8; 4] {
        self.0[..4].try_into().unwrap()
    }

    /// Level of consistency requested by the emitter.
    pub fn consistency_level(&self) -> u8 {
        self.0[4]
    }

    /// Time the message was submitted.
    pub fn timestamp(&self) -> Timestamp {
        u32::from_le_bytes(self.0[5..9].try_into().unwrap()).into()
    }

    /// Pubkey of `SignatureSet` account that represent this VAA's signature verification.
    pub fn signature_set(&self) -> Pubkey {
        Pubkey::try_from(&self.0[9..41]).unwrap()
    }

    /// Guardian set index used to verify signatures for `SignatureSet`.
    ///
    /// NOTE: In the previous implementation, this member was referred to as the `posted_timestamp`,
    /// which is zero for VAA data (posted messages and VAAs resemble the same account schema). By
    /// changing this to the guardian set index, we patch a bug with verifying governance VAAs for
    /// the Core Bridge (other Core Bridge implementations require that the guardian set that
    /// attested for the governance VAA is the current one).
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_le_bytes(self.0[41..45].try_into().unwrap())
    }

    /// Unique ID for this message.
    pub fn nonce(&self) -> u32 {
        u32::from_le_bytes(self.0[45..49].try_into().unwrap())
    }

    /// Sequence number of this message.
    pub fn sequence(&self) -> u64 {
        u64::from_le_bytes(self.0[49..57].try_into().unwrap())
    }

    /// The Wormhole chain ID denoting the origin of this message.
    pub fn emitter_chain(&self) -> u16 {
        u16::from_le_bytes(self.0[57..59].try_into().unwrap())
    }

    /// Emitter of the message.
    pub fn emitter_address(&self) -> [u8; 32] {
        self.0[59..91].try_into().unwrap()
    }

    pub fn payload_size(&self) -> usize {
        u32::from_le_bytes(self.0[91..Self::PAYLOAD_START].try_into().unwrap())
            .try_into()
            .unwrap()
    }

    /// Message payload.
    pub fn payload(&self) -> &[u8] {
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

    pub(super) fn new(acc_info: &'a AccountInfo) -> Result<Self> {
        let parsed = Self(acc_info.try_borrow_data()?);
        require!(
            parsed.0.len() >= Self::PAYLOAD_START,
            ErrorCode::AccountDidNotDeserialize
        );
        require_eq!(
            parsed.0.len(),
            Self::PAYLOAD_START + parsed.payload_size(),
            ErrorCode::AccountDidNotDeserialize
        );

        // Recompute message hash to re-derive PDA address.
        let (expected_address, _) = Pubkey::find_program_address(
            &[POSTED_VAA_V1_SEED_PREFIX, parsed.message_hash().as_ref()],
            &crate::ID,
        );
        require_keys_eq!(acc_info.key(), expected_address, ErrorCode::ConstraintSeeds);

        Ok(parsed)
    }
}

impl<'a> crate::zero_copy::LoadZeroCopy<'a> for PostedVaaV1<'a> {
    fn load(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let parsed = Self::new(acc_info)?;
        require!(
            parsed.discriminator() == Self::DISC,
            ErrorCode::AccountDidNotDeserialize
        );

        Ok(parsed)
    }
}
