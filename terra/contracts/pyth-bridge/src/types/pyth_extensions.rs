//! This module contains 1:1 (or close) copies of selected Pyth types
//! with quick and dirty enhancements.

use std::{
    convert::TryInto,
    io::Read,
    mem,
};

use pyth_client::{
    CorpAction,
    Ema,
    PriceStatus,
    PriceType,
};

/// 1:1 Copy of pyth_client::PriceType with derived additional traits.
#[derive(Clone, Debug, Eq, PartialEq, serde_derive::Serialize, serde_derive::Deserialize)]
#[repr(u8)]
pub enum P2WPriceType {
    Unknown,
    Price,
}

impl From<&PriceType> for P2WPriceType {
    fn from(pt: &PriceType) -> Self {
        match pt {
            PriceType::Unknown => Self::Unknown,
            PriceType::Price => Self::Price,
        }
    }
}

impl Default for P2WPriceType {
    fn default() -> Self {
        Self::Price
    }
}

/// 1:1 Copy of pyth_client::PriceStatus with derived additional traits.
#[derive(Clone, Debug, Eq, PartialEq, serde_derive::Serialize, serde_derive::Deserialize)]
pub enum P2WPriceStatus {
    Unknown,
    Trading,
    Halted,
    Auction,
}

impl From<&PriceStatus> for P2WPriceStatus {
    fn from(ps: &PriceStatus) -> Self {
        match ps {
            PriceStatus::Unknown => Self::Unknown,
            PriceStatus::Trading => Self::Trading,
            PriceStatus::Halted => Self::Halted,
            PriceStatus::Auction => Self::Auction,
        }
    }
}

impl Default for P2WPriceStatus {
    fn default() -> Self {
        Self::Trading
    }
}

/// 1:1 Copy of pyth_client::CorpAction with derived additional traits.
#[derive(Clone, Debug, Eq, PartialEq, serde_derive::Serialize, serde_derive::Deserialize)]
pub enum P2WCorpAction {
    NoCorpAct,
}

impl Default for P2WCorpAction {
    fn default() -> Self {
        Self::NoCorpAct
    }
}

impl From<&CorpAction> for P2WCorpAction {
    fn from(ca: &CorpAction) -> Self {
        match ca {
            CorpAction::NoCorpAct => P2WCorpAction::NoCorpAct,
        }
    }
}

/// 1:1 Copy of pyth_client::Ema with all-pub fields.
#[derive(
    Clone, Default, Debug, Eq, PartialEq, serde_derive::Serialize, serde_derive::Deserialize,
)]
#[repr(C)]
pub struct P2WEma {
    pub val: i64,
    pub numer: i64,
    pub denom: i64,
}

/// CAUTION: This impl may panic and requires an unsafe cast
impl From<&Ema> for P2WEma {
    fn from(ema: &Ema) -> Self {
        let our_size = mem::size_of::<P2WEma>();
        let upstream_size = mem::size_of::<Ema>();
        if our_size == upstream_size {
            unsafe { std::mem::transmute_copy(ema) }
        } else {
            dbg!(our_size);
            dbg!(upstream_size);
            // Because of private upstream fields it's impossible to
            // complain about type-level changes at compile-time
            panic!("P2WEma sizeof mismatch")
        }
    }
}

/// CAUTION: This impl may panic and requires an unsafe cast
impl Into<Ema> for &P2WEma {
    fn into(self) -> Ema {
        let our_size = mem::size_of::<P2WEma>();
        let upstream_size = mem::size_of::<Ema>();
        if our_size == upstream_size {
            unsafe { std::mem::transmute_copy(self) }
        } else {
            dbg!(our_size);
            dbg!(upstream_size);
            // Because of private upstream fields it's impossible to
            // complain about type-level changes at compile-time
            panic!("P2WEma sizeof mismatch")
        }
    }
}

impl P2WEma {
    pub fn serialize(&self) -> Vec<u8> {
        let mut v = vec![];
        // val
        v.extend(&self.val.to_be_bytes()[..]);

        // numer
        v.extend(&self.numer.to_be_bytes()[..]);

        // denom
        v.extend(&self.denom.to_be_bytes()[..]);

        v
    }

    pub fn deserialize(mut bytes: impl Read) -> Result<Self, Box<dyn std::error::Error>> {
        let mut val_vec = vec![0u8; mem::size_of::<i64>()];
        bytes.read_exact(val_vec.as_mut_slice())?;
        let val = i64::from_be_bytes(val_vec.as_slice().try_into()?);

        let mut numer_vec = vec![0u8; mem::size_of::<i64>()];
        bytes.read_exact(numer_vec.as_mut_slice())?;
        let numer = i64::from_be_bytes(numer_vec.as_slice().try_into()?);

        let mut denom_vec = vec![0u8; mem::size_of::<i64>()];
        bytes.read_exact(denom_vec.as_mut_slice())?;
        let denom = i64::from_be_bytes(denom_vec.as_slice().try_into()?);

        Ok(Self { val, numer, denom })
    }
}
