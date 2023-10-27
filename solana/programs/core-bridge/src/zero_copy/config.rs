use std::cell::Ref;

use crate::state;
use anchor_lang::{
    prelude::{error, require_eq, require_keys_eq, AccountInfo, ErrorCode, Result},
    Space,
};

/// Account used to store the current configuration of the bridge, including tracking Wormhole fee
/// payments. For governance decrees, the guardian set index is used to determine whether a decree
/// was attested for using the latest guardian set.
pub struct Config<'a>(Ref<'a, &'a mut [u8]>);

impl<'a> Config<'a> {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_le_bytes(self.0[..4].try_into().unwrap())
    }

    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub fn guardian_set_ttl(&self) -> crate::types::Duration {
        u32::from_le_bytes(self.0[12..16].try_into().unwrap()).into()
    }

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fn fee_lamports(&self) -> u64 {
        u64::from_le_bytes(self.0[16..24].try_into().unwrap())
    }

    pub fn parse(acc_info: &'a AccountInfo) -> Result<Self> {
        require_keys_eq!(*acc_info.owner, crate::ID, ErrorCode::ConstraintOwner);

        let data = acc_info.try_borrow_data()?;
        require_eq!(
            data.len(),
            state::Config::INIT_SPACE,
            ErrorCode::AccountDidNotDeserialize
        );
        Ok(Self(data))
    }
}
