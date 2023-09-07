mod encoded_vaa;
pub use encoded_vaa::*;

mod posted_vaa_v1;
pub use posted_vaa_v1::*;

use anchor_lang::prelude::{err, error, require, require_keys_eq, AccountInfo, ErrorCode, Result};
use wormhole_raw_vaas::Payload;

use crate::state::VaaVersion;

use super::LoadZeroCopy;

#[non_exhaustive]
pub enum VaaAccount<'a> {
    EncodedVaa(EncodedVaa<'a>),
    PostedVaaV1(PostedVaaV1<'a>),
}

impl<'a> VaaAccount<'a> {
    pub fn version(&'a self) -> u8 {
        match self {
            Self::EncodedVaa(inner) => inner.version(),
            Self::PostedVaaV1(_) => 1,
        }
    }

    pub fn try_emitter_info(&self) -> Result<([u8; 32], u16, u64)> {
        match self {
            Self::EncodedVaa(inner) => match inner.as_vaa()? {
                VaaVersion::V1(vaa) => Ok((
                    vaa.body().emitter_address(),
                    vaa.body().emitter_chain(),
                    vaa.body().sequence(),
                )),
            },
            Self::PostedVaaV1(inner) => Ok((
                inner.emitter_address(),
                inner.emitter_chain(),
                inner.sequence(),
            )),
        }
    }

    pub fn try_payload(&self) -> Result<Payload> {
        match self {
            Self::EncodedVaa(inner) => match inner.as_vaa()? {
                VaaVersion::V1(vaa) => Ok(vaa.body().payload()),
            },
            Self::PostedVaaV1(inner) => Ok(Payload::parse(inner.payload())),
        }
    }

    pub fn encoded_vaa(&'a self) -> Option<&'a EncodedVaa<'a>> {
        match self {
            Self::EncodedVaa(inner) => Some(inner),
            _ => None,
        }
    }

    pub fn posted_vaa_v1(&'a self) -> Option<&'a PostedVaaV1<'a>> {
        match self {
            Self::PostedVaaV1(inner) => Some(inner),
            _ => None,
        }
    }
}

impl<'a> LoadZeroCopy<'a> for VaaAccount<'a> {
    fn load(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let discriminator = {
            let data = acc_info.try_borrow_data()?;
            require!(data.len() > 8, ErrorCode::AccountDidNotDeserialize);

            data[..8].try_into().unwrap()
        };

        match discriminator {
            EncodedVaa::DISC => Ok(Self::EncodedVaa(EncodedVaa::new(acc_info)?)),
            [118, 97, 97, 1, _, _, _, _] => Ok(Self::PostedVaaV1(PostedVaaV1::new(acc_info)?)),
            _ => err!(ErrorCode::AccountDidNotDeserialize),
        }
    }
}
