use crate::{error::CoreBridgeError, state, types::VaaVersion};
use anchor_lang::{
    prelude::{error, require, AnchorDeserialize, ErrorCode, Pubkey, Result},
    Discriminator,
};
use wormhole_raw_vaas::Vaa;

/// Account used to warehouse VAA buffer.
pub struct EncodedVaa<'a>(&'a [u8]);

impl<'a> EncodedVaa<'a> {
    pub const DISCRIMINATOR: [u8; 8] = state::EncodedVaa::DISCRIMINATOR;
    pub const VAA_START: usize = state::EncodedVaa::BYTES_START - Self::DISC_LEN;

    const DISC_LEN: usize = Self::DISCRIMINATOR.len();

    /// Processing status. **This encoded VAA is only considered usable when this status is set
    /// to [Verified](state::ProcessingStatus::Verified).**
    pub fn status(&self) -> state::ProcessingStatus {
        AnchorDeserialize::deserialize(&mut &self.0[..1]).unwrap()
    }

    /// The authority that has write privilege to this account.
    pub fn write_authority(&self) -> Pubkey {
        Pubkey::try_from(&self.0[1..33]).unwrap()
    }

    /// VAA version. Only when the VAA is verified is this version set to something that is not
    /// [Unset](VaaVersion::Unset).
    pub fn version(&self) -> VaaVersion {
        AnchorDeserialize::deserialize(&mut &self.0[33..34]).unwrap()
    }

    pub fn vaa_size(&self) -> usize {
        let mut buf = &self.0[34..Self::VAA_START];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    /// VAA (Version 1).
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

    /// Parse account data assumed to match the [EncodedVaa](state::EncodedVaa) schema.
    ///
    /// NOTE: There is no ownership check because [AccountInfo](anchor_lang::prelude::AccountInfo)
    /// is not passed into this method.
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

    /// Method to try to deserialize account data as VAA (Version 1).
    pub fn try_v1(span: &'a [u8]) -> Result<Vaa<'a>> {
        Self::parse(span)?.v1()
    }
}
