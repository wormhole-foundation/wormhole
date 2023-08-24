use std::ops::Deref;

use crate::{
    error::CoreBridgeError,
    types::{MessageHash, VaaVersion},
};
use anchor_lang::{prelude::*, Discriminator};

use super::ProcessingStatus;

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
    pub(crate) const BYTES_START: usize = super::VAA_BUF_START;

    pub fn payload_size(&self) -> Result<usize> {
        match self.version {
            VaaVersion::V1 => Ok(self.buf.len() - self.body_index() - 51),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn compute_message_hash(&self) -> Result<MessageHash> {
        match self.version {
            VaaVersion::V1 => {
                let body = &self.buf[self.body_index()..];
                Ok(solana_program::keccak::hash(body).into())
            }
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    fn body_index(&self) -> usize {
        match self.version {
            VaaVersion::Unset => 0,
            VaaVersion::V1 => 6 + 66 * usize::from(self.buf[5]),
        }
    }

    pub fn try_acc_header_deserialize(buf: &mut &[u8]) -> Result<Header> {
        if buf.len() < 8 {
            return err!(ErrorCode::AccountDiscriminatorNotFound);
        }
        require!(
            EncodedVaa::DISCRIMINATOR == buf[..8],
            ErrorCode::AccountDiscriminatorMismatch
        );
        Self::try_acc_header_deserialize_unchecked(buf)
    }

    pub fn try_acc_header_deserialize_unchecked(buf: &mut &[u8]) -> Result<Header> {
        *buf = &buf[8..];
        Header::deserialize(buf).map_err(|_| error!(ErrorCode::AccountDidNotDeserialize))
    }
}

impl Deref for EncodedVaa {
    type Target = Header;

    fn deref(&self) -> &Self::Target {
        &self.header
    }
}
