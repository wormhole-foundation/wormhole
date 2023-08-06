use std::ops::{Deref, DerefMut};

use crate::state::PostedMessageV1Data;
use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator, NewAccountSize};

#[legacy_account]
#[derive(Debug, PartialEq, Eq)]
pub struct PostedMessageV1Unreliable {
    pub data: PostedMessageV1Data,
}

impl LegacyDiscriminator<4> for PostedMessageV1Unreliable {
    const LEGACY_DISCRIMINATOR: [u8; 4] = *b"msu\x00";
}

impl NewAccountSize for PostedMessageV1Unreliable {
    fn compute_size(payload_len: usize) -> usize {
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
