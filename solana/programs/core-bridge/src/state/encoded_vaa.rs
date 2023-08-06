use std::ops::Deref;

use crate::{
    error::CoreBridgeError,
    types::{MessageHash, VaaVersion},
};
use anchor_lang::{prelude::*, Discriminator};

#[derive(
    Default, Copy, Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace,
)]
pub enum ProcessingStatus {
    #[default]
    Unset,
    Writing,
    HashComputed,
    Verified,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct ProcessingHeader {
    pub status: ProcessingStatus,
    pub write_authority: Pubkey,
    pub version: VaaVersion,
}

impl ProcessingHeader {
    pub(crate) fn try_account_serialize<W: std::io::Write>(&self, writer: &mut W) -> Result<()> {
        EncodedVaa::DISCRIMINATOR.serialize(writer)?;
        self.serialize(writer).map_err(Into::into)
    }

    pub fn try_account_deserialize(buf: &mut &[u8]) -> Result<Self> {
        if buf.len() < 8 {
            return err!(ErrorCode::AccountDiscriminatorNotFound);
        }
        require!(
            EncodedVaa::DISCRIMINATOR == buf[..8],
            ErrorCode::AccountDiscriminatorMismatch
        );
        Self::try_account_deserialize_unchecked(buf)
    }

    pub fn try_account_deserialize_unchecked(buf: &mut &[u8]) -> Result<Self> {
        let mut data: &[u8] = &buf[8..];
        AnchorDeserialize::deserialize(&mut data)
            .map_err(|_| error!(ErrorCode::AccountDidNotDeserialize))
    }
}

#[account]
#[derive(Debug, PartialEq, Eq)]
pub struct EncodedVaa {
    pub header: ProcessingHeader,
    pub bytes: Vec<u8>,
}

impl EncodedVaa {
    pub const BYTES_START: usize = 8 // DISCRIMINATOR
        + crate::state::ProcessingHeader::INIT_SPACE
        + 4 // bytes.len()
        ;

    pub fn payload_size(&self) -> Result<usize> {
        match self.version {
            VaaVersion::V1 => Ok(self.bytes.len() - self.body_index() - 51),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn compute_message_hash(&self) -> Result<MessageHash> {
        match self.version {
            VaaVersion::V1 => {
                let body = &self.bytes[self.body_index()..];
                Ok(solana_program::keccak::hash(body).into())
            }
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    fn body_index(&self) -> usize {
        match self.version {
            VaaVersion::Unset => 0,
            VaaVersion::V1 => 6 + 66 * usize::from(self.bytes[5]),
        }
    }
}

impl Deref for EncodedVaa {
    type Target = ProcessingHeader;

    fn deref(&self) -> &Self::Target {
        &self.header
    }
}
