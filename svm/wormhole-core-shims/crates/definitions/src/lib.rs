#![deny(dead_code, unused_imports, unused_mut, unused_variables)]
#![doc = include_str!("../README.md")]

#[cfg(feature = "borsh")]
pub mod borsh;
pub mod solana;
pub mod zero_copy;

// NOTE: Expand this conditional as Wormhole supports more SVM networks.
cfg_if::cfg_if! {
    if #[cfg(feature = "solana")] {
        pub use solana::*;
    }
}

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
pub trait EncodeFinality: Sized + Copy {
    /// Encode SVM finality into a byte.
    fn encode(&self) -> u8;

    /// Decode SVM finality from a byte.
    fn decode(data: u8) -> Option<Self>;
}

/// Trait that defines an arbitrary discriminator for deserializing data. This
/// discriminator acts as a prefix to identify which kind of data is encoded.
/// For Anchor accounts and events, this discriminator is 8 bytes long.
pub trait DataDiscriminator {
    const DISCRIMINATOR: &'static [u8];
}
