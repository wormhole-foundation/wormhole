use std::ops::Deref;

use anchor_lang::prelude::*;
use wormhole_raw_vaas::Vaa;

use crate::error::CoreBridgeError;

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
    /// VAA version. Only when the VAA is verified is this version set to a value.
    pub version: u8,
}

/// Representation of VAA versions.
#[non_exhaustive]
pub enum VaaVersion<'a> {
    V1(Vaa<'a>),
}

impl<'a> VaaVersion<'a> {
    pub fn v1(&'a self) -> Option<&'a Vaa<'a>> {
        match self {
            Self::V1(inner) => Some(inner),
        }
    }

    pub fn to_v1(self) -> Result<Vaa<'a>> {
        match self {
            Self::V1(inner) => Ok(inner),
        }
    }
}

impl<'a> AsRef<[u8]> for VaaVersion<'a> {
    fn as_ref(&self) -> &[u8] {
        match self {
            Self::V1(inner) => inner.as_ref(),
        }
    }
}

/// Account used to warehouse VAA buffer.
///
/// NOTE: This account should not be used by an external application unless the header's status is
/// `Verified`. It is encouraged to use the `EncodedVaa` zero-copy account struct instead.
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
    pub(crate) const VAA_START: usize = 8 // DISCRIMINATOR
        + crate::state::Header::INIT_SPACE
        + 4 // bytes.len()
    ;

    /// Return as [VaaVersion] if the version number is valid.
    pub fn as_vaa(&self) -> Result<VaaVersion> {
        match self.version {
            1 => Ok(VaaVersion::V1(Vaa::parse(&self.buf).unwrap())),
            _ => err!(CoreBridgeError::UnverifiedVaa),
        }
    }

    pub(crate) fn require_draft_vaa(
        acc_info: &AccountInfo,
        write_authority: &Signer,
    ) -> Result<bool> {
        let data = acc_info.try_borrow_data()?;
        require!(
            data.len() > 8 && data[..8] == <Self as anchor_lang::Discriminator>::DISCRIMINATOR,
            ErrorCode::AccountDidNotDeserialize
        );

        require!(
            Self::status_unsafe(&data) == ProcessingStatus::Writing,
            CoreBridgeError::NotInWritingStatus
        );

        require_keys_eq!(
            Self::write_authority_unsafe(&data),
            write_authority.key(),
            CoreBridgeError::WriteAuthorityMismatch
        );

        Ok(true)
    }

    pub(crate) fn status_unsafe(data: &[u8]) -> ProcessingStatus {
        AnchorDeserialize::deserialize(&mut &data[8..9]).unwrap()
    }

    pub(crate) fn write_authority_unsafe(data: &[u8]) -> Pubkey {
        TryFrom::try_from(&data[9..41]).unwrap()
    }

    pub(crate) fn payload_size_unsafe(data: &[u8]) -> u32 {
        u32::from_le_bytes(
            data[(Self::VAA_START - 4)..Self::VAA_START]
                .try_into()
                .unwrap(),
        )
    }

    pub(crate) fn try_deserialize_header(acc_info: &AccountInfo) -> Result<Header> {
        let data = acc_info.try_borrow_data()?;
        require!(
            data.len() > 8 && data[..8] == <Self as anchor_lang::Discriminator>::DISCRIMINATOR,
            ErrorCode::AccountDidNotDeserialize
        );

        AnchorDeserialize::deserialize(&mut &data[8..]).map_err(Into::into)
    }
}

impl Deref for EncodedVaa {
    type Target = Header;

    fn deref(&self) -> &Self::Target {
        &self.header
    }
}
