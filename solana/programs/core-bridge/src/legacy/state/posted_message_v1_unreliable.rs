use std::ops::{Deref, DerefMut};

use crate::state::PostedMessageV1Data;
use anchor_lang::prelude::*;

pub(crate) const POSTED_MESSAGE_V1_UNRELIABLE_DISCRIMINATOR: [u8; 4] = *b"msu\x00";

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq)]
pub struct PostedMessageV1Unreliable {
    pub data: PostedMessageV1Data,
}

impl Owner for PostedMessageV1Unreliable {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl crate::legacy::utils::LegacyDiscriminator<4> for PostedMessageV1Unreliable {
    const LEGACY_DISCRIMINATOR: [u8; 4] = POSTED_MESSAGE_V1_UNRELIABLE_DISCRIMINATOR;
}

impl PostedMessageV1Unreliable {
    pub(crate) fn compute_size(payload_len: usize) -> usize {
        PostedMessageV1Data::compute_size(payload_len)
    }
}

impl Deref for PostedMessageV1Unreliable {
    type Target = PostedMessageV1Data;

    fn deref(&self) -> &Self::Target {
        &self.data
    }
}

impl DerefMut for PostedMessageV1Unreliable {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.data
    }
}
