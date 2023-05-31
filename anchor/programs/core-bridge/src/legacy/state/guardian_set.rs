use std::{fmt, io};

use crate::{
    message::{WormDecode, WormEncode},
    types::Timestamp,
};
use anchor_lang::prelude::*;
use wormhole_common::{legacy_account, LegacyDiscriminator, NewAccountSize, SeedPrefix};

#[derive(Debug, Copy, Clone, PartialEq, Eq, AnchorSerialize, AnchorDeserialize)]
pub struct Guardian {
    key: [u8; 20],
}

impl From<[u8; 20]> for Guardian {
    fn from(key: [u8; 20]) -> Self {
        Guardian { key }
    }
}

impl From<&[u8; 20]> for Guardian {
    fn from(key: &[u8; 20]) -> Self {
        Guardian { key: *key }
    }
}

impl AsRef<[u8; 20]> for Guardian {
    fn as_ref(&self) -> &[u8; 20] {
        &self.key
    }
}

impl WormDecode for Guardian {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let mut key = [0; 20];
        reader.read_exact(&mut key)?;
        Ok(Guardian { key })
    }
}

impl WormEncode for Guardian {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        writer.write_all(&self.key)
    }
}

impl fmt::Display for Guardian {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "0x{}", hex::encode(self.key))
    }
}

#[legacy_account]
#[derive(Debug, PartialEq, Eq)]
pub struct GuardianSet {
    /// Index representing an incrementing version number for this guardian set.
    pub index: u32,

    /// Ethereum-style public keys.
    pub keys: Vec<Guardian>,

    /// Timestamp representing the time this guardian became active.
    pub creation_time: Timestamp,

    /// Expiration time when VAAs issued by this set are no longer valid.
    pub expiration_time: Timestamp,
}

impl GuardianSet {
    pub fn is_active(&self, timestamp: &Timestamp) -> bool {
        // Note: This is a fix for Wormhole on mainnet.  The initial guardian set was never expired
        // so we block it here.
        if self.index == 0 && self.creation_time == 1628099186.into() {
            false
        } else {
            self.expiration_time == Default::default() || self.expiration_time >= *timestamp
        }
    }
}

impl LegacyDiscriminator<0> for GuardianSet {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}

impl SeedPrefix for GuardianSet {
    #[inline]
    fn seed_prefix() -> &'static [u8] {
        b"GuardianSet"
    }
}

impl NewAccountSize for GuardianSet {
    fn compute_size(num_guardians: usize) -> usize {
        4 // index
        + 4 + num_guardians * 20 // keys
        + Timestamp::INIT_SPACE // creation_time
        + Timestamp::INIT_SPACE // expiration_time
    }
}

#[cfg(test)]
pub mod guardian_set_test {
    // use crate::{
    //     test_utils::{AccountInfoCreator, DefaultAccountInfo},
    //     GuardianSet,
    // };

    // impl DefaultAccountInfo for AccountInfoCreator<GuardianSet> {
    //     fn default_info(&mut self) -> solana_program::account_info::AccountInfo {
    //         self.make_info(GuardianSet {
    //             index: 1,
    //             creation_time: 2,
    //             expiration_time: 3,
    //         })
    //     }
    // }
}
