use std::cell::Ref;

use crate::{error::CoreBridgeError, state, types::VaaVersion};
use anchor_lang::{
    prelude::{
        err, error, require, require_eq, require_gte, require_keys_eq, AccountInfo,
        AnchorDeserialize, ErrorCode, Pubkey, Result,
    },
    Discriminator,
};
use wormhole_raw_vaas::Vaa;

/// Account used to warehouse VAA buffer.
pub struct EncodedVaa<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> EncodedVaa<'a> {
    pub const DISC: [u8; 8] = state::EncodedVaa::DISCRIMINATOR;
    pub const VAA_START: usize = state::EncodedVaa::VAA_START;

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
    pub fn version(&self) -> VaaVersion {
        AnchorDeserialize::deserialize(&mut &self.0[41..42]).unwrap()
    }

    pub fn vaa_size(&self) -> usize {
        let mut buf = &self.0[42..Self::VAA_START];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    pub fn buf(&self) -> &[u8] {
        &self.0[Self::VAA_START..]
    }

    /// Parse account data assumed to match the [EncodedVaa](state::EncodedVaa) schema.
    ///
    /// NOTE: There is no ownership check because [AccountInfo](anchor_lang::prelude::AccountInfo)
    /// is not passed into this method.
    pub fn parse(acc_info: &'a AccountInfo) -> Result<Self> {
        let vaa = Self::parse_unverified(acc_info)?;

        // We only allow verified VAAs to be read.
        require!(
            vaa.status() == state::ProcessingStatus::Verified,
            CoreBridgeError::UnverifiedVaa
        );

        Ok(vaa)
    }

    pub fn try_emitter_chain(&self) -> Result<u16> {
        match self.version() {
            VaaVersion::V1 => parse_v1(self.buf()).map(|v1| v1.body().emitter_chain()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn try_emitter_address(&self) -> Result<[u8; 32]> {
        match self.version() {
            VaaVersion::V1 => parse_v1(self.buf()).map(|v1| v1.body().emitter_address()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn try_sequence(&self) -> Result<u64> {
        match self.version() {
            VaaVersion::V1 => parse_v1(self.buf()).map(|v1| v1.body().sequence()),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    pub fn as_v1(&'a self) -> Result<Vaa<'a>> {
        parse_v1(self.buf())
    }

    /// Be careful with using this method. This method does not check the owner of the account or
    /// check for the size of borrowed account data.
    pub fn parse_unchecked(acc_info: &'a AccountInfo) -> Self {
        Self(acc_info.data.borrow())
    }

    pub(crate) fn parse_unverified(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let data = acc_info.try_borrow_data()?;
        require_gte!(
            data.len(),
            Self::VAA_START,
            ErrorCode::AccountDidNotDeserialize
        );
        require!(
            data[..8] == Self::DISC,
            ErrorCode::AccountDiscriminatorMismatch
        );

        let parsed = Self(data);
        require_eq!(
            parsed.0.len(),
            Self::VAA_START + parsed.vaa_size(),
            ErrorCode::AccountDidNotDeserialize
        );

        Ok(parsed)
    }
}

fn parse_v1(buf: &[u8]) -> Result<Vaa<'_>> {
    Vaa::parse(buf).map_err(|_| error!(CoreBridgeError::CannotParseVaa))
}
