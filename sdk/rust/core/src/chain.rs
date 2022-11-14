//! Provide Types and Data about Wormhole's supported chains.

use std::{fmt, str::FromStr};

use serde::{Deserialize, Deserializer, Serialize, Serializer};

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
    Ropsten,
    Wormchain,

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
            3104 => Chain::Wormchain,
            10001 => Chain::Ropsten,
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
            Chain::Wormchain => 3104,
            Chain::Ropsten => 10001,
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
            Self::Ropsten => f.write_str("Ropsten"),
            Self::Wormchain => f.write_str("Wormchain"),
            Self::Unknown(v) => write!(f, "Unknown({v})"),
        }
    }
}

impl FromStr for Chain {
    type Err = String;

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
            "Ropsten" | "ropsten" | "ROPSTEN" => Ok(Chain::Ropsten),
            "Wormchain" | "wormchain" | "WORMCHAIN" => Ok(Chain::Wormchain),
            _ => Err(format!("invalid chain: {s}")),
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
