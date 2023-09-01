use crate::{state, types::Timestamp};
use anchor_lang::{
    prelude::{require, ErrorCode, Pubkey},
    solana_program::keccak,
};
use wormhole_solana_common::SeedPrefix;

pub struct PostedVaaV1<'a>(&'a [u8]);

impl<'a> PostedVaaV1<'a> {
    pub const DISCRIMINATOR: [u8; 4] = state::POSTED_VAA_V1_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 91;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    pub fn consistency_level(&self) -> u8 {
        self.0[0]
    }

    pub fn timestamp(&self) -> Timestamp {
        u32::from_le_bytes(self.0[1..5].try_into().unwrap()).into()
    }

    pub fn signature_set(&self) -> Pubkey {
        Pubkey::try_from(&self.0[5..37]).unwrap()
    }

    pub fn guardian_set_index(&self) -> u32 {
        u32::from_le_bytes(self.0[37..41].try_into().unwrap())
    }

    pub fn nonce(&self) -> u32 {
        u32::from_le_bytes(self.0[41..45].try_into().unwrap())
    }

    pub fn sequence(&self) -> u64 {
        u64::from_le_bytes(self.0[45..53].try_into().unwrap())
    }

    pub fn emitter_chain(&self) -> u16 {
        u16::from_le_bytes(self.0[53..55].try_into().unwrap())
    }

    pub fn emitter_address(&self) -> [u8; 32] {
        self.0[55..87].try_into().unwrap()
    }

    pub fn payload_size(&self) -> usize {
        u32::from_le_bytes(self.0[87..Self::PAYLOAD_START].try_into().unwrap())
            .try_into()
            .unwrap()
    }

    pub fn payload(&self) -> &'a [u8] {
        &self.0[Self::PAYLOAD_START..]
    }
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

    pub fn digest(&self) -> keccak::Hash {
        keccak::hash(self.message_hash().as_ref())
    }

    pub fn parse(span: &'a [u8]) -> anchor_lang::Result<Self> {
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
