//! Provide Types and Data about Wormhole's supported chains.

use Chain::*;

/// Chain is a mapping of Wormhole supported chains to their u16 representation.
#[cfg_attr(test, derive(strum_macros::EnumIter))]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Chain {
    // In Wormhole payload format, 0 indicates that a message is for any destination chain.
    Any,

    // Chains
    Acala,
    Algorand,
    Aptos,
    Arbitrum,
    Aurora,
    Avalanche,
    Binance,
    Celo,
    Ethereum,
    Fantom,
    Gnosis,
    Injective,
    Karura,
    Klaytn,
    Moonbeam,
    Near,
    Neon,
    Oasis,
    Optimism,
    Osmosis,
    Polygon,
    Pythnet,
    Solana,
    Sui,
    Terra,
    TerraClassic,
    WormholeChain,

    // Testnet Chains
    Ropsten,

    // Allow parsing an arbitrary u16 to support future chains.
    Unknown(u16),
}

impl From<u16> for Chain {
    fn from(other: u16) -> Chain {
        match other {
            0 => Any,
            1 => Solana,
            2 => Ethereum,
            3 => TerraClassic,
            4 => Binance,
            5 => Polygon,
            6 => Avalanche,
            7 => Oasis,
            8 => Algorand,
            9 => Aurora,
            10 => Fantom,
            11 => Karura,
            12 => Acala,
            13 => Klaytn,
            14 => Celo,
            15 => Near,
            16 => Moonbeam,
            17 => Neon,
            18 => Terra,
            19 => Injective,
            20 => Osmosis,
            21 => Sui,
            22 => Aptos,
            23 => Arbitrum,
            24 => Optimism,
            25 => Gnosis,
            26 => Pythnet,
            3104 => WormholeChain,
            10001 => Ropsten,
            _ => Unknown(other),
        }
    }
}

impl From<Chain> for u16 {
    fn from(other: Chain) -> u16 {
        match other {
            Any => 0,
            Solana => 1,
            Ethereum => 2,
            TerraClassic => 3,
            Binance => 4,
            Polygon => 5,
            Avalanche => 6,
            Oasis => 7,
            Algorand => 8,
            Aurora => 9,
            Fantom => 10,
            Karura => 11,
            Acala => 12,
            Klaytn => 13,
            Celo => 14,
            Near => 15,
            Moonbeam => 16,
            Neon => 17,
            Terra => 18,
            Injective => 19,
            Osmosis => 20,
            Sui => 21,
            Aptos => 22,
            Arbitrum => 23,
            Optimism => 24,
            Gnosis => 25,
            Pythnet => 26,
            WormholeChain => 3104,
            Ropsten => 10001,
            Unknown(other) => other,
        }
    }
}

impl Default for Chain {
    fn default() -> Self {
        Self::Any
    }
}

#[cfg(test)]
mod testing {
    use {
        super::Chain,
        strum::IntoEnumIterator,
    };

    #[test]
    fn check_reverse_mapping() {
        for chain in Chain::iter() {
            // IntoIter defaults to 0 for the Unknown variant, so we skip it to avoid the error.
            // There's no normal way to construct this value so It's not a problem.
            if matches!(chain, Chain::Unknown(_)) {
                continue;
            }

            // Check: Solana -> 1 -> Solana, etc.
            assert_eq!(chain, Chain::from(u16::from(chain)));
        }
    }
}
