use std::cell::Ref;

use crate::{error::CoreBridgeError, state};
use anchor_lang::prelude::*;
use solana_program::keccak;
use wormhole_raw_vaas::Vaa;

pub(super) const ENCODED_VAA_DISCRIMINATOR: [u8; 8] =
    <state::EncodedVaa as anchor_lang::Discriminator>::DISCRIMINATOR;
const VAA_START: usize = state::EncodedVaa::VAA_START;

/// Account used to warehouse VAA buffer.
pub struct EncodedVaa<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> EncodedVaa<'a> {
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
        let mut buf = &self.0[42..VAA_START];
        u32::deserialize(&mut buf).unwrap().try_into().unwrap()
    }

    pub fn buf(&self) -> &[u8] {
        &self.0[VAA_START..]
    }

    pub fn as_vaa(&self) -> Result<state::VaaVersion> {
        match self.version() {
            1 => Ok(state::VaaVersion::V1(
                Vaa::parse(&self.0[VAA_START..]).unwrap(),
            )),
            _ => err!(CoreBridgeError::InvalidVaaVersion),
        }
    }

    /// Recompute the message hash.
    pub fn message_hash(&self) -> Result<keccak::Hash> {
        match self.as_vaa()? {
            state::VaaVersion::V1(vaa) => Ok(keccak::hash(vaa.body().as_ref())),
        }
    }

    /// Compute digest (hash of [message_hash](Self::message_hash)).
    pub fn digest(&self) -> Result<keccak::Hash> {
        Ok(keccak::hash(self.message_hash()?.as_ref()))
    }

    pub(super) fn new(acc_info: &'a AccountInfo) -> Result<Self> {
        let parsed = Self(acc_info.try_borrow_data()?);

        // We only allow verified VAAs to be read.
        require!(
            parsed.status() == state::ProcessingStatus::Verified,
            CoreBridgeError::UnverifiedVaa
        );
        Ok(parsed)
    }
}
