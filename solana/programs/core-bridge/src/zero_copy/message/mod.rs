mod posted_message_v1;
pub use posted_message_v1::*;

use anchor_lang::prelude::{err, error, require, require_keys_eq, AccountInfo, ErrorCode, Result};

use super::LoadZeroCopy;

#[non_exhaustive]
pub enum MessageAccount<'a> {
    V1(PostedMessageV1<'a>),
    UnreliableV1(PostedMessageV1<'a>),
}

impl<'a> MessageAccount<'a> {
    pub fn v1(&'a self) -> Option<&'a PostedMessageV1<'a>> {
        match self {
            Self::V1(inner) => Some(inner),
            _ => None,
        }
    }

    pub fn v1_unreliable(&'a self) -> Option<&'a PostedMessageV1<'a>> {
        match self {
            Self::UnreliableV1(inner) => Some(inner),
            _ => None,
        }
    }
}

impl<'a> LoadZeroCopy<'a> for MessageAccount<'a> {
    fn load(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let discriminator = {
            let data = acc_info.try_borrow_data()?;
            require!(data.len() > 4, ErrorCode::AccountDidNotDeserialize);

            data[..4].try_into().unwrap()
        };

        match discriminator {
            PostedMessageV1::DISC => Ok(Self::V1(PostedMessageV1::new(acc_info)?)),
            PostedMessageV1::UNRELIABLE_DISC => {
                Ok(Self::UnreliableV1(PostedMessageV1::new(acc_info)?))
            }
            _ => err!(ErrorCode::AccountDidNotDeserialize),
        }
    }
}
