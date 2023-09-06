use crate::state;
use anchor_lang::{
    prelude::{error, require_eq, ErrorCode, Result},
    Space,
};

/// Account used to store the current configuration of the bridge, including tracking Wormhole fee
/// payments. For governance decrees, the guardian set index is used to determine whether a decree
/// was attested for using the latest guardian set.
pub struct Config<'a>(&'a [u8]);

impl<'a> Config<'a> {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub fn guardian_set_index(&self) -> u32 {
        u32::from_le_bytes(self.0[..4].try_into().unwrap())
    }

    /// Lamports in the collection account
    pub fn last_lamports(&self) -> u64 {
        u64::from_le_bytes(self.0[4..12].try_into().unwrap())
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

    pub fn parse(span: &'a [u8]) -> Result<Self> {
        require_eq!(
            span.len(),
            state::Config::INIT_SPACE,
            ErrorCode::AccountDidNotDeserialize
        );
        Ok(Self(span))
    }
}
