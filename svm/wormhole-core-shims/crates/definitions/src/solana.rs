/// Wormhole Chain ID identifying Solana's network. This ID is shared between Solana mainnet and
/// devnet.
pub const CHAIN_ID: u16 = 1;

pub mod mainnet {
    use const_crypto::bs58;

    /// Core Bridge program ID on Solana mainnet.
    pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth");

    crate::derive_core_consts!();

    /// Post message shim program ID on Solana mainnet.
    pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

    crate::derive_post_message_shim_consts!();

    /// Verify VAA shim program ID on Solana mainnet.
    pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

    crate::derive_verify_vaa_shim_consts!();
}

pub mod devnet {
    use const_crypto::bs58;

    /// Core Bridge program ID on Solana devnet.
    pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5");

    crate::derive_core_consts!();

    /// Post message shim program ID on Solana devnet.
    pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

    crate::derive_post_message_shim_consts!();

    /// Verify VAA shim program ID on Solana devnet.
    pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

    crate::derive_verify_vaa_shim_consts!();
}

pub mod localnet {
    use const_crypto::bs58;

    /// Core Bridge program ID on Wormhole's Tilt (dev) network.
    pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");

    crate::derive_core_consts!();

    /// Post message shim program ID on Wormhole's Tilt (dev) network.
    pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

    crate::derive_post_message_shim_consts!();

    /// Verify VAA shim program ID on Wormhole's Tilt (dev) network.
    pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
        bs58::decode_pubkey("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

    crate::derive_verify_vaa_shim_consts!();
}

cfg_if::cfg_if! {
    if #[cfg(feature = "testnet")] {
        pub use devnet::*;
    } else if #[cfg(feature = "localnet")] {
        pub use localnet::*;
    }
    // Default to mainnet.
    else {
        pub use mainnet::*;
    }
}
