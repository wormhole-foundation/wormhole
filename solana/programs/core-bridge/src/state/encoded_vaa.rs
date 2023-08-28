use std::ops::Deref;

use crate::{error::CoreBridgeError, types::VaaVersion};
use anchor_lang::prelude::*;
use wormhole_raw_vaas::Vaa;

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub enum ProcessingStatus {
    #[default]
    Unset,
    Writing,
    Verified,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Header {
    pub status: ProcessingStatus,
    pub write_authority: Pubkey,
    pub version: VaaVersion,
}

#[account]
#[derive(Debug, PartialEq, Eq)]
pub struct EncodedVaa {
    pub header: Header,
    pub buf: Vec<u8>,
}

impl EncodedVaa {
    pub(crate) const BYTES_START: usize = 8 // DISCRIMINATOR
        + crate::state::Header::INIT_SPACE
        + 4 // bytes.len()
    ;

    pub fn v1(&self) -> Result<Vaa> {
        require!(
            self.header.version == VaaVersion::V1,
            CoreBridgeError::InvalidVaaVersion
        );
        Ok(Vaa::parse(&self.buf).unwrap())
    }
}

impl Deref for EncodedVaa {
    type Target = Header;

    fn deref(&self) -> &Self::Target {
        &self.header
    }
}
