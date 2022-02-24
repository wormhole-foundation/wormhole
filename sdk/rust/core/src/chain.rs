//! Exposes an API implementation depending on which feature flags have been toggled for the
//! library. Check submodules for chain runtime specific documentation.
use std::convert::TryFrom; // Remove in 2021


/// Chain contains a mapping of Wormhole supported chains to their u16 representation. These are
/// universally defined among all Wormhole contracts.
#[repr(u16)]
#[derive(Clone, Debug, PartialEq)]
pub enum Chain {
    All      = 0,
    Solana   = 1,
    Ethereum = 2,
    Terra    = 3,
    Binance  = 4,
    Polygon  = 5,
    AVAX     = 6,
    Oasis    = 7,
}

impl TryFrom<u16> for Chain {
    type Error = ();
    fn try_from(other: u16) -> Result<Chain, Self::Error> {
        match other {
            0 => Ok(Chain::All),
            1 => Ok(Chain::Solana),
            2 => Ok(Chain::Ethereum),
            3 => Ok(Chain::Terra),
            4 => Ok(Chain::Binance),
            5 => Ok(Chain::Polygon),
            6 => Ok(Chain::AVAX),
            7 => Ok(Chain::Oasis),
            _ => Err(()),
        }
    }
}

impl Default for Chain {
    fn default() -> Self {
        Self::All
    }
}
