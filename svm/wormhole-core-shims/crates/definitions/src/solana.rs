/// Wormhole Chain ID identifying Solana's network. This ID is shared between Solana mainnet and
/// devnet.
pub const CHAIN_ID: u16 = 1;

use solana_program::{pubkey, pubkey::Pubkey};

pub const POST_MESSAGE_SHIM_PROGRAM_ID: Pubkey =
    pubkey!("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

cfg_if::cfg_if! {
    if #[cfg(feature = "testnet")] {
        /// Core Bridge program ID on Solana devnet.
        pub const CORE_BRIDGE_PROGRAM_ID: Pubkey = pubkey!("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5");
        pub const CORE_BRIDGE_FEE_COLLECTOR: Pubkey = pubkey!("7s3a1ycs16d6SNDumaRtjcoyMaTDZPavzgsmS3uUZYWX");
        pub const CORE_BRIDGE_CONFIG: Pubkey = pubkey!("6bi4JGDoRwUs9TYBuvoA7dUVyikTJDrJsJU1ew6KVLiu");
    } else if #[cfg(feature = "localnet")] {
        /// Core Bridge program ID on Wormhole's Tilt (dev) network.
        pub const CORE_BRIDGE_PROGRAM_ID: Pubkey = pubkey!("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");
        pub const CORE_BRIDGE_FEE_COLLECTOR: Pubkey = pubkey!("GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs");
        pub const CORE_BRIDGE_CONFIG: Pubkey = pubkey!("FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP");
    }
    // Default to mainnet.
    else {
        /// Core Bridge program ID on Solana mainnet.
        pub const CORE_BRIDGE_PROGRAM_ID: Pubkey = pubkey!("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth");
        pub const CORE_BRIDGE_FEE_COLLECTOR: Pubkey = pubkey!("9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy");
        pub const CORE_BRIDGE_CONFIG: Pubkey = pubkey!("2yVjuQwpsvdsrywzsJJVs9Ueh4zayyo5DYJbBNc3DDpn");
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn core_bridge_fee_collector() {
        let (expected, _) =
            Pubkey::find_program_address(&[b"fee_collector"], &CORE_BRIDGE_PROGRAM_ID);
        assert_eq!(CORE_BRIDGE_FEE_COLLECTOR, expected);
    }

    #[test]
    fn core_bridge_config() {
        let (expected, _) = Pubkey::find_program_address(&[b"Bridge"], &CORE_BRIDGE_PROGRAM_ID);
        assert_eq!(CORE_BRIDGE_CONFIG, expected);
    }
}
