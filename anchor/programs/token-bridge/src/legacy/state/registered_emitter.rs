use anchor_lang::prelude::*;
use core_bridge_program::types::{ChainId, ExternalAddress};
use wormhole_common::{legacy_account, LegacyDiscriminator};

#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct RegisteredEmitter {
    pub chain: ChainId,
    pub contract: ExternalAddress,
}

impl LegacyDiscriminator<0> for RegisteredEmitter {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}
