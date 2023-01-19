use std::{
    fmt,
    ops::{Deref, DerefMut},
    str::FromStr,
};

use anyhow::{anyhow, Context};
use cosmwasm_schema::cw_serde;
use cosmwasm_std::{StdError, StdResult};
use cw_storage_plus::{Key, KeyDeserialize, Prefixer, PrimaryKey};

#[cw_serde]
#[derive(Copy, Default, Eq, Hash, PartialOrd, Ord)]
#[repr(transparent)]
pub struct TokenAddress(
    #[serde(with = "hex")]
    #[schemars(with = "String")]
    [u8; 32],
);

impl TokenAddress {
    pub const fn new(addr: [u8; 32]) -> TokenAddress {
        TokenAddress(addr)
    }
}

impl From<[u8; 32]> for TokenAddress {
    fn from(addr: [u8; 32]) -> Self {
        TokenAddress(addr)
    }
}

impl From<TokenAddress> for [u8; 32] {
    fn from(addr: TokenAddress) -> Self {
        addr.0
    }
}

impl TryFrom<Vec<u8>> for TokenAddress {
    type Error = anyhow::Error;
    fn try_from(value: Vec<u8>) -> Result<Self, Self::Error> {
        <[u8; 32]>::try_from(value)
            .map(Self)
            .map_err(|v: Vec<u8>| anyhow!("invalid length; want 32, got {}", v.len()))
    }
}

impl Deref for TokenAddress {
    type Target = [u8; 32];
    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for TokenAddress {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl AsRef<[u8; 32]> for TokenAddress {
    fn as_ref(&self) -> &[u8; 32] {
        &self.0
    }
}

impl AsMut<[u8; 32]> for TokenAddress {
    fn as_mut(&mut self) -> &mut [u8; 32] {
        &mut self.0
    }
}

impl AsRef<[u8]> for TokenAddress {
    fn as_ref(&self) -> &[u8] {
        &self.0
    }
}

impl AsMut<[u8]> for TokenAddress {
    fn as_mut(&mut self) -> &mut [u8] {
        &mut self.0
    }
}

impl fmt::Display for TokenAddress {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        f.write_str(&hex::encode(self))
    }
}

impl FromStr for TokenAddress {
    type Err = anyhow::Error;
    fn from_str(s: &str) -> Result<Self, Self::Err> {
        hex::decode(s)
            .context("failed to decode hex")
            .and_then(Self::try_from)
    }
}

impl KeyDeserialize for TokenAddress {
    type Output = Self;

    fn from_vec(v: Vec<u8>) -> StdResult<Self::Output> {
        v.try_into()
            .map(TokenAddress)
            .map_err(|v| StdError::InvalidDataSize {
                expected: 32,
                actual: v.len() as u64,
            })
    }
}

impl<'a> PrimaryKey<'a> for TokenAddress {
    type Prefix = ();
    type SubPrefix = ();
    type Suffix = Self;
    type SuperSuffix = Self;

    fn key(&self) -> Vec<Key> {
        vec![Key::Ref(&**self)]
    }
}

impl<'a> Prefixer<'a> for TokenAddress {
    fn prefix(&self) -> Vec<Key> {
        vec![Key::Ref(&**self)]
    }
}
