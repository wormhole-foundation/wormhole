use crate::{state, types::Timestamp};
use anchor_lang::prelude::{require, AnchorDeserialize, ErrorCode, Pubkey, Result};

pub struct PostedMessageV1<'a>(&'a [u8]);

impl<'a> PostedMessageV1<'a> {
    pub const DISCRIMINATOR: [u8; 4] = state::POSTED_MESSAGE_V1_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 91;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    pub fn consistency_level(&self) -> u8 {
        let mut buf = &self.0[..1];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn emitter_authority(&self) -> Pubkey {
        let mut buf = &self.0[1..33];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn status(&self) -> state::MessageStatus {
        let mut buf = &self.0[33..34];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn posted_timestamp(&self) -> Timestamp {
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

    pub fn emitter(&self) -> Pubkey {
        let mut buf = &self.0[55..87];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn payload_size(&self) -> usize {
        let mut buf = &self.0[87..Self::PAYLOAD_START];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    pub fn payload(&self) -> &'a [u8] {
        &self.0[Self::PAYLOAD_START..]
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
