mod unreliable;
pub use unreliable::*;

use std::ops::{Deref, DerefMut};

use crate::{
    error::CoreBridgeError,
    legacy::utils::LegacyAccount,
    types::{ChainIdSolanaOnly, Timestamp},
};
use anchor_lang::prelude::*;

/// Status of a message. When a message is posted, its status is
/// [Published](MessageStatus::Published).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
#[repr(u8)]
pub enum MessageStatus {
    /// When a message is posted, this status is set. When the guardians observe this message
    /// account, it makes sure that this status is set before attesting to its observation.
    ///
    /// NOTE: This enum value being the first one is important for the legacy implementation.
    /// Originally, where this value lives in the message account was always zero because this data
    /// was never used for anything. This data is now repurposed for crafting large Wormhole
    /// messages.
    Published,
    /// Message is still being written to by the emitter authority.
    ///
    /// NOTE: The message account can be closed when this status is set.
    Writing,
    /// Emitter authority is finished writing and this message is ready to be published via post
    /// message instruction.
    ///
    /// NOTE: The message account cannot be closed when this status is set.
    ReadyForPublishing,
}

/// Message metadata defining information about a published Wormhole message.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct PostedMessageV1Info {
    /// Level of consistency requested by the emitter.
    pub consistency_level: u8,

    /// Authority used to write the message. This field is set to default when the message is
    /// posted.
    pub emitter_authority: Pubkey,

    /// If a message is being written to, this status is used to determine which state this
    /// account is in (e.g. [MessageStatus::Writing] indicates that the emitter authority is still
    /// writing its message to this account). When this message is posted, this value will be
    /// set to [MessageStatus::Published].
    pub status: MessageStatus,

    /// No data is stored here.
    pub _gap_0: [u8; 3],

    /// Time the posted message was created.
    pub posted_timestamp: Timestamp,

    /// Unique id for this message.
    pub nonce: u32,

    /// Sequence number of this message.
    pub sequence: u64,

    /// Always `1`.
    ///
    /// NOTE: Saving this value is silly, but we are keeping it to be consistent with how the posted
    /// message account is written.
    pub solana_chain_id: ChainIdSolanaOnly,

    /// Emitter of the message. This may either be the emitter authority or a program ID.
    pub emitter: Pubkey,
}

/// Underlying data for either [PostedMessageV1](crate::legacy::state::PostedMessageV1) or
/// [PostedMessageV1Unreliable](crate::legacy::state::PostedMessageV1Unreliable).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct PostedMessageV1Data {
    /// Message metadata.
    pub info: PostedMessageV1Info,

    /// Encoded message.
    pub payload: Vec<u8>,
}

impl Deref for PostedMessageV1Data {
    type Target = PostedMessageV1Info;

    fn deref(&self) -> &Self::Target {
        &self.info
    }
}

impl DerefMut for PostedMessageV1Data {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.info
    }
}

impl PostedMessageV1Data {
    pub(crate) fn compute_size(payload_len: usize) -> usize {
        4
        + PostedMessageV1Info::INIT_SPACE
        + 4 // payload.len()
        + payload_len
    }
}

/// Account used to store a published Wormhole message.
///
/// NOTE: If your integration requires reusable message accounts, please see
/// [PostedMessageV1Unreliable](crate::legacy::state::PostedMessageV1Unreliable).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct PostedMessageV1 {
    /// Message data.
    pub data: PostedMessageV1Data,
}

impl PostedMessageV1 {
    pub const PAYLOAD_START: usize = 4 // DISCRIMINATOR
        + PostedMessageV1Info::INIT_SPACE
        + 4 // payload.len()
        ;

    pub(crate) fn compute_size(payload_len: usize) -> usize {
        PostedMessageV1Data::compute_size(payload_len)
    }

    pub(crate) fn require_draft_message(
        acc_info: &AccountInfo,
        emitter_authority: &Signer,
    ) -> Result<bool> {
        let data = acc_info.try_borrow_data()?;
        require!(
            data.len() > 4 && data[..4] == *Self::DISCRIMINATOR,
            ErrorCode::AccountDidNotDeserialize
        );

        require_keys_eq!(
            Self::emitter_authority_unsafe(&data),
            emitter_authority.key(),
            CoreBridgeError::EmitterAuthorityMismatch
        );

        require!(
            Self::status_unsafe(&data) == MessageStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        Ok(true)
    }

    pub(crate) fn emitter_authority_unsafe(data: &[u8]) -> Pubkey {
        TryFrom::try_from(&data[5..37]).unwrap()
    }

    pub(crate) fn status_unsafe(data: &[u8]) -> MessageStatus {
        AnchorDeserialize::deserialize(&mut &data[37..38]).unwrap()
    }

    pub(crate) fn payload_size_unsafe(data: &[u8]) -> u32 {
        u32::from_le_bytes(
            data[(Self::PAYLOAD_START - 4)..Self::PAYLOAD_START]
                .try_into()
                .unwrap(),
        )
    }

    pub(crate) fn emitter_unsafe(data: &[u8]) -> Pubkey {
        TryFrom::try_from(&data[59..91]).unwrap()
    }

    pub(crate) fn try_deserialize_info(acc_info: &AccountInfo) -> Result<PostedMessageV1Info> {
        let data = acc_info.try_borrow_data()?;
        require!(
            data.len() > 4 && data[..4] == *Self::DISCRIMINATOR,
            ErrorCode::AccountDidNotDeserialize
        );

        AnchorDeserialize::deserialize(&mut &data[4..]).map_err(Into::into)
    }
}

impl From<PostedMessageV1Data> for PostedMessageV1 {
    fn from(value: PostedMessageV1Data) -> Self {
        Self { data: value }
    }
}

impl LegacyAccount for PostedMessageV1 {
    const DISCRIMINATOR: &'static [u8] = b"msg\x00";

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl Deref for PostedMessageV1 {
    type Target = PostedMessageV1Data;

    fn deref(&self) -> &Self::Target {
        &self.data
    }
}

impl DerefMut for PostedMessageV1 {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.data
    }
}
