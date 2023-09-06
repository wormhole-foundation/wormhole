use crate::{state, types::Timestamp};
use anchor_lang::prelude::{require, ErrorCode, Pubkey};

/// Account used to store a published (reusable) Wormhole message.
pub struct PostedMessageV1Unreliable<'a>(&'a [u8]);

impl<'a> PostedMessageV1Unreliable<'a> {
    pub const DISCRIMINATOR: [u8; 4] = state::POSTED_MESSAGE_V1_UNRELIABLE_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 91;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    /// Level of consistency requested by the emitter.
    pub fn consistency_level(&self) -> u8 {
        self.0[0]
    }

    /// Authority used to write the message. This field is set to default when the message is
    /// posted.
    pub fn emitter_authority(&self) -> Pubkey {
        Pubkey::try_from(&self.0[1..33]).unwrap()
    }

    /// If a large message is been written, this is the expected length of the message. When this
    /// message is posted, this value will be overwritten as zero.
    pub fn status(&self) -> state::MessageStatus {
        anchor_lang::AnchorDeserialize::deserialize(&mut &self.0[33..34]).unwrap()
    }

    /// Time the posted message was created.
    pub fn posted_timestamp(&self) -> Timestamp {
        u32::from_le_bytes(self.0[37..41].try_into().unwrap()).into()
    }

    /// Unique id for this message.
    pub fn nonce(&self) -> u32 {
        u32::from_le_bytes(self.0[41..45].try_into().unwrap())
    }

    /// Sequence number of this message.
    pub fn sequence(&self) -> u64 {
        u64::from_le_bytes(self.0[45..53].try_into().unwrap())
    }

    /// Emitter of the message. This may either be the emitter authority or a program ID.
    pub fn emitter(&self) -> Pubkey {
        Pubkey::try_from(&self.0[55..87]).unwrap()
    }

    pub fn payload_size(&self) -> usize {
        u32::from_le_bytes(self.0[87..Self::PAYLOAD_START].try_into().unwrap())
            .try_into()
            .unwrap()
    }

    /// Encoded message.
    pub fn payload(&self) -> &'a [u8] {
        &self.0[Self::PAYLOAD_START..]
    }

    /// Parse account data assumed to match the
    /// [PostedMessageV1Unreliable](state::PostedMessageV1Unreliable) schema.
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
