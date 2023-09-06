use std::ops::Deref;

use crate::{error::CoreBridgeError, types::VaaVersion};
use anchor_lang::prelude::*;
use wormhole_raw_vaas::Vaa;

/// Encoded VAA's processing status.
#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub enum ProcessingStatus {
    /// `EncodedVaa` account is uninitialized.
    #[default]
    Unset,
    /// VAA is still being written to the `EncodedVaa` account.
    Writing,
    /// VAA is verified (i.e. validating message attestation is complete).
    Verified,
}

/// `EncodedVaa` account header.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Header {
    /// Processing status. **This encoded VAA is only considered usable when this status is set
    /// to [Verified](ProcessingStatus::Verified).**
    pub status: ProcessingStatus,
    /// The authority that has write privilege to this account.
    pub write_authority: Pubkey,
    /// VAA version. Only when the VAA is verified is this version set to something that is not
    /// [Unset](VaaVersion::Unset).
    pub version: VaaVersion,
}

/// Account used to warehouse VAA buffer.
///
/// NOTE: This account should not be used by an external application unless the header's status is
/// `Verified`. It is encouraged to use the `EncodedVaa` zero-copy account struct instead. See
/// [zero_copy](mod@crate::zero_copy) for more info.
#[account]
#[derive(Debug, PartialEq, Eq)]
pub struct EncodedVaa {
    /// Status, write authority and VAA version.
    pub header: Header,
    /// VAA buffer.
    pub buf: Vec<u8>,
}

impl EncodedVaa {
    /// Index of the first byte of the VAA buffer.
    pub(crate) const BYTES_START: usize = 8 // DISCRIMINATOR
        + crate::state::Header::INIT_SPACE
        + 4 // bytes.len()
    ;

    /// Return VAA as zero-copy reader.
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
