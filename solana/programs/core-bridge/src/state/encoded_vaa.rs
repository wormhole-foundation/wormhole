use std::ops::Deref;

use crate::{
    error::CoreBridgeError,
    types::{MessageHash, VaaVersion},
};
use anchor_lang::{prelude::*, Discriminator};
use wormhole_raw_vaas::{Payload, Vaa};

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
        *buf = &buf[8..];
        //let mut data: &[u8] = &buf[8..];
        Self::deserialize(buf).map_err(|_| error!(ErrorCode::AccountDidNotDeserialize))
    }
}

#[account]
#[derive(Debug, PartialEq, Eq)]
pub struct EncodedVaa {
    pub header: ProcessingHeader,
    pub buf: Vec<u8>,
}

impl EncodedVaa {
    pub const BYTES_START: usize = 8 // DISCRIMINATOR
        + crate::state::ProcessingHeader::INIT_SPACE
        + 4 // bytes.len()
        ;

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
}

impl Deref for EncodedVaa {
    type Target = ProcessingHeader;

    fn deref(&self) -> &Self::Target {
        &self.header
    }
}

pub struct ZeroCopyEncodedVaa<'a>(&'a [u8]);

impl<'a> ZeroCopyEncodedVaa<'a> {
    pub fn status(&self) -> ProcessingStatus {
        let mut buf = &self.0[..1];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn write_authority(&self) -> Pubkey {
        let mut buf = &self.0[1..33];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn version(&self) -> VaaVersion {
        let mut buf = &self.0[33..34];
        AnchorDeserialize::deserialize(&mut buf).unwrap()
    }

    pub fn vaa_v1(&self) -> Result<Vaa<'a>> {
        require!(
            self.version() == VaaVersion::V1,
            CoreBridgeError::InvalidVaaVersion
        );
        Ok(self.vaa_v1_unchecked())
    }

    fn vaa_v1_unchecked(&self) -> Vaa<'a> {
        Vaa::parse(&self.0[EncodedVaa::BYTES_START..]).unwrap()
    }

    pub fn emitter_chain(&self) -> Result<u16> {
        match self.version() {
            VaaVersion::V1 => Ok(self.vaa_v1_unchecked().body().emitter_chain()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn emitter_address(&self) -> Result<[u8; 32]> {
        match self.version() {
            VaaVersion::V1 => Ok(self.vaa_v1_unchecked().body().emitter_address()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn sequence(&self) -> Result<u64> {
        match self.version() {
            VaaVersion::V1 => Ok(self.vaa_v1_unchecked().body().sequence()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn payload(&self) -> Result<Payload<'a>> {
        match self.version() {
            VaaVersion::V1 => Ok(self.vaa_v1_unchecked().body().payload()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        if span.len() < 8 {
            return err!(ErrorCode::AccountDiscriminatorNotFound);
        }

        require!(
            Self::DISCRIMINATOR == span[..8],
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(Self(&span[8..]))
    }
}

impl<'a> Discriminator for ZeroCopyEncodedVaa<'a> {
    const DISCRIMINATOR: [u8; 8] = EncodedVaa::DISCRIMINATOR;
}
