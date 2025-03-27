/// Wormhole Chain ID identifying Solana's network. This ID is shared between Solana mainnet and
/// devnet.
pub const CHAIN_ID: u16 = 1;

use const_crypto::{bs58, ed25519};

use solana_program::pubkey::Pubkey;

cfg_if::cfg_if! {
    if #[cfg(feature = "testnet")] {
        /// Core Bridge program ID on Solana devnet.
        pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32]
            bs58::decode_pubkey("3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5");
        pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =

        /// Post message shim program ID on Solana devnet.
        pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
            bs58::decode_pubkey("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

        /// Verify VAA shim program ID on Solana devnet.
        pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8;32] =
            bs58::decode_pubkey("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");


    } else if #[cfg(feature = "localnet")] {
        /// Core Bridge program ID on Wormhole's Tilt (dev) network.
        pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
            bs58::decode_pubkey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o");

        /// Post message shim program ID on Wormhole's Tilt (dev) network.
        pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
            bs58::decode_pubkey("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

        /// Verify VAA shim program ID on Wormhole's Tilt (dev) network.
        pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8;32] =
            bs58::decode_pubkey("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

    }
    // Default to mainnet.
    else {
        /// Core Bridge program ID on Solana mainnet.
        pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
            bs58::decode_pubkey("worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth");

        /// Post message shim program ID on Solana mainnet.
        pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
            bs58::decode_pubkey("EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX");

        /// Verify VAA shim program ID on Solana mainnet.
        pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8;32] =
            bs58::decode_pubkey("EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at");

    }
}

pub const CORE_BRIDGE_PROGRAM_ID: Pubkey = Pubkey::new_from_array(CORE_BRIDGE_PROGRAM_ID_ARRAY);

pub const CORE_BRIDGE_FEE_COLLECTOR: Pubkey = Pubkey::new_from_array(
    ed25519::derive_program_address(&[crate::FEE_COLLECTOR_SEED], &CORE_BRIDGE_PROGRAM_ID_ARRAY).0,
);

pub const CORE_BRIDGE_CONFIG: Pubkey = Pubkey::new_from_array(
    ed25519::derive_program_address(
        &[crate::CORE_BRIDGE_CONFIG_SEED],
        &CORE_BRIDGE_PROGRAM_ID_ARRAY,
    )
    .0,
);

pub const POST_MESSAGE_SHIM_PROGRAM_ID: Pubkey =
    Pubkey::new_from_array(POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY);

const POST_MESSAGE_SHIM_EVENT_AUTHORITY_PDA: ([u8; 32], u8) = ed25519::derive_program_address(
    &[crate::EVENT_AUTHORITY_SEED],
    &POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY,
);

pub const POST_MESSAGE_SHIM_EVENT_AUTHORITY: Pubkey =
    Pubkey::new_from_array(POST_MESSAGE_SHIM_EVENT_AUTHORITY_PDA.0);

pub const POST_MESSAGE_SHIM_EVENT_AUTHORITY_BUMP: u8 = POST_MESSAGE_SHIM_EVENT_AUTHORITY_PDA.1;

pub const VERIFY_VAA_SHIM_PROGRAM_ID: Pubkey =
    Pubkey::new_from_array(VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY);

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
