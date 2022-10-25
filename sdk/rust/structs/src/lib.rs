use std::{fmt, str::FromStr};

use serde::{Deserialize, Serialize};
use serde_repr::{Deserialize_repr, Serialize_repr};

mod arraystring;
mod serde_array;
pub mod tokenbridge;
pub mod wormhole;

#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Address(pub [u8; 32]);

#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Amount(pub [u8; 32]);

#[derive(
    Serialize_repr, Deserialize_repr, Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash,
)]
#[repr(u16)]
pub enum Chain {
    Unset = 0,
    Solana = 1,
    Ethereum = 2,
    Terra = 3,
    Bsc = 4,
    Polygon = 5,
    Avalanche = 6,
    Oasis = 7,
    Algorand = 8,
    Aurora = 9,
    Fantom = 10,
    Karura = 11,
    Acala = 12,
    Klaytn = 13,
    Celo = 14,
    Near = 15,
    Moonbeam = 16,
    Neon = 17,
    Terra2 = 18,
    Injective = 19,
    Osmosis = 20,
    Sui = 21,
    Aptos = 22,
    Arbitrum = 23,
    Optimism = 24,
    Gnosis = 25,
    Pythnet = 26,
    Xpla = 28,
    Ropsten = 10001,
    Wormchain = 3104,
    #[serde(other)]
    Unknown,
}

impl fmt::Display for Chain {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Unset => f.write_str("Unset"),
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
            Self::Unknown => f.write_str("Unknown"),
        }
    }
}

impl FromStr for Chain {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "Unset" | "unset" | "UNSET" => Ok(Chain::Unset),
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
            "Unknown" | "unknown" | "UNKNOWN" => Ok(Chain::Unknown),
            _ => Err(format!("invalid chain: {s}")),
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GuardianAddress(pub [u8; 20]);

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct GuardianSetInfo {
    pub addresses: Vec<GuardianAddress>,
    #[serde(skip)]
    pub expiration_time: u64,
}

impl GuardianSetInfo {
    pub fn quorum(&self) -> usize {
        // allow quorum of 0 for testing purposes...
        if self.addresses.is_empty() {
            0
        } else {
            ((self.addresses.len() * 10 / 3) * 2) / 10 + 1
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Signature {
    pub index: u8,
    #[serde(with = "crate::serde_array")]
    pub signature: [u8; 65],
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Vaa<P> {
    pub version: u8,
    pub guardian_set_index: u32,
    pub signatures: Vec<Signature>,
    pub timestamp: u32, // Seconds since UNIX epoch
    pub nonce: u32,
    pub emitter_chain: Chain,
    pub emitter_address: Address,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: P,
}

