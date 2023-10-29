use std::cell::Ref;

use crate::{
    state::{self, MessageStatus},
    types::Timestamp,
};
use anchor_lang::prelude::{
    error, require, require_eq, require_keys_eq, AccountInfo, ErrorCode, Pubkey, Result,
};

/// Account used to store a published Wormhole message. There are two types of message accounts,
/// one for reliable messages (discriminator == "msg\0") and unreliable messages (discriminator ==
/// "msu\0").
pub struct PostedMessageV1<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> PostedMessageV1<'a> {
    pub const DISC: [u8; 4] = state::POSTED_MESSAGE_V1_DISCRIMINATOR;
    pub const UNRELIABLE_DISC: [u8; 4] = state::POSTED_MESSAGE_V1_UNRELIABLE_DISCRIMINATOR;
    pub const PAYLOAD_START: usize = 95;

    pub fn discriminator(&self) -> [u8; 4] {
        self.0[..4].try_into().unwrap()
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

    /// If a message is being written to, this status is used to determine which state this
    /// account is in (e.g. [MessageStatus::Writing] indicates that the emitter authority is still
    /// writing its message to this account). When this message is posted, this value will be
    /// set to [MessageStatus::Unset].
    pub fn status(&self) -> MessageStatus {
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

        Ok(parsed)
    }
}

impl<'a> crate::zero_copy::LoadZeroCopy<'a> for PostedMessageV1<'a> {
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
