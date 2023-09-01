use crate::{state, types::Timestamp};
use anchor_lang::prelude::{require, ErrorCode, Pubkey};

pub struct PostedMessageV1Unreliable<'a>(&'a [u8]);

impl<'a> PostedMessageV1Unreliable<'a> {
    pub const DISCRIMINATOR: [u8; 4] = state::POSTED_MESSAGE_V1_UNRELIABLE_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 91;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    pub fn consistency_level(&self) -> u8 {
        self.0[0]
    }

    pub fn emitter_authority(&self) -> Pubkey {
        Pubkey::try_from(&self.0[1..33]).unwrap()
    }

    pub fn status(&self) -> state::MessageStatus {
        anchor_lang::AnchorDeserialize::deserialize(&mut &self.0[33..34]).unwrap()
    }

    pub fn posted_timestamp(&self) -> Timestamp {
        u32::from_le_bytes(self.0[37..41].try_into().unwrap()).into()
    }

    pub fn nonce(&self) -> u32 {
        u32::from_le_bytes(self.0[41..45].try_into().unwrap())
    }

    pub fn sequence(&self) -> u64 {
        u64::from_le_bytes(self.0[45..53].try_into().unwrap())
    }

    pub fn emitter(&self) -> Pubkey {
        Pubkey::try_from(&self.0[55..87]).unwrap()
    }

    pub fn payload_size(&self) -> usize {
        u32::from_le_bytes(self.0[87..Self::PAYLOAD_START].try_into().unwrap())
            .try_into()
            .unwrap()
    }

    pub fn payload(&self) -> &'a [u8] {
        &self.0[Self::PAYLOAD_START..]
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
