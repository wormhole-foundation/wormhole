use crate::{error::CoreBridgeError, state, types::VaaVersion};
use anchor_lang::{
    prelude::{require, AnchorDeserialize, ErrorCode, Pubkey, Result},
    Discriminator,
};
use wormhole_raw_vaas::Vaa;

pub struct EncodedVaa<'a>(&'a [u8]);

impl<'a> EncodedVaa<'a> {
    pub fn status(&self) -> state::ProcessingStatus {
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
        Ok(Vaa::parse(&self.0[state::EncodedVaa::BYTES_START..]).unwrap())
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        const DISC_LEN: usize = state::EncodedVaa::DISCRIMINATOR.len();

        require!(
            span.len() > DISC_LEN,
            ErrorCode::AccountDiscriminatorNotFound
        );
        require!(
            state::EncodedVaa::DISCRIMINATOR == span[..DISC_LEN],
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(Self(&span[DISC_LEN..]))
    }

    pub fn try_v1(span: &'a [u8]) -> Result<Vaa<'a>> {
        Self::parse(span)?.v1()
    }
}
