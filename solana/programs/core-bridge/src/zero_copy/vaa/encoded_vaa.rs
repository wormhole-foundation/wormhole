use std::cell::Ref;

use crate::{error::CoreBridgeError, state};
use anchor_lang::{
    prelude::{
        err, error, require, require_eq, require_keys_eq, AccountInfo, AnchorDeserialize,
        ErrorCode, Pubkey, Result,
    },
    Discriminator,
};
use wormhole_raw_vaas::Vaa;

/// Account used to warehouse VAA buffer.
pub struct EncodedVaa<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> EncodedVaa<'a> {
    pub const DISC: [u8; 8] = state::EncodedVaa::DISCRIMINATOR;
    pub const VAA_START: usize = state::EncodedVaa::VAA_START;

    pub fn discriminator(&self) -> [u8; 8] {
        self.0[..8].try_into().unwrap()
    }

    /// Processing status. **This encoded VAA is only considered usable when this status is set
    /// to [Verified](state::ProcessingStatus::Verified).**
    pub fn status(&self) -> state::ProcessingStatus {
        AnchorDeserialize::deserialize(&mut &self.0[8..9]).unwrap()
    }

    /// The authority that has write privilege to this account.
    pub fn write_authority(&self) -> Pubkey {
        Pubkey::try_from(&self.0[9..41]).unwrap()
    }

    /// VAA version. Only when the VAA is verified is this version set to something that is not
    /// [Unset](VaaVersion::Unset).
    pub fn version(&self) -> u8 {
        self.0[41]
    }

    pub fn vaa_size(&self) -> usize {
        let mut buf = &self.0[42..Self::VAA_START];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    pub fn buf(&self) -> &[u8] {
        &self.0[Self::VAA_START..]
    }

    pub fn as_vaa(&self) -> Result<state::VaaVersion> {
        match self.version() {
            1 => Ok(state::VaaVersion::V1(
                Vaa::parse(&self.0[Self::VAA_START..]).unwrap(),
            )),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub(super) fn new(acc_info: &'a AccountInfo) -> Result<Self> {
        let parsed = Self(acc_info.try_borrow_data()?);
        parsed.require_correct_size()?;

        // We only allow verified VAAs to be read.
        require!(
            parsed.status() == state::ProcessingStatus::Verified,
            CoreBridgeError::UnverifiedVaa
        );
        Ok(parsed)
    }

    pub(crate) fn parse_unverified(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let parsed = Self(acc_info.try_borrow_data()?);
        parsed.require_correct_size()?;

        require!(
            parsed.discriminator() == Self::DISC,
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(parsed)
    }

    fn require_correct_size(&self) -> Result<()> {
        require!(
            self.0.len() >= Self::VAA_START,
            ErrorCode::AccountDidNotDeserialize
        );
        require_eq!(
            self.0.len(),
            Self::VAA_START + self.vaa_size(),
            ErrorCode::AccountDidNotDeserialize
        );
        Ok(())
    }
}

impl<'a> crate::zero_copy::LoadZeroCopy<'a> for EncodedVaa<'a> {
    fn load(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let parsed = Self::new(acc_info)?;

        require!(
            parsed.discriminator() == Self::DISC,
            ErrorCode::AccountDiscriminatorMismatch
        );

        Ok(parsed)
    }
}
