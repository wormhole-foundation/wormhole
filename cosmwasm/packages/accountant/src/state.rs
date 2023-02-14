use std::fmt;

use cosmwasm_schema::cw_serde;
use cosmwasm_std::Uint256;
use cw_storage_plus::Map;

pub mod account;
mod addr;
pub mod transfer;

pub use account::Account;
pub use addr::TokenAddress;
pub use transfer::Transfer;

pub const ACCOUNTS: Map<account::Key, account::Balance> = Map::new("accountant/accounts");

pub const TRANSFERS: Map<transfer::Key, transfer::Data> = Map::new("accountant/transfers");

#[cw_serde]
#[derive(Eq, PartialOrd, Ord)]
pub enum Kind {
    Add = 1,
    Sub = 2,
}

impl fmt::Display for Kind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Kind::Add => f.write_str("ADD"),
            Kind::Sub => f.write_str("SUB"),
        }
    }
}

#[cw_serde]
#[derive(Copy, Default, Eq, Hash, PartialOrd, Ord)]
#[repr(transparent)]
pub struct Reason(
    #[serde(with = "fixed_bytes_as_string_serde")]
    #[schemars(with = "String")]
    pub [u8; 32],
);

impl Reason {
    pub const fn new(reason: [u8; 32]) -> Reason {
        Reason(reason)
    }
    // pub fn to_string(&self) -> String {
    //     unsafe { std::str::from_utf8_unchecked(&self.0).to_string() }
    // }
}

impl From<[u8; 32]> for Reason {
    fn from(reason: [u8; 32]) -> Self {
        Reason(reason)
    }
}

impl TryFrom<&str> for Reason {
    type Error = &'static str;
    fn try_from(reason: &str) -> Result<Self, Self::Error> {
        // default all spaces
        let mut bz = [b' '; 32];
        if reason.len() <= 32 {
            for (i, byte) in reason.as_bytes().iter().enumerate() {
                bz[i] = *byte;
            }
            Ok(Reason(bz))
        } else {
            Err("reason cannot be longer than 32 bytes")
        }
    }
}

#[cw_serde]
#[derive(Eq, PartialOrd, Ord)]
pub struct Modification {
    // The sequence number of this modification.  Each modification must be
    // uniquely identifiable just by its sequnce number.
    pub sequence: u64,
    // The chain id of the account to be modified.
    pub chain_id: u16,
    // The chain id of the native chain for the token.
    pub token_chain: u16,
    // The address of the token on its native chain.
    pub token_address: TokenAddress,
    // The kind of modification to be made.
    pub kind: Kind,
    // The amount to be modified.
    pub amount: Uint256,
    // A human-readable reason for the modification.
    pub reason: Reason,
}

pub const MODIFICATIONS: Map<u64, Modification> = Map::new("accountant/modifications");

pub mod fixed_bytes_as_string_serde {
    use serde::{Deserialize, Serialize};
    use serde::{Deserializer, Serializer};

    pub fn serialize<S: Serializer>(v: &[u8; 32], s: S) -> Result<S::Ok, S::Error> {
        let str = String::from_utf8((*v).into())
            .map_err(|_| serde::ser::Error::custom("invalid utf8"))?;
        String::serialize(&str, s)
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<[u8; 32], D::Error> {
        static EXPECTING: &str = "reason string of 32 bytes";
        let str = String::deserialize(d)?;
        let bz = str.as_bytes();
        if bz.len() != 32 {
            return Err(serde::de::Error::invalid_length(32, &EXPECTING));
        }
        let mut bz32 = [0u8; 32];
        bz32.clone_from_slice(bz);
        Ok(bz32)
    }
}
