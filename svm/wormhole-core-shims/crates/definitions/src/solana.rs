/// Wormhole Chain ID identifying Solana's network. This ID is shared between Solana mainnet and
/// devnet.
pub const CHAIN_ID: u16 = 1;

use solana_program::{pubkey, pubkey::Pubkey};

pub const POST_MESSAGE_SHIM_PROGRAM_ID: Pubkey =
    pubkey!("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");
pub const POST_MESSAGE_SHIM_EVENT_AUTHORITY: Pubkey =
    pubkey!("HQS31aApX3DDkuXgSpV9XyDUNtFgQ31pUn5BNWHG2PSp");
pub const POST_MESSAGE_SHIM_EVENT_AUTHORITY_BUMP: u8 = 255;

pub const VERIFY_VAA_SHIM_PROGRAM_ID: Pubkey =
    pubkey!("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

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

/// Finality of the message (which is when the Wormhole guardians will attest to
/// this message's observation).
///
/// On Solana, there are only two commitment levels that the Wormhole guardians
/// recognize.
#[cfg_attr(
    feature = "borsh",
    derive(borsh::BorshDeserialize, borsh::BorshSerialize)
)]
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[repr(u8)]
pub enum Finality {
    /// Equivalent to observing after one slot.
    Confirmed,

    /// Equivalent to observing after 32 slots.
    Finalized,
}

impl super::EncodeFinality for Finality {
    fn encode(&self) -> u8 {
        *self as u8
    }

    fn decode(data: u8) -> Option<Self> {
        match data {
            0 => Some(Self::Confirmed),
            1 => Some(Self::Finalized),
            _ => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_core_bridge_fee_collector() {
        let (expected, _) = crate::find_fee_collector_address(&CORE_BRIDGE_PROGRAM_ID);
        assert_eq!(CORE_BRIDGE_FEE_COLLECTOR, expected);
    }

    #[test]
    fn test_core_bridge_config() {
        let (expected, _) = crate::find_core_bridge_config_address(&CORE_BRIDGE_PROGRAM_ID);
        assert_eq!(CORE_BRIDGE_CONFIG, expected);
    }

    #[test]
    fn test_post_message_shim_event_authority() {
        let expected = crate::find_event_authority_address(&POST_MESSAGE_SHIM_PROGRAM_ID);
        assert_eq!(
            (
                POST_MESSAGE_SHIM_EVENT_AUTHORITY,
                POST_MESSAGE_SHIM_EVENT_AUTHORITY_BUMP
            ),
            expected
        );
    }
}
