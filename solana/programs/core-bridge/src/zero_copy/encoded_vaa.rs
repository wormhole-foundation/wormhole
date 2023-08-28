use crate::{error::CoreBridgeError, state, types::VaaVersion};
use anchor_lang::{
    prelude::{error, require, AnchorDeserialize, ErrorCode, Pubkey, Result},
    Discriminator,
};
use wormhole_raw_vaas::Vaa;

pub struct EncodedVaa<'a>(&'a [u8]);

impl<'a> EncodedVaa<'a> {
    pub const DISCRIMINATOR: [u8; 8] = state::EncodedVaa::DISCRIMINATOR;
    pub const VAA_START: usize = state::EncodedVaa::BYTES_START - Self::DISC_LEN;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

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

    pub fn vaa_size(&self) -> usize {
        let mut buf = &self.0[34..38];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    pub fn v1(&self) -> Result<Vaa<'a>> {
        require!(
            self.version() == VaaVersion::V1,
            CoreBridgeError::InvalidVaaVersion
        );
        Ok(Vaa::parse(&self.0[Self::VAA_START..]).unwrap())
    }

    pub(crate) fn v1_unverified(&self) -> Result<Vaa<'a>> {
        Vaa::parse(&self.0[Self::VAA_START..]).map_err(|_| error!(CoreBridgeError::CannotParseVaa))
    }

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        let vaa = Self::parse_unverified(span)?;

        // We only allow verified VAAs to be read.
        require!(
            vaa.status() == state::ProcessingStatus::Verified,
            CoreBridgeError::UnverifiedVaa
        );

        Ok(vaa)
    }

    pub(crate) fn parse_unverified(span: &'a [u8]) -> Result<Self> {
        require!(
            span.len() > Self::DISC_LEN,
            ErrorCode::AccountDiscriminatorNotFound
        );
        require!(
            span[..Self::DISC_LEN] == Self::DISCRIMINATOR,
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(Self(&span[Self::DISC_LEN..]))
    }

    pub fn try_v1(span: &'a [u8]) -> Result<Vaa<'a>> {
        Self::parse(span)?.v1()
    }
}
