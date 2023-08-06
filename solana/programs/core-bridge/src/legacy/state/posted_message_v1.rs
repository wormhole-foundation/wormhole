use std::ops::{Deref, DerefMut};

use crate::{constants::SOLANA_CHAIN, types::Timestamp};
use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, NewAccountSize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub enum MessageStatus {
    Unset,
    Writing,
}

/// This type is kind of silly. But because `PostedMessageV1` has the emitter chain ID as a field,
/// which is unnecessary since it's always Solana's chain ID, we use this type to guarantee that the
/// encoded chain ID is always `1`.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct ChainIdSolanaOnly {
    chain_id: u16,
}

impl Default for ChainIdSolanaOnly {
    fn default() -> Self {
        Self {
            chain_id: SOLANA_CHAIN,
        }
    }
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct PostedMessageV1Info {
    /// Level of consistency requested by the emitter.
    pub consistency_level: u8,

    /// TODO: Fix comment.
    pub emitter_authority: Pubkey,

    /// If a large message is been written, this is the expected length of the message. When this
    /// message is posted, this value will be overwritten as zero.
    //pub expected_msg_length: u16,
    pub status: MessageStatus,

    /// No data is stored here.
    pub _gap_0: [u8; 3],

    /// Time the posted message was created.
    pub posted_timestamp: Timestamp,

    /// Unique id for this message.
    pub nonce: u32,

    /// Sequence number of this message.
    pub sequence: u64,

    /// NOTE: Saving this value is silly, but we are keeping it to be consistent with how the posted
    /// message account is written. This should always equal Solana's chain ID (i.e. `1`).
    pub solana_chain_id: ChainIdSolanaOnly,

    /// Emitter of the message.
    pub emitter: Pubkey,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct PostedMessageV1Data {
    pub info: PostedMessageV1Info,

    /// encoded message.
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

impl NewAccountSize for PostedMessageV1Data {
    fn compute_size(payload_len: usize) -> usize {
        4
        + PostedMessageV1Info::INIT_SPACE
        + 4 // payload.len()
        + payload_len
    }
}

#[legacy_account]
#[derive(Debug, PartialEq, Eq)]
pub struct PostedMessageV1 {
    pub data: PostedMessageV1Data,
}

impl PostedMessageV1 {
    pub(crate) const BYTES_START: usize = 4 // LEGACY_DISCRIMINATOR
        + PostedMessageV1Info::INIT_SPACE
        + 4 // payload.len()
        ;
}

impl LegacyDiscriminator<4> for PostedMessageV1 {
    const LEGACY_DISCRIMINATOR: [u8; 4] = *b"msg\x00";
}

impl NewAccountSize for PostedMessageV1 {
    fn compute_size(payload_len: usize) -> usize {
        PostedMessageV1Data::compute_size(payload_len)
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
