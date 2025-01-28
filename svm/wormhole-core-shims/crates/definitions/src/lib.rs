pub mod solana;

// NOTE: Expand this conditional as Wormhole supports more SVM networks.
cfg_if::cfg_if! {
    if #[cfg(feature = "solana")] {
        pub use solana::*;
    }
}

use solana_program::pubkey::Pubkey;

/// Wormhole Core Bridge program's bridge config account seed.
pub const CORE_BRIDGE_CONFIG_SEED: &[u8] = b"Bridge";

/// Wormhole Core Bridge program's fee collector account seed.
pub const FEE_COLLECTOR_SEED: &[u8] = b"fee_collector";

/// Wormhole Core Bridge program's emitter sequence account seed.
pub const EMITTER_SEQUENCE_SEED: &[u8] = b"Sequence";

/// Anchor event CPI's authority seed.
pub const EVENT_AUTHORITY_SEED: &[u8] = b"__event_authority";

/// Derive the Wormhole Core Bridge program's bridge config account address and
/// bump.
pub fn find_core_bridge_config_address(wormhole_program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"Bridge"], wormhole_program_id)
}

/// Derive the Wormhole Core Bridge program's fee collector account address and
/// bump.
pub fn find_fee_collector_address(wormhole_program_id: &Pubkey) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"fee_collector"], wormhole_program_id)
}

/// Derive the Wormhole Core Bridge program's emitter sequence account address
/// and bump.
pub fn find_emitter_sequence_address(
    emitter: &Pubkey,
    wormhole_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"Sequence", emitter.as_ref()], wormhole_program_id)
}

/// Derive the Wormhole Core Bridge program's guardian set address and bump.
pub fn find_guardian_set_address(
    index_be_bytes: [u8; 4],
    wormhole_program_id: &Pubkey,
) -> (Pubkey, u8) {
    Pubkey::find_program_address(&[b"GuardianSet", &index_be_bytes], wormhole_program_id)
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

/// Wormhole Post Message Shim program message event. This message is encoded
/// as instruction data when the Shim program calls itself via CPI.
#[cfg_attr(
    feature = "borsh",
    derive(borsh::BorshDeserialize, borsh::BorshSerialize)
)]
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct MessageEvent {
    pub emitter: Pubkey,
    pub sequence: u64,
    pub submission_time: u32,
}

impl MessageEvent {
    pub const DISCRIMINATOR: [u8; 8] = make_anchor_discriminator(b"event:MessageEvent");
}

/// Trait to encode and decode the SVM finality of a message.
pub trait EncodeFinality: Sized + Copy {
    fn encode(&self) -> u8;

    fn decode(data: u8) -> Option<Self>;
}
