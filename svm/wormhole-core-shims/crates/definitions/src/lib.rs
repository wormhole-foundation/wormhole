#![deny(dead_code, unused_imports, unused_mut, unused_variables)]
#![doc = include_str!("../README.md")]

#[cfg(feature = "borsh")]
pub mod borsh;
pub mod env;
pub mod solana;
pub mod zero_copy;

pub use env::*;

// We define the constants (chain id + addresses) here.
// - For 'solana', we just re-export the definitions in the solana module.
// - For 'from-env', we pick these up from environment variables and parse (+
// validate) them into the right types via const functions.
//
// Consumers of this crate should mark 'from-env' as the default.
mod defs {
    cfg_if::cfg_if! {
        if #[cfg(feature = "solana")] {
            pub use crate::solana::*;
        } else if #[cfg(feature = "from-env")] {
            #[cfg(any(feature = "testnet"))]
            panic!("The 'testnet' feature is meaningless without the 'solana' feature.");
            #[cfg(any(feature = "localnet"))]
            panic!("The 'localnet' feature is meaningless without the 'solana' feature.");

            use super::*;
            pub const CHAIN_ID: u16 = match u16::from_str_radix(env!("CHAIN_ID"), 10) {
                Ok(c) => c,
                Err(_err) => panic!("CHAIN_ID is not a valid u16")
            };

            pub const CORE_BRIDGE_PROGRAM_ID_ARRAY: [u8; 32] =
            // we use the same variable name as the core contracts
                env_pubkey!("BRIDGE_ADDRESS");

            pub const POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY: [u8; 32] =
                env_pubkey!("POST_MESSAGE_SHIM_PROGRAM_ID");

            pub const VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY: [u8;32] =
                env_pubkey!("VERIFY_VAA_SHIM_PROGRAM_ID");

            derive_consts!();
        }
    }
}

#[allow(unused_imports)]
pub use defs::*;

pub use solana_program::keccak::{Hash, HASH_BYTES};

use solana_program::pubkey::Pubkey;

/// Wormhole Core Bridge program's bridge config account seed.
pub const CORE_BRIDGE_CONFIG_SEED: &[u8] = b"Bridge";

/// Wormhole Core Bridge program's fee collector account seed.
pub const FEE_COLLECTOR_SEED: &[u8] = b"fee_collector";

/// Wormhole Core Bridge program's emitter sequence account seed.
pub const EMITTER_SEQUENCE_SEED: &[u8] = b"Sequence";

/// Wormhole Core Bridge program's guardian set account seed.
pub const GUARDIAN_SET_SEED: &[u8] = b"GuardianSet";

/// Anchor event CPI's authority seed.
pub const EVENT_AUTHORITY_SEED: &[u8] = b"__event_authority";

/// Wormhole Post Message Shim's message event instruction data discriminator.
pub const MESSAGE_EVENT_DISCRIMINATOR: [u8; 8] = make_anchor_discriminator(b"event:MessageEvent");

/// Wormhole Verify VAA Shim's guardian signatures account data discriminator.
pub const GUARDIAN_SIGNATURES_DISCRIMINATOR: [u8; 8] =
    make_anchor_discriminator(b"account:GuardianSignatures");

pub const GUARDIAN_SIGNATURE_LENGTH: usize = 66;
pub const GUARDIAN_PUBKEY_LENGTH: usize = 20;

pub const ANCHOR_EVENT_CPI_SELECTOR: [u8; 8] = u64::to_be_bytes(0xe445a52e51cb9a1d);

/// Convenience method to compute the Keccak-256 digest of Keccak-256 hashed
/// input data. For some messages (like query responses), there is a prefix
/// prepended to this hash before producing the digest. Otherwise the hash is
/// simply hashed again. This digest along with a guardian's signature is used
/// to recover this guardian's pubkey.
///
/// NOTE: In an SVM program, prefer this method or
/// [solana_program::keccak::hashv]. Both of these methods use the
/// `sol_keccak256` syscall under the hood in SVM runtime, which uses minimal
/// compute units.
///
/// # Examples
///
/// A v1 VAA digest can be computed as follows:
/// ```rust
/// use wormhole_svm_definitions::compute_keccak_digest;
///
/// // `vec_body` is the encoded body of the VAA.
/// # let vaa_body = vec![];
/// let digest = compute_keccak_digest(
///     solana_program::keccak::hash(&vaa_body),
///     None, // there is no prefix for V1 messages
/// );
/// ```
///
/// A QueryResponse digest can be computed as follows:
/// ```rust
/// # mod wormhole_query_sdk {
/// #    pub const MESSAGE_PREFIX: &'static [u8] = b"ruh roh";
/// # }
/// use wormhole_query_sdk::MESSAGE_PREFIX;
/// use wormhole_svm_definitions::compute_keccak_digest;
///
/// # let query_response_bytes = vec![];
/// let digest = compute_keccak_digest(
///     solana_program::keccak::hash(&query_response_bytes),
///     Some(MESSAGE_PREFIX)
/// );
#[inline]
pub fn compute_keccak_digest(hashed_data: Hash, prefix: Option<&[u8]>) -> Hash {
    match prefix {
        Some(prefix) => solana_program::keccak::hashv(&[prefix, &hashed_data.0]),
        None => solana_program::keccak::hashv(&[&hashed_data.0]),
    }
}

/// Derive the Wormhole Core Bridge program's bridge config account address and
/// bump.
pub fn find_core_bridge_config_address(wormhole_program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[CORE_BRIDGE_CONFIG_SEED], wormhole_program_id)
}

/// Derive the Wormhole Core Bridge program's fee collector account address and
/// bump.
pub fn find_fee_collector_address(wormhole_program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[FEE_COLLECTOR_SEED], wormhole_program_id)
}

/// Derive the Wormhole Core Bridge program's emitter sequence account address
/// and bump.
pub fn find_emitter_sequence_address(
    emitter: &Pubkey,
    wormhole_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(
        &[EMITTER_SEQUENCE_SEED, emitter.as_ref()],
        wormhole_program_id,
    )
}

/// Derive the Wormhole Core Bridge program's guardian set address and bump.
pub fn find_guardian_set_address(
    index_be_bytes: [u8; 4],
    wormhole_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[GUARDIAN_SET_SEED, &index_be_bytes], wormhole_program_id)
}

/// Derive the Wormhole Post Message Shim program's message account address and
/// bump.
pub fn find_shim_message_address(
    emitter: &Pubkey,
    post_message_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[emitter.as_ref()], post_message_program_id)
}

/// Derive the Anchor event CPI's authority address and bump.
pub fn find_event_authority_address(program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[EVENT_AUTHORITY_SEED], program_id)
}

/// Generate an 8-byte Anchor discriminator from the input data.
pub const fn make_anchor_discriminator(input: &[u8]) -> [u8; 8] {
    let digest = sha2_const_stable::Sha256::new().update(input).finalize();
    let mut trimmed = [0; 8];
    let mut i = 0;

    loop {
        if i >= 8 {
            break;
        }
        trimmed[i] = digest[i];
        i += 1;
    }

    trimmed
}

/// Trait to encode and decode the SVM finality of a message.
/// Different SVM runtimes may have different finality options, so we just
/// provide this conversion trait between u8s.
/// We also define the "standard" finality options ([`std_finality`]) which are
/// the available ones on Solana.
/// If you use this crate with a chain that support different finality modes,
/// either just use `u8`, or define a a custom enum and an impl of this trait.
pub trait EncodeFinality: Sized + Copy {
    /// Encode SVM finality into a byte.
    fn encode(&self) -> u8;

    /// Decode SVM finality from a byte.
    fn decode(data: u8) -> Option<Self>;
}

impl EncodeFinality for u8 {
    fn encode(&self) -> u8 {
        *self
    }

    fn decode(data: u8) -> Option<Self> {
        Some(data)
    }
}

pub mod std_finality {
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
}

#[cfg(feature = "std-finality")]
pub use std_finality::*;

/// Trait that defines an arbitrary discriminator for deserializing data. This
/// discriminator acts as a prefix to identify which kind of data is encoded.
/// For Anchor accounts and events, this discriminator is 8 bytes long.
pub trait DataDiscriminator {
    const DISCRIMINATOR: &'static [u8];
}

/// Derive constants from the defined addresses. We use const functions here so
/// we only need to define the program ids, and PDAs are derived at compile time
/// and made available as consts.
///
/// We expose this as a macro, so that it can be invoked in multiple different scopes.
/// Alternatively, we could just simply define the below constants at the
/// top-level, but then they would only be available to the network that's
/// specified as the feature flag.
///
/// This way, we can derive these constants within the solana network modules,
/// such as 'solana::mainnet' and 'solana::devnet', and have both of them
/// available, even when not compiling for solana mainnet or solana devnet.
/// The network flags just define which of these is available at the top-level.
#[macro_export]
macro_rules! derive_consts {
    () => {
        pub const CORE_BRIDGE_PROGRAM_ID: solana_program::pubkey::Pubkey =
            solana_program::pubkey::Pubkey::new_from_array(CORE_BRIDGE_PROGRAM_ID_ARRAY);

        pub const CORE_BRIDGE_FEE_COLLECTOR_PDA: ([u8; 32], u8) =
            const_crypto::ed25519::derive_program_address(&[crate::FEE_COLLECTOR_SEED], &CORE_BRIDGE_PROGRAM_ID_ARRAY);

        pub const CORE_BRIDGE_FEE_COLLECTOR: solana_program::pubkey::Pubkey =
            solana_program::pubkey::Pubkey::new_from_array(CORE_BRIDGE_FEE_COLLECTOR_PDA.0);

        pub const CORE_BRIDGE_FEE_COLLECTOR_BUMP: u8 = CORE_BRIDGE_FEE_COLLECTOR_PDA.1;

        pub const CORE_BRIDGE_CONFIG_PDA: ([u8; 32], u8) =
            const_crypto::ed25519::derive_program_address(
                &[crate::CORE_BRIDGE_CONFIG_SEED],
                &CORE_BRIDGE_PROGRAM_ID_ARRAY,
            );

        pub const CORE_BRIDGE_CONFIG: solana_program::pubkey::Pubkey =
            solana_program::pubkey::Pubkey::new_from_array(CORE_BRIDGE_CONFIG_PDA.0);

        pub const CORE_BRIDGE_CONFIG_BUMP: u8 = CORE_BRIDGE_CONFIG_PDA.1;

        pub const POST_MESSAGE_SHIM_PROGRAM_ID: solana_program::pubkey::Pubkey =
            solana_program::pubkey::Pubkey::new_from_array(POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY);

        const POST_MESSAGE_SHIM_EVENT_AUTHORITY_PDA: ([u8; 32], u8) =
            const_crypto::ed25519::derive_program_address(
                &[crate::EVENT_AUTHORITY_SEED],
                &POST_MESSAGE_SHIM_PROGRAM_ID_ARRAY,
            );

        pub const POST_MESSAGE_SHIM_EVENT_AUTHORITY: solana_program::pubkey::Pubkey =
            solana_program::pubkey::Pubkey::new_from_array(POST_MESSAGE_SHIM_EVENT_AUTHORITY_PDA.0);

        pub const POST_MESSAGE_SHIM_EVENT_AUTHORITY_BUMP: u8 =
            POST_MESSAGE_SHIM_EVENT_AUTHORITY_PDA.1;

        pub const VERIFY_VAA_SHIM_PROGRAM_ID: solana_program::pubkey::Pubkey =
            solana_program::pubkey::Pubkey::new_from_array(VERIFY_VAA_SHIM_PROGRAM_ID_ARRAY);
    };
}

#[allow(dead_code)]
/// A test to make sure the expected IDs are available.
fn available_ids() {
    let _ = crate::solana::mainnet::CORE_BRIDGE_PROGRAM_ID;
    let _ = crate::solana::devnet::CORE_BRIDGE_PROGRAM_ID;
    let _ = crate::solana::mainnet::CORE_BRIDGE_PROGRAM_ID;
    // defaults to mainnet (for backwards compatibility)
    let _ = crate::solana::CORE_BRIDGE_PROGRAM_ID;
    let _ = crate::CORE_BRIDGE_PROGRAM_ID;
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
