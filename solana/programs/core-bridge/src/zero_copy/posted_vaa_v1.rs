use crate::{state, types::Timestamp};
use anchor_lang::prelude::{require, AnchorDeserialize, ErrorCode, Pubkey, Result};
use wormhole_solana_common::SeedPrefix;

pub struct PostedVaaV1<'a>(&'a [u8]);

impl<'a> PostedVaaV1<'a> {
    pub const DISCRIMINATOR: [u8; 4] = state::POSTED_VAA_V1_DISCRIMINATOR;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    pub fn consistency_level(&self) -> u8 {
        let mut buf = &self.0[..1];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn timestamp(&self) -> Timestamp {
        let mut buf = &self.0[1..5];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn signature_set(&self) -> Pubkey {
        let mut buf = &self.0[5..37];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn guardian_set_index(&self) -> u32 {
        let mut buf = &self.0[37..41];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn nonce(&self) -> u32 {
        let mut buf = &self.0[41..45];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn sequence(&self) -> u64 {
        let mut buf = &self.0[45..53];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn emitter_chain(&self) -> u16 {
        let mut buf = &self.0[53..55];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn emitter_address(&self) -> [u8; 32] {
        let mut buf = &self.0[55..87];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn payload_size(&self) -> usize {
        let mut buf = &self.0[87..91];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    pub fn payload(&self) -> &'a [u8] {
        &self.0[91..]
    }

    pub fn message_hash(&self) -> solana_program::keccak::Hash {
        solana_program::keccak::hashv(&[
            self.timestamp().to_be_bytes().as_ref(),
            self.nonce().to_be_bytes().as_ref(),
            self.emitter_chain().to_be_bytes().as_ref(),
            &self.emitter_address(),
            &self.sequence().to_be_bytes(),
            &[self.consistency_level()],
            self.payload(),
        ])
    }

    pub fn digest(&self) -> solana_program::keccak::Hash {
        solana_program::keccak::hash(self.message_hash().as_ref())
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        require!(
            span.len() > Self::DISC_LEN,
            ErrorCode::AccountDidNotDeserialize
        );
        require!(
            span[..Self::DISC_LEN] == Self::DISCRIMINATOR,
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(Self(&span[Self::DISC_LEN..]))
    }
}

impl<'a> SeedPrefix for PostedVaaV1<'a> {
    const SEED_PREFIX: &'static [u8] = state::POSTED_VAA_V1_SEED_PREFIX;
}
