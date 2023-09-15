use std::cell::Ref;

use crate::{state, types::Timestamp};
use anchor_lang::prelude::{
    error, require, require_eq, require_gte, require_keys_eq, AccountInfo, ErrorCode, Pubkey,
    Result,
};

/// Account used to store a published Wormhole message. There are two types of message accounts,
/// one for reliable messages (discriminator == "msg\0") and unreliable messages (discriminator ==
/// "msu\0").
pub struct PostedMessageV1<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> PostedMessageV1<'a> {
    pub const RELIABLE_DISC: [u8; 4] = state::POSTED_MESSAGE_V1_DISCRIMINATOR;
    pub const UNRELIABLE_DISC: [u8; 4] = state::POSTED_MESSAGE_V1_UNRELIABLE_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 95;

    pub fn discriminator(&self) -> [u8; 4] {
        self.0[..4].try_into().unwrap()
    }

    pub fn reliable(&self) -> bool {
        self.discriminator() == Self::RELIABLE_DISC
    }

    pub fn unreliable(&self) -> bool {
        self.discriminator() == Self::UNRELIABLE_DISC
    }

    /// Level of consistency requested by the emitter.
    pub fn consistency_level(&self) -> u8 {
        self.0[4]
    }

    /// Authority used to write the message. This field is set to default when the message is
    /// posted.
    pub fn emitter_authority(&self) -> Pubkey {
        Pubkey::try_from(&self.0[5..37]).unwrap()
    }

    /// If a large message is been written, this is the expected length of the message. When this
    /// message is posted, this value will be overwritten as zero.
    pub fn status(&self) -> state::MessageStatus {
        anchor_lang::AnchorDeserialize::deserialize(&mut &self.0[37..38]).unwrap()
    }

    /// Time the posted message was created.
    pub fn posted_timestamp(&self) -> Timestamp {
        u32::from_le_bytes(self.0[41..45].try_into().unwrap()).into()
    }

    /// Unique id for this message.
    pub fn nonce(&self) -> u32 {
        u32::from_le_bytes(self.0[45..49].try_into().unwrap())
    }

    /// Sequence number of this message.
    pub fn sequence(&self) -> u64 {
        u64::from_le_bytes(self.0[49..57].try_into().unwrap())
    }

    /// Emitter of the message. This may either be the emitter authority or a program ID.
    pub fn emitter(&self) -> Pubkey {
        Pubkey::try_from(&self.0[59..91]).unwrap()
    }

    pub fn payload_size(&self) -> usize {
        u32::from_le_bytes(self.0[91..Self::PAYLOAD_START].try_into().unwrap())
            .try_into()
            .unwrap()
    }

    /// Encoded message.
    pub fn payload(&self) -> &[u8] {
        &self.0[Self::PAYLOAD_START..]
    }

    /// Parse account data assumed to match the [PostedMessageV1](state::PostedMessageV1) schema.
    ///
    /// NOTE: There is no ownership check because [AccountInfo](anchor_lang::prelude::AccountInfo)
    /// is not passed into this method.
    pub fn parse_reliable(acc_info: &'a AccountInfo) -> Result<Self> {
        let parsed = Self::parse(acc_info)?;
        require!(
            parsed.discriminator() == Self::RELIABLE_DISC,
            ErrorCode::AccountDidNotDeserialize
        );

        Ok(parsed)
    }

    /// Parse account data assumed to match the [PostedMessageV1](state::PostedMessageV1Unreliable)
    /// schema.
    ///
    /// NOTE: There is no ownership check because [AccountInfo](anchor_lang::prelude::AccountInfo)
    /// is not passed into this method.
    pub fn parse_unreliable(acc_info: &'a AccountInfo) -> Result<Self> {
        let parsed = Self::parse(acc_info)?;
        require!(
            parsed.discriminator() == Self::UNRELIABLE_DISC,
            ErrorCode::AccountDidNotDeserialize
        );

        Ok(parsed)
    }

    pub(crate) fn parse_unchecked(acc_info: &'a AccountInfo) -> Self {
        Self(acc_info.data.borrow())
    }

    fn parse(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let data = acc_info.try_borrow_data()?;
        require_gte!(
            data.len(),
            Self::PAYLOAD_START,
            ErrorCode::AccountDidNotDeserialize
        );

        let parsed = Self(data);
        require_eq!(
            parsed.0.len(),
            Self::PAYLOAD_START + parsed.payload_size(),
            ErrorCode::AccountDidNotDeserialize
        );

        Ok(parsed)
    }
}
