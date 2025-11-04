//! Provide Types and Data about Wormhole's supported chains.

use std::{fmt, str::FromStr};

use serde::{Deserialize, Deserializer, Serialize, Serializer};
use thiserror::Error;

#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Chain {
    /// In the wormhole wire format, 0 indicates that a message is for any destination chain, it is
    /// represented here as `Any`.
    #[default]
    Any,

    /// Chains
    Solana,
    Ethereum,
    /// WARNING: Terra is only supported in devnet / Tilt.
    Terra,
    Bsc,
    Polygon,
    Avalanche,
    // OBSOLETE: Oasis was ID 7
    Algorand,
    // OBSOLETE: Aurora was ID 9
    Fantom,
    // OBSOLETE: Karura was ID 11
    // OBSOLETE: Acala was ID 12
    Klaytn,
    Celo,
    Near,
    Moonbeam,
    // OBSOLETE: Neon was ID 17
    /// WARNING: Terra2 is only supported in devnet / Tilt.
    Terra2,
    Injective,
    Osmosis,
    Sui,
    Aptos,
    Arbitrum,
    Optimism,
    Gnosis,
    Pythnet,
    // NOTE: 27 belongs to a chain that was never deployed.
    // OBSOLETE: Xpla was ID 28
    Btc,
    Base,
    FileCoin,
    Sei,
    Rootstock,
    Scroll,
    Mantle,
    // OBSOLETE: Blast was ID 36
    XLayer,
    Linea,
    Berachain,
    SeiEVM,
    Eclipse,
    BOB,
    // OBSOLETE: Snaxchain was ID 43
    Unichain,
    Worldchain,
    Ink,
    HyperEVM,
    Monad,
    Movement,
    Mezo,
    Fogo,
    Sonic,
    Converge,
    Codex,
    Plume,
    Aztec,
    XRPLEVM,
    Plasma,
    CreditCoin,
    Stacks,
    Stellar,
    TON,
    Moca,
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
    Noble,
    Sepolia,
    ArbitrumSepolia,
    BaseSepolia,
    OptimismSepolia,
    Holesky,
    PolygonSepolia,
    // OBSOLETE: MonadDevnet was ID 10008

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
            8 => Chain::Algorand,
            10 => Chain::Fantom,
            13 => Chain::Klaytn,
            14 => Chain::Celo,
            15 => Chain::Near,
            16 => Chain::Moonbeam,
            18 => Chain::Terra2,
            19 => Chain::Injective,
            20 => Chain::Osmosis,
            21 => Chain::Sui,
            22 => Chain::Aptos,
            23 => Chain::Arbitrum,
            24 => Chain::Optimism,
            25 => Chain::Gnosis,
            26 => Chain::Pythnet,
            29 => Chain::Btc,
            30 => Chain::Base,
            31 => Chain::FileCoin,
            32 => Chain::Sei,
            33 => Chain::Rootstock,
            34 => Chain::Scroll,
            35 => Chain::Mantle,
            37 => Chain::XLayer,
            38 => Chain::Linea,
            39 => Chain::Berachain,
            40 => Chain::SeiEVM,
            41 => Chain::Eclipse,
            42 => Chain::BOB,
            44 => Chain::Unichain,
            45 => Chain::Worldchain,
            46 => Chain::Ink,
            47 => Chain::HyperEVM,
            48 => Chain::Monad,
            49 => Chain::Movement,
            50 => Chain::Mezo,
            51 => Chain::Fogo,
            52 => Chain::Sonic,
            53 => Chain::Converge,
            54 => Chain::Codex,
            55 => Chain::Plume,
            56 => Chain::Aztec,
            57 => Chain::XRPLEVM,
            58 => Chain::Plasma,
            59 => Chain::CreditCoin,
            60 => Chain::Stacks,
            61 => Chain::Stellar,
            62 => Chain::TON,
            63 => Chain::Moca,
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
            4009 => Chain::Noble,
            10002 => Chain::Sepolia,
            10003 => Chain::ArbitrumSepolia,
            10004 => Chain::BaseSepolia,
            10005 => Chain::OptimismSepolia,
            10006 => Chain::Holesky,
            10007 => Chain::PolygonSepolia,
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
            Chain::Algorand => 8,
            Chain::Fantom => 10,
            Chain::Klaytn => 13,
            Chain::Celo => 14,
            Chain::Near => 15,
            Chain::Moonbeam => 16,
            Chain::Terra2 => 18,
            Chain::Injective => 19,
            Chain::Osmosis => 20,
            Chain::Sui => 21,
            Chain::Aptos => 22,
            Chain::Arbitrum => 23,
            Chain::Optimism => 24,
            Chain::Gnosis => 25,
            Chain::Pythnet => 26,
            Chain::Btc => 29,
            Chain::Base => 30,
            Chain::FileCoin => 31,
            Chain::Sei => 32,
            Chain::Rootstock => 33,
            Chain::Scroll => 34,
            Chain::Mantle => 35,
            Chain::XLayer => 37,
            Chain::Linea => 38,
            Chain::Berachain => 39,
            Chain::SeiEVM => 40,
            Chain::Eclipse => 41,
            Chain::BOB => 42,
            Chain::Unichain => 44,
            Chain::Worldchain => 45,
            Chain::Ink => 46,
            Chain::HyperEVM => 47,
            Chain::Monad => 48,
            Chain::Movement => 49,
            Chain::Mezo => 50,
            Chain::Fogo => 51,
            Chain::Sonic => 52,
            Chain::Converge => 53,
            Chain::Codex => 54,
            Chain::Plume => 55,
            Chain::Aztec => 56,
            Chain::XRPLEVM => 57,
            Chain::Plasma => 58,
            Chain::CreditCoin => 59,
            Chain::Stacks => 60,
            Chain::Stellar => 61,
            Chain::TON => 62,
            Chain::Moca => 63,
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
            Chain::Noble => 4009,
            Chain::Sepolia => 10002,
            Chain::ArbitrumSepolia => 10003,
            Chain::BaseSepolia => 10004,
            Chain::OptimismSepolia => 10005,
            Chain::Holesky => 10006,
            Chain::PolygonSepolia => 10007,
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
            Self::Algorand => f.write_str("Algorand"),
            Self::Fantom => f.write_str("Fantom"),
            Self::Klaytn => f.write_str("Klaytn"),
            Self::Celo => f.write_str("Celo"),
            Self::Near => f.write_str("Near"),
            Self::Moonbeam => f.write_str("Moonbeam"),
            Self::Terra2 => f.write_str("Terra2"),
            Self::Injective => f.write_str("Injective"),
            Self::Osmosis => f.write_str("Osmosis"),
            Self::Sui => f.write_str("Sui"),
            Self::Aptos => f.write_str("Aptos"),
            Self::Arbitrum => f.write_str("Arbitrum"),
            Self::Optimism => f.write_str("Optimism"),
            Self::Gnosis => f.write_str("Gnosis"),
            Self::Pythnet => f.write_str("Pythnet"),
            Self::Btc => f.write_str("Btc"),
            Self::Base => f.write_str("Base"),
            Self::FileCoin => f.write_str("FileCoin"),
            Self::Sei => f.write_str("Sei"),
            Self::Rootstock => f.write_str("Rootstock"),
            Self::Scroll => f.write_str("Scroll"),
            Self::Mantle => f.write_str("Mantle"),
            Self::XLayer => f.write_str("XLayer"),
            Self::Linea => f.write_str("Linea"),
            Self::Berachain => f.write_str("Berachain"),
            Self::SeiEVM => f.write_str("SeiEVM"),
            Self::Eclipse => f.write_str("Eclipse"),
            Self::BOB => f.write_str("BOB"),
            Self::Unichain => f.write_str("Unichain"),
            Self::Worldchain => f.write_str("Worldchain"),
            Self::Ink => f.write_str("Ink"),
            Self::HyperEVM => f.write_str("HyperEVM"),
            Self::Monad => f.write_str("Monad"),
            Self::Movement => f.write_str("Movement"),
            Self::Mezo => f.write_str("Mezo"),
            Self::Fogo => f.write_str("Fogo"),
            Self::Sonic => f.write_str("Sonic"),
            Self::Converge => f.write_str("Converge"),
            Self::Codex => f.write_str("Codex"),
            Self::Plume => f.write_str("Plume"),
            Self::Aztec => f.write_str("Aztec"),
            Self::XRPLEVM => f.write_str("XRPLEVM"),
            Self::Plasma => f.write_str("Plasma"),
            Self::CreditCoin => f.write_str("CreditCoin"),
            Self::Stacks => f.write_str("Stacks"),
            Self::Stellar => f.write_str("Stellar"),
            Self::TON => f.write_str("TON"),
            Self::Moca => f.write_str("Moca"),
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
            Self::Noble => f.write_str("Noble"),
            Self::Sepolia => f.write_str("Sepolia"),
            Self::ArbitrumSepolia => f.write_str("ArbitrumSepolia"),
            Self::BaseSepolia => f.write_str("BaseSepolia"),
            Self::OptimismSepolia => f.write_str("OptimismSepolia"),
            Self::Holesky => f.write_str("Holesky"),
            Self::PolygonSepolia => f.write_str("PolygonSepolia"),
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
            "Algorand" | "algorand" | "ALGORAND" => Ok(Chain::Algorand),
            "Fantom" | "fantom" | "FANTOM" => Ok(Chain::Fantom),
            "Klaytn" | "klaytn" | "KLAYTN" => Ok(Chain::Klaytn),
            "Celo" | "celo" | "CELO" => Ok(Chain::Celo),
            "Near" | "near" | "NEAR" => Ok(Chain::Near),
            "Moonbeam" | "moonbeam" | "MOONBEAM" => Ok(Chain::Moonbeam),
            "Terra2" | "terra2" | "TERRA2" => Ok(Chain::Terra2),
            "Injective" | "injective" | "INJECTIVE" => Ok(Chain::Injective),
            "Osmosis" | "osmosis" | "OSMOSIS" => Ok(Chain::Osmosis),
            "Sui" | "sui" | "SUI" => Ok(Chain::Sui),
            "Aptos" | "aptos" | "APTOS" => Ok(Chain::Aptos),
            "Arbitrum" | "arbitrum" | "ARBITRUM" => Ok(Chain::Arbitrum),
            "Optimism" | "optimism" | "OPTIMISM" => Ok(Chain::Optimism),
            "Gnosis" | "gnosis" | "GNOSIS" => Ok(Chain::Gnosis),
            "Pythnet" | "pythnet" | "PYTHNET" => Ok(Chain::Pythnet),
            "Btc" | "btc" | "BTC" => Ok(Chain::Btc),
            "Base" | "base" | "BASE" => Ok(Chain::Base),
            "FileCoin" | "filecoin" | "FILECOIN" => Ok(Chain::FileCoin),
            "Sei" | "sei" | "SEI" => Ok(Chain::Sei),
            "Rootstock" | "rootstock" | "ROOTSTOCK" => Ok(Chain::Rootstock),
            "Scroll" | "scroll" | "SCROLL" => Ok(Chain::Scroll),
            "Mantle" | "mantle" | "MANTLE" => Ok(Chain::Mantle),
            "XLayer" | "xlayer" | "XLAYER" => Ok(Chain::XLayer),
            "Linea" | "linea" | "LINEA" => Ok(Chain::Linea),
            "Berachain" | "berachain" | "BERACHAIN" => Ok(Chain::Berachain),
            "SeiEVM" | "seievm" | "SEIEVM" => Ok(Chain::SeiEVM),
            "Eclipse" | "eclipse" | "ECLIPSE" => Ok(Chain::Eclipse),
            "BOB" | "bob" => Ok(Chain::BOB),
            "Unichain" | "unichain" | "UNICHAIN" => Ok(Chain::Unichain),
            "Worldchain" | "worldchain" | "WORLDCHAIN" => Ok(Chain::Worldchain),
            "Ink" | "ink" | "INK" => Ok(Chain::Ink),
            "HyperEVM" | "hyperevm" | "HYPEREVM" => Ok(Chain::HyperEVM),
            "Monad" | "monad" | "MONAD" => Ok(Chain::Monad),
            "Movement" | "movement" | "MOVEMENT" => Ok(Chain::Movement),
            "Mezo" | "mezo" | "MEZO" => Ok(Chain::Mezo),
            "Fogo" | "fogo" | "FOGO" => Ok(Chain::Fogo),
            "Sonic" | "sonic" | "SONIC" => Ok(Chain::Sonic),
            "Converge" | "converge" | "CONVERGE" => Ok(Chain::Converge),
            "Codex" | "codex" | "CODEX" => Ok(Chain::Codex),
            "Plume" | "plume" | "PLUME" => Ok(Chain::Plume),
            "Aztec" | "aztec" | "AZTEC" => Ok(Chain::Aztec),
            "XRPLEVM" | "xrplevm" => Ok(Chain::XRPLEVM),
            "Plasma" | "plasma" | "PLASMA" => Ok(Chain::Plasma),
            "CreditCoin" | "creditcoin" | "CREDITCOIN" => Ok(Chain::CreditCoin),
            "Stacks" | "stacks" | "STACKS" => Ok(Chain::Stacks),
            "Stellar" | "stellar" | "STELLAR" => Ok(Chain::Stellar),
            "TON" | "ton" => Ok(Chain::TON),
            "Moca" | "moca" | "MOCA" => Ok(Chain::Moca),
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
            "Noble" | "noble" | "NOBLE" => Ok(Chain::Noble),
            "Sepolia" | "sepolia" | "SEPOLIA" => Ok(Chain::Sepolia),
            "ArbitrumSepolia" | "arbitrumsepolia" | "ARBITRUMSEPOLIA" => Ok(Chain::ArbitrumSepolia),
            "BaseSepolia" | "basesepolia" | "BASESEPOLIA" => Ok(Chain::BaseSepolia),
            "OptimismSepolia" | "optimismsepolia" | "OPTIMISMSEPOLIA" => Ok(Chain::OptimismSepolia),
            "Holesky" | "holesky" | "HOLESKY" => Ok(Chain::Holesky),
            "PolygonSepolia" | "polygonsepolia" | "POLYGONSEPOLIA" => Ok(Chain::PolygonSepolia),
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
