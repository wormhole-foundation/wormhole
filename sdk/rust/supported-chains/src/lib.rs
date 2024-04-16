//! Provide Types and Data about Wormhole's supported chains.

use std::{fmt, str::FromStr};

use serde::{Deserialize, Deserializer, Serialize, Serializer};
use thiserror::Error;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Chain {
    /// In the wormhole wire format, 0 indicates that a message is for any destination chain, it is
    /// represented here as `Any`.
    Any,

    /// Chains
    Solana,
    Ethereum,
    Terra,
    Bsc,
    Polygon,
    Avalanche,
    Oasis,
    Algorand,
    Aurora,
    Fantom,
    Karura,
    Acala,
    Klaytn,
    Celo,
    Near,
    Moonbeam,
    Neon,
    Terra2,
    Injective,
    Osmosis,
    Sui,
    Aptos,
    Arbitrum,
    Optimism,
    Gnosis,
    Pythnet,
    Xpla,
    Btc,
    Base,
    Sei,
    Rootstock,
    Scroll,
    Mantle,
    Wormchain,
    CosmosHub,
    Evmos,
    Kujira,
    Neutron,
    Celestia,
    Stargaze,
    Seda,
    Dymension,
    Provenance,
    Sepolia,

    // Allow arbitrary u16s to support future chains.
    Unknown(u16),
}

impl From<u16> for Chain {
    fn from(other: u16) -> Chain {
        match other {
            0 => Chain::Any,
            1 => Chain::Solana,
            2 => Chain::Ethereum,
            3 => Chain::Terra,
            4 => Chain::Bsc,
            5 => Chain::Polygon,
            6 => Chain::Avalanche,
            7 => Chain::Oasis,
            8 => Chain::Algorand,
            9 => Chain::Aurora,
            10 => Chain::Fantom,
            11 => Chain::Karura,
            12 => Chain::Acala,
            13 => Chain::Klaytn,
            14 => Chain::Celo,
            15 => Chain::Near,
            16 => Chain::Moonbeam,
            17 => Chain::Neon,
            18 => Chain::Terra2,
            19 => Chain::Injective,
            20 => Chain::Osmosis,
            21 => Chain::Sui,
            22 => Chain::Aptos,
            23 => Chain::Arbitrum,
            24 => Chain::Optimism,
            25 => Chain::Gnosis,
            26 => Chain::Pythnet,
            28 => Chain::Xpla,
            29 => Chain::Btc,
            30 => Chain::Base,
            32 => Chain::Sei,
            33 => Chain::Rootstock,
            34 => Chain::Scroll,
            35 => Chain::Mantle,
            3104 => Chain::Wormchain,
            4000 => Chain::CosmosHub,
            4001 => Chain::Evmos,
            4002 => Chain::Kujira,
            4003 => Chain::Neutron,
            4004 => Chain::Celestia,
            4005 => Chain::Stargaze,
            4006 => Chain::Seda,
            4007 => Chain::Dymension,
            4008 => Chain::Provenance,
            10002 => Chain::Sepolia,
            c => Chain::Unknown(c),
        }
    }
}

impl From<Chain> for u16 {
    fn from(other: Chain) -> u16 {
        match other {
            Chain::Any => 0,
            Chain::Solana => 1,
            Chain::Ethereum => 2,
            Chain::Terra => 3,
            Chain::Bsc => 4,
            Chain::Polygon => 5,
            Chain::Avalanche => 6,
            Chain::Oasis => 7,
            Chain::Algorand => 8,
            Chain::Aurora => 9,
            Chain::Fantom => 10,
            Chain::Karura => 11,
            Chain::Acala => 12,
            Chain::Klaytn => 13,
            Chain::Celo => 14,
            Chain::Near => 15,
            Chain::Moonbeam => 16,
            Chain::Neon => 17,
            Chain::Terra2 => 18,
            Chain::Injective => 19,
            Chain::Osmosis => 20,
            Chain::Sui => 21,
            Chain::Aptos => 22,
            Chain::Arbitrum => 23,
            Chain::Optimism => 24,
            Chain::Gnosis => 25,
            Chain::Pythnet => 26,
            Chain::Xpla => 28,
            Chain::Btc => 29,
            Chain::Base => 30,
            Chain::Sei => 32,
            Chain::Rootstock => 33,
            Chain::Scroll => 34,
            Chain::Mantle => 35,
            Chain::Wormchain => 3104,
            Chain::CosmosHub => 4000,
            Chain::Evmos => 4001,
            Chain::Kujira => 4002,
            Chain::Neutron => 4003,
            Chain::Celestia => 4004,
            Chain::Stargaze => 4005,
            Chain::Seda => 4006,
            Chain::Dymension => 4007,
            Chain::Provenance => 4008,
            Chain::Sepolia => 10002,
            Chain::Unknown(c) => c,
        }
    }
}

impl fmt::Display for Chain {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Any => f.write_str("Any"),
            Self::Solana => f.write_str("Solana"),
            Self::Ethereum => f.write_str("Ethereum"),
            Self::Terra => f.write_str("Terra"),
            Self::Bsc => f.write_str("Bsc"),
            Self::Polygon => f.write_str("Polygon"),
            Self::Avalanche => f.write_str("Avalanche"),
            Self::Oasis => f.write_str("Oasis"),
            Self::Algorand => f.write_str("Algorand"),
            Self::Aurora => f.write_str("Aurora"),
            Self::Fantom => f.write_str("Fantom"),
            Self::Karura => f.write_str("Karura"),
            Self::Acala => f.write_str("Acala"),
            Self::Klaytn => f.write_str("Klaytn"),
            Self::Celo => f.write_str("Celo"),
            Self::Near => f.write_str("Near"),
            Self::Moonbeam => f.write_str("Moonbeam"),
            Self::Neon => f.write_str("Neon"),
            Self::Terra2 => f.write_str("Terra2"),
            Self::Injective => f.write_str("Injective"),
            Self::Osmosis => f.write_str("Osmosis"),
            Self::Sui => f.write_str("Sui"),
            Self::Aptos => f.write_str("Aptos"),
            Self::Arbitrum => f.write_str("Arbitrum"),
            Self::Optimism => f.write_str("Optimism"),
            Self::Gnosis => f.write_str("Gnosis"),
            Self::Pythnet => f.write_str("Pythnet"),
            Self::Xpla => f.write_str("Xpla"),
            Self::Btc => f.write_str("Btc"),
            Self::Base => f.write_str("Base"),
            Self::Sei => f.write_str("Sei"),
            Self::Rootstock => f.write_str("Rootstock"),
            Self::Scroll => f.write_str("Scroll"),
            Self::Mantle => f.write_str("Mantle"),
            Self::Sepolia => f.write_str("Sepolia"),
            Self::Wormchain => f.write_str("Wormchain"),
            Self::CosmosHub => f.write_str("CosmosHub"),
            Self::Evmos => f.write_str("Evmos"),
            Self::Kujira => f.write_str("Kujira"),
            Self::Neutron => f.write_str("Neutron"),
            Self::Celestia => f.write_str("Celestia"),
            Self::Stargaze => f.write_str("Stargaze"),
            Self::Seda => f.write_str("Seda"),
            Self::Dymension => f.write_str("Dymension"),
            Self::Provenance => f.write_str("Provenance"),
            Self::Unknown(v) => write!(f, "Unknown({v})"),
        }
    }
}

#[derive(Debug, Error)]
#[error("invalid chain: {0}")]
pub struct InvalidChainError(String);

impl FromStr for Chain {
    type Err = InvalidChainError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "Any" | "any" | "ANY" => Ok(Chain::Any),
            "Solana" | "solana" | "SOLANA" => Ok(Chain::Solana),
            "Ethereum" | "ethereum" | "ETHEREUM" => Ok(Chain::Ethereum),
            "Terra" | "terra" | "TERRA" => Ok(Chain::Terra),
            "Bsc" | "bsc" | "BSC" => Ok(Chain::Bsc),
            "Polygon" | "polygon" | "POLYGON" => Ok(Chain::Polygon),
            "Avalanche" | "avalanche" | "AVALANCHE" => Ok(Chain::Avalanche),
            "Oasis" | "oasis" | "OASIS" => Ok(Chain::Oasis),
            "Algorand" | "algorand" | "ALGORAND" => Ok(Chain::Algorand),
            "Aurora" | "aurora" | "AURORA" => Ok(Chain::Aurora),
            "Fantom" | "fantom" | "FANTOM" => Ok(Chain::Fantom),
            "Karura" | "karura" | "KARURA" => Ok(Chain::Karura),
            "Acala" | "acala" | "ACALA" => Ok(Chain::Acala),
            "Klaytn" | "klaytn" | "KLAYTN" => Ok(Chain::Klaytn),
            "Celo" | "celo" | "CELO" => Ok(Chain::Celo),
            "Near" | "near" | "NEAR" => Ok(Chain::Near),
            "Moonbeam" | "moonbeam" | "MOONBEAM" => Ok(Chain::Moonbeam),
            "Neon" | "neon" | "NEON" => Ok(Chain::Neon),
            "Terra2" | "terra2" | "TERRA2" => Ok(Chain::Terra2),
            "Injective" | "injective" | "INJECTIVE" => Ok(Chain::Injective),
            "Osmosis" | "osmosis" | "OSMOSIS" => Ok(Chain::Osmosis),
            "Sui" | "sui" | "SUI" => Ok(Chain::Sui),
            "Aptos" | "aptos" | "APTOS" => Ok(Chain::Aptos),
            "Arbitrum" | "arbitrum" | "ARBITRUM" => Ok(Chain::Arbitrum),
            "Optimism" | "optimism" | "OPTIMISM" => Ok(Chain::Optimism),
            "Gnosis" | "gnosis" | "GNOSIS" => Ok(Chain::Gnosis),
            "Pythnet" | "pythnet" | "PYTHNET" => Ok(Chain::Pythnet),
            "Xpla" | "xpla" | "XPLA" => Ok(Chain::Xpla),
            "Btc" | "btc" | "BTC" => Ok(Chain::Btc),
            "Base" | "base" | "BASE" => Ok(Chain::Base),
            "Sei" | "sei" | "SEI" => Ok(Chain::Sei),
            "Rootstock" | "rootstock" | "ROOTSTOCK" => Ok(Chain::Rootstock),
            "Scroll" | "scroll" | "SCROLL" => Ok(Chain::Scroll),
            "Mantle" | "mantle" | "MANTLE" => Ok(Chain::Mantle),
            "Sepolia" | "sepolia" | "SEPOLIA" => Ok(Chain::Sepolia),
            "Wormchain" | "wormchain" | "WORMCHAIN" => Ok(Chain::Wormchain),
            "CosmosHub" | "cosmoshub" | "COSMOSHUB" => Ok(Chain::CosmosHub),
            "Evmos" | "evmos" | "EVMOS" => Ok(Chain::Evmos),
            "Kujira" | "kujira" | "KUJIRA" => Ok(Chain::Kujira),
            "Neutron" | "neutron" | "NEUTRON" => Ok(Chain::Neutron),
            "Celestia" | "celestia" | "CELESTIA" => Ok(Chain::Celestia),
            "Stargaze" | "stargaze" | "STARGAZE" => Ok(Chain::Stargaze),
            "Seda" | "seda" | "SEDA" => Ok(Chain::Seda),
            "Dymension" | "dymension" | "DYMENSION" => Ok(Chain::Dymension),
            "Provenance" | "provenance" | "PROVENANCE" => Ok(Chain::Provenance),
            _ => {
                let mut parts = s.split(&['(', ')']);
                let _ = parts
                    .next()
                    .filter(|name| name.eq_ignore_ascii_case("unknown"))
                    .ok_or_else(|| InvalidChainError(s.into()))?;

                parts
                    .next()
                    .and_then(|v| v.parse::<u16>().ok())
                    .map(Chain::from)
                    .ok_or_else(|| InvalidChainError(s.into()))
            }
        }
    }
}

impl Default for Chain {
    fn default() -> Self {
        Self::Any
    }
}

impl Serialize for Chain {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_u16((*self).into())
    }
}

impl<'de> Deserialize<'de> for Chain {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        <u16 as Deserialize>::deserialize(deserializer).map(Self::from)
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn isomorphic_from() {
        for i in 0u16..=u16::MAX {
            assert_eq!(i, u16::from(Chain::from(i)));
        }
    }

    #[test]
    fn isomorphic_display() {
        for i in 0u16..=u16::MAX {
            let c = Chain::from(i);
            assert_eq!(c, c.to_string().parse().unwrap());
        }
    }
}
