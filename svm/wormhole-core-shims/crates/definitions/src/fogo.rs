/// Wormhole Chain ID identifying Fogo's network.
pub const CHAIN_ID: u16 = 51;

pub mod mainnet {
    use const_crypto::bs58;

    /// Core Bridge program ID on Fogo mainnet.
    pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("worm2mrQkG1B1KTz37erMfWN8anHkSK24nzca7UD8BB");

    crate::derive_core_consts!();
}

pub mod testnet {
    use const_crypto::bs58;

    /// Core Bridge program ID on Fogo testnet.
    pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("BhnQyKoQQgpuRTRo6D8Emz93PvXCYfVgHhnrR4T3qhw4");

    crate::derive_core_consts!();
}

cfg_if::cfg_if! {
    if #[cfg(feature = "testnet")] {
        pub use testnet::*;
    }
    // Default to mainnet.
    else {
        pub use mainnet::*;
    }
}
