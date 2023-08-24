use crate::{error::CoreBridgeError, types::VaaVersion};
use anchor_lang::prelude::{err, require, AnchorDeserialize, ErrorCode, Pubkey, Result};
use wormhole_raw_vaas::Vaa;

use super::ProcessingStatus;

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

    pub fn v1(&self) -> Result<Vaa<'a>> {
        require!(
            self.version() == VaaVersion::V1,
            CoreBridgeError::InvalidVaaVersion
        );
        Ok(Vaa::parse(&self.0[super::VAA_BUF_START..]).unwrap())
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        if span.len() < 8 {
            return err!(ErrorCode::AccountDiscriminatorNotFound);
        }

        require!(
            super::DISCRIMINATOR == span[..8],
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(Self(&span[8..]))
    }

    pub fn try_v1(span: &'a [u8]) -> Result<Vaa<'a>> {
        let acc = Self::parse(span)?;
        acc.v1()
    }
}
